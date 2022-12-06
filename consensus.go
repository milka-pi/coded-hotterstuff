package hotstuff

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sort"
	"time"

	// "math/rand"

	"github.com/dshulyak/go-hotstuff/types"
	"go.uber.org/zap"

	"github.com/vivint/infectious"
)

type Signer interface {
	Sign([]byte, []byte) []byte
}

type Verifier interface {
	VerifyAggregated([]byte, *types.AggregatedSignature) bool
	Verify(uint64, []byte, []byte) bool
	Merge(*types.AggregatedSignature, uint64, []byte)
}

type AllVotes struct{
	votes *Votes
	pendingVotes []*types.Vote
}


func newConsensus(
	logger *zap.Logger,
	store *BlockStore,
	signer Signer,
	verifier Verifier,
	id uint64,
	replicas []uint64,
) *consensus {
	logger = logger.Named(fmt.Sprintf("replica=%d", id))
	view, err := store.GetView()
	if err != nil {
		logger.Fatal("failed to load view", zap.Error(err))
	}
	voted, err := store.GetVoted()
	if err != nil {
		logger.Fatal("failed to load voted view", zap.Error(err))
	}
	prepare, err := store.GetTagHeader(PrepareTag)
	if err != nil {
		logger.Fatal("failed to load prepare header", zap.Error(err))
	}
	locked, err := store.GetTagHeader(LockedTag)
	if err != nil {
		logger.Fatal("failed to load locked header", zap.Error(err))
	}
	commit, err := store.GetTagHeader(DecideTag)
	if err != nil {
		logger.Fatal("failed to load commit header", zap.Error(err))
	}
	prepareCert, err := store.GetTagCert(PrepareTag)
	if err != nil {
		logger.Fatal("failed to load prepare certificate", zap.Error(err))
	}

	// new: calculate erasure code configuration
	maxNumFaulty := (len(replicas) - 1) / 3

	return &consensus{
		log:         logger,
		store:       store,
		signer:      signer,
		verifier:    verifier,
		id:          id,
		replicas:    replicas, // array of public keys
		timeouts:    NewTimeouts(verifier, 2*len(replicas)/3+1),
		votes:       NewVotes(verifier, 2*len(replicas)/3+1),
		hashToVotes: make(map[uint64]*AllVotes), // new
		prepare:     prepare,
		locked:      locked,
		commit:      commit,
		prepareCert: prepareCert,
		view:        view,
		voted:       voted,
		// new
		randomIDToChunks: make(map[uint64]*codedChunks),
		outdatedRandomIDs: make(map[uint64]struct{}),

		errCodeConfig: erasureCodeConfig{
			required: len(replicas) - 1 - maxNumFaulty,
			// required: 2,
			// oversampled: ,
			total: len(replicas) - 1, 
			// total: 3,
		},
		// new
		// proposalsToRevisit: make(map[*[]byte]*types.Proposal), 
		proposalsToRevisit: []*types.Proposal{},
	}
}

type consensus struct {
	log, vlog *zap.Logger

	store *BlockStore

	// signer and verifier are done this way to remove all public key references from consensus logic
	// uint64 refers to validator's public key in sorted set of validators keys
	signer   Signer
	verifier Verifier

	// timeout controlled by consensus. doubles the number of expected ticks for every gap since last block
	// was signed by quorum.
	ticks, timeout int

	// TODO maybe it wasn't great idea to remove public key from consensus
	id       uint64
	replicas []uint64 // all replicas, including current

	votes    *Votes
	hashToVotes map[uint64]*AllVotes // new
	timeouts *Timeouts

	view                    uint64 // current view.
	voted                   uint64 // last voted view. must be persisted to ensure no accidental double vote on the same view
	prepare, locked, commit *types.Header
	prepareCert             *types.Certificate // prepareCert always updated to highest known certificate
	// TODO should it be persisted? can it help with synchronization after restart?
	timeoutCert *types.TimeoutCertificate

	// maybe instead Progress should be returned by each public method
	Progress Progress

	waitingData bool // if waitingData true then node must create a proposal when it receives data

	//----------------------------------------------------------------------------------------------
	// new field: for Coded Broadcast
	// DONE: cannot store all chunks, when should I delete? after confirming a block? after decoding. OK
	randomIDToChunks map[uint64]*codedChunks
	outdatedRandomIDs map[uint64]struct{}

	errCodeConfig erasureCodeConfig

	// TODO 6/6: new map to track blocks which could not be matched with parent block
	proposalsToRevisit []*types.Proposal
}

func (c *consensus) Tick() {
	c.ticks++
	if c.ticks == c.timeout {
		c.onTimeout()
	}
}

type erasureCodeConfig struct{
	required int
	// oversampled int
	total int
}

// const(
// 	required = 2
// 	oversampled = 1
// 	total = 3
// )

type codedChunks struct{
	randomID uint64
	originalChunk *types.Proposal
	chunksList []infectious.Share
}


func (c *consensus) getDataShares(data_bytes []byte) []infectious.Share {
	// Create a *FEC, which will require required pieces for reconstruction at
	// minimum, and generate total total pieces.
	f, err := infectious.NewFEC(c.errCodeConfig.required , c.errCodeConfig.total)
	if err != nil {
		panic(err)
	}

	// Prepare to receive the shares of encoded data.
	shares := make([]infectious.Share, c.errCodeConfig.total)
	output := func(s infectious.Share) {
		// the memory in s gets reused, so we need to make a deep copy
		shares[s.Number] = s.DeepCopy()
	}

	// prepend length in un-encoded data and pad to make it multiple of `required`
	// slice of slice of bytes ([][] bytes): encoded data
	data_bytes = augmentByteArrayWithLength(data_bytes)
	data_bytes = pad(data_bytes, c.errCodeConfig.required)
	if len(data_bytes) % c.errCodeConfig.required != 0 {
		panic("Error: impoper padding!")
	}

	// encode data
	err = f.Encode(data_bytes, output)
	if err != nil {
		panic(err)
	}

	return shares
}

// leaderBit supposed to be 0/1
func prependShare(share infectious.Share, leaderBit byte, randomID []byte) infectious.Share{
	numberFieldBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(numberFieldBytes[0 : 4], uint32(share.Number))
	
	token := append([]byte{leaderBit}, randomID...)
	token = append(token, numberFieldBytes...)
	share.Data = append(token, share.Data...)
	return share
}


func (c *consensus) Send(state, root []byte, data []byte) {
	if c.waitingData {
		c.waitingData = false
		header := &types.Header{
			View:       c.view,
			ParentView: c.prepare.View,
			Parent:     c.prepare.Hash(),
			DataRoot:   root,
			StateRoot:  state,
		}
		proposal := &types.Proposal{
			Header:     header,
			Data:       data,
			ParentCert: c.prepareCert,
			Timeout:    c.timeoutCert,
			Sig:        c.signer.Sign(nil, header.Hash()),
		}
		c.vlog.Debug("sending proposal",
			zap.Binary("hash", header.Hash()),
			zap.Binary("parent", header.Parent),
			zap.Uint64("signer", c.id))


		// Round 1 (Leader): Encode proposal into chunks and send one chunk per replica

		// DONE: encode proposal into chunks and sendMsg with specific recipient

		// type data []byte instead of Transaction type -- need to change types.proto -- how to compile types.proto?

		// data_bytes := make([]byte, 1000)
		// rand.Read(data_bytes)
		data_bytes := data


		// DONE 6/3: generate random id outside for loop
		shares := c.getDataShares(data_bytes)
		// we now have total shares.
		// generate random ID number
		randomID := make([]byte, 8)
		rand.Read(randomID)
		for i, share := range shares {
			// fmt.Println(share.Number, share.Data)
			share = prependShare(share, byte(1), randomID)
			// fmt.Println("prepended share: ", share.Data)
			shares[i] = share			
		}

		// format : [0/1 leader bit: 1 byte] | [random id: 8 bytes] | [number Field: 4 bytes] | [share : custom?]


		// prepend shares[i].Data with random id (32 bytes)
		// also prepend with 0/1 leader or not? (original/forwarded)

		// proposals := make([]types.Proposal, total)
		
		// sharesWrapped: map: id -> data share, for all ids except for the leader id 
		wrappedShares := make(map[int]*types.Proposal, c.errCodeConfig.total)
		j := 0
		for i := 0; i < len(c.replicas); i++ {
			if uint64(i) != c.id {

				wrappedShares[i] = &types.Proposal{
					Header:     proposal.GetHeader(),
					// DONE: replace with 'Data: shares[i].Data'
					Data:       shares[j].Data,
					ParentCert: proposal.GetParentCert(),
					Timeout:    proposal.GetTimeout(),
					Sig:        proposal.GetSig(),
				}
				j += 1
			}
		}

		// DONE 6/3: prepend data bytes array also with Number field.
		// try this on simple reed-solomon

		for peerID, wrappedShare := range wrappedShares  {
			// FIXED: send all 3f shares. No share associated with the leader.
			c.sendMsg(NewProposalMsg(wrappedShare), uint64(peerID))
		}

		// [SOS] FIXED 7/18: Leader needs to invoke onProposal() directly (for himself!) to make progress and vote.. 
		c.onProposal(proposal) 

		// c.sendMsg(NewProposalMsg(proposal))
	}
}


// - borrow functionality from onVote etc. to count up to 2f+1 received chunks

func (c *consensus) broadcastCodedChunk(msg *types.Proposal){
	// Round 2: broadcast received chunk
	for i := 0; i < len(c.replicas); i++ {
		// FIXED: broadcast chunk to all nodes except myself (and the leader?)
		if uint64(i) != c.id && uint64(i) != c.getLeader(msg.Header.View) {
			log := c.vlog.With(
				zap.String("msg", "chunk"),
				zap.Uint64("header view", msg.Header.View),
				zap.Uint64("prepare view", c.prepare.View),
				zap.Binary("hash", msg.Header.Hash()),
				zap.Binary("parent", msg.Header.Parent))
			// DONE: What is the correct way to index into the replicas?
			c.sendMsg(NewProposalMsg(msg), uint64(i))
			log.Debug("replica " + fmt.Sprint(c.id) + " is forwarding chunk to replica " + fmt.Sprint(i))

			if uint64(i) == c.getLeader(msg.Header.View) {
				log := c.vlog.With(
					zap.String("msg", "chunk"),
					zap.Uint64("header view", msg.Header.View),
					zap.Uint64("prepare view", c.prepare.View),
					zap.Binary("hash", msg.Header.Hash()),
					zap.Binary("parent", msg.Header.Parent))
			
				log.Debug("forwarding chunk to leader " + fmt.Sprint(i))

			}
		}
	}
}


// use chunks/shares to decode and reconstruct the proposal
func (c *consensus) decodeProposal(randomID uint64) ([]byte, error) {

	// shares should be trimmed at this point
	shares := c.randomIDToChunks[randomID].chunksList

	f, err := infectious.NewFEC(c.errCodeConfig.required, c.errCodeConfig.total)
	if err != nil {
		panic(err)
	}

	result, err := f.Decode(nil, shares)
	if err != nil {
		return []byte{}, err
		// panic(err)
	}

	// we have the original data!
	// fmt.Printf("got: %#v\n", string(result))
	padded_bytes := result

	original_bytes := getBytesFromAugmented(padded_bytes)
	// original_msg := string(original_bytes)
	// fmt.Println("Original data bytes:", original_msg)
	return original_bytes, nil
}

func (c *consensus) Step(msg *types.Message) {
	switch m := msg.GetType().(type) {

	case *types.Message_Proposal:
		// change logic

		if c.id == c.getLeader(m.Proposal.Header.View) {
			fmt.Println("LEADER ENTERED Step -> types.Message_Proposal !")
		} else {

			// check sender? if leader, then call onCodedChunk()
			// store received proposals in dictionary indexed by viewNumber? -- indexed by random id

			// DONE: fetch view number of proposal
			// answer: will use random id

			shareData := m.Proposal.Data
			// fmt.Println(c.id, "shareData: ", shareData)

			// strip share from leader bit random id
			leaderBit := int(shareData[0])
			randomID := uint64(binary.BigEndian.Uint32(shareData[1:9]))
			shareNumberField := int(binary.BigEndian.Uint32(shareData[9:13]))
			// share.Data = share.Data[9:]


			// check if leading bit is 0/1 to decide whether to broadcast
			// also save other fields (header, ..)
			if leaderBit == 1 {
				// fmt.Println("replica", c.id, "received coded chunk from leader", c.getLeader(m.Proposal.Header.View))
				modifiedShareData := append([]byte{0}, shareData[1:]...)
				modifiedProposal := types.Proposal{
										Header:     m.Proposal.GetHeader(),
										// might need fixing
										Data:       modifiedShareData,
										ParentCert: m.Proposal.GetParentCert(),
										Timeout:    m.Proposal.GetTimeout(),
										Sig:        m.Proposal.GetSig(),
									}
				c.broadcastCodedChunk(&modifiedProposal)
			}

			// TODO: check if header and other info is the same as the one sent by the leader
			// for now: assume everyone is honest
			// ANSWER: can filter out when using Merkle trees to authenticate
			trimmedShare := infectious.Share{Number: shareNumberField,
											Data: shareData[13:]}

			_, ok1 := c.outdatedRandomIDs[randomID]
			// Update chunksList only if randomID is not outdated
			if !ok1 {
				if chunks, ok2 := c.randomIDToChunks[randomID]; ok2 {
					// already received some chunk for this random id
		
					// DONE: fix Data type issue 
					chunks.chunksList = append(chunks.chunksList, trimmedShare)			
				} else {
					// this is the first chunk received for this random id
					c.randomIDToChunks[randomID] = &codedChunks{
												randomID: randomID,
												originalChunk: m.Proposal,
												// TODO: only if leader (okay for now)
												chunksList: []infectious.Share{trimmedShare},			
											}
				}
				// fmt.Println("replica", c.id, "has received", len(c.randomIDToChunks[randomID].chunksList), "chunks")


				// check if have received enough chunks. If so, attempt decoding the proposal
				if len(c.randomIDToChunks[randomID].chunksList) >= c.errCodeConfig.required {
					// TODO 6/3: add error handling --> don't call onProposal
					proposalData, errDecode := c.decodeProposal(randomID)
					if errDecode != nil {
						c.vlog.Debug("Could not decode original proposal")
					} else {
						originalChunk := c.randomIDToChunks[randomID].originalChunk
						fullProposal := types.Proposal{
											Header:     originalChunk.GetHeader(),
											// might need fixing
											Data:       proposalData,
											ParentCert: originalChunk.GetParentCert(),
											Timeout:    originalChunk.GetTimeout(),
											Sig:        originalChunk.GetSig(),
										}
						// c.vlog.Debug("Decoded original proposal" + fmt.Sprint(fullProposal.Header.Hash()))
						c.vlog.Debug("Decoded original proposal, time: " + time.Now().String() + " header hash: " + fmt.Sprint(fullProposal.Header.Hash()))
						c.onProposal(&fullProposal)
						
						// discard saved chunks
						delete(c.randomIDToChunks, randomID)
						c.outdatedRandomIDs[randomID] = struct{}{}

						for i:=0; i<= len(c.proposalsToRevisit)-1; i++ {
							if bytes.Compare(c.proposalsToRevisit[i].Header.Hash(), m.Proposal.Header.Hash()) == 0 {
								c.proposalsToRevisit = removeIndex(c.proposalsToRevisit, i)
							}
						}

						// DONE? 6/3: implementation level issue: create outdated randomIDs map. 
					}
				}
			}
		}


		// c.onProposal(m.Proposal)

	case *types.Message_Vote:
		c.onVote(m.Vote)
	case *types.Message_Newview:
		c.onNewView(m.Newview)
	case *types.Message_Sync:
		c.onSync(m.Sync)
	case *types.Message_Syncreq:
		c.onSyncReq(m.Syncreq)
	}
	
}

func (c *consensus) sendMsg(msg *types.Message, ids ...uint64) {
	c.Progress.AddMessage(msg, ids...)
	if ids == nil {
		c.Step(msg)
	}
	for _, id := range ids {
		if c.id == id {
			c.Step(msg)
		}
	}
}

func (c *consensus) Start() {
	c.nextRound(false)
}

func (c *consensus) onTimeout() {
	c.view++
	err := c.store.SaveView(c.view)
	if err != nil {
		c.vlog.Fatal("can't update view", zap.Error(err))
	}
	c.nextRound(true)
}

func (c *consensus) resetTimeout() {
	c.ticks = 0
	c.timeout = c.newTimeout()
}

func (c *consensus) nextRound(timedout bool) {
	c.vlog = c.log.With(zap.Uint64("CURRENT VIEW", c.view))

	c.resetTimeout()
	c.waitingData = false
	// c.votes.Reset()

	c.vlog.Debug("entered new view",
		zap.Int("view timeout", c.timeout),
		zap.Bool("timedout", timedout))

	// release timeout certificate once it is useless
	if c.timeoutCert != nil && c.timeoutCert.View < c.view-1 {
		c.timeoutCert = nil
	}

	// timeouts will be collected for two rounds, before leadership and the round when replica is a leader
	if c.id == c.getLeader(c.view-1) {
		c.timeouts.Reset()
	}
	// if this replica is a leader for next view start collecting new view messages
	if c.id == c.getLeader(c.view+1) {
		c.vlog.Debug("replica is a leader for next view. collecting new-view messages")

		c.timeouts.Reset()
		c.timeouts.Start(c.view)
	}

	if timedout && c.view-1 > c.voted {
		c.sendNewView()
	} else if c.id == c.getLeader(c.view) {
		c.vlog.Debug("replica is a leader for this view. waiting for data")
		c.waitingData = true
		c.Progress.WaitingData = true
	}
}

func removeIndex(s []*types.Proposal, index int) []*types.Proposal {
	ret := make([]*types.Proposal, 0)
	ret = append(ret, s[:index]...)
	return append(ret, s[index+1:]...)
}

// simple check that should pass on proposalsToRevisit
// proposals should be sorted according to view number
func (c *consensus) checkProposalsToRevisit() bool {
	for i:=0; i<=len(c.proposalsToRevisit)-2; i++ {
		if c.proposalsToRevisit[i].Header.View > c.proposalsToRevisit[i+1].Header.View {
			return false
		}
	}
	return true
}


func (c *consensus) onProposal(msg *types.Proposal) {
	log := c.vlog.With(
		zap.String("msg", "proposal"),
		zap.Uint64("header view", msg.Header.View),
		zap.Uint64("prepare view", c.prepare.View),
		zap.Binary("hash", msg.Header.Hash()),
		zap.Binary("parent", msg.Header.Parent))

	log.Debug("received proposal")

	if msg.ParentCert == nil {
		return
	}

	if bytes.Compare(msg.Header.Parent, msg.ParentCert.Block) != 0 {
		return
	}

	if !c.verifier.VerifyAggregated(msg.Header.Parent, msg.ParentCert.Sig) {
		log.Debug("certificate is invalid")
		return
	}

	// fmt.Println("------------ msg.Header.Parent: ", msg.Header.Parent, "-------------------")	
	parent, err := c.store.GetHeader(msg.Header.Parent)
	if err != nil {
		log.Debug("header for parent is not found", zap.Error(err))
		// TODO if certified block is not found we need to sync with another node
		c.Progress.AddNotFound(msg.Header.ParentView, msg.Header.Parent)

		// add an attribute to consensus - what msg caused the sync event
		// call onProposal on msg that caused this
		// reason: proposal gets dropped, you will not be able to participate in future proposals

		// save any type of message that cannot be processed at the moment
		c.proposalsToRevisit = append(c.proposalsToRevisit, msg)

		// sort array if not sorted (may be redundant)
		if c.checkProposalsToRevisit() == false {
			sort.Slice(c.proposalsToRevisit, func(i, j int) bool {
				return c.proposalsToRevisit[i].Header.View < c.proposalsToRevisit[j].Header.View
			  })
		}

		// DONE?? 6/10 - Optimization: if the proposal's parent is already in `proposalsToRevisit`, do not make syncRequest 
		// 								but def append to `proposalsToRevisit`.
		// idea: only check most recent one for parent?

		// TODO 6/13: check for error due to aliasing (need to deepcopy)

		if len(c.proposalsToRevisit) > 0 {
			// check if parent is already in `proposalsToRevisit`. If not, issue sync request.
			if bytes.Compare(msg.Header.Parent, c.proposalsToRevisit[len(c.proposalsToRevisit) - 1].Header.Hash()) != 1 {
				// 6/8 Question: how to get most recent finalized block?  How to use block store? 
				// 6/10 answer: just use c.commit -- contains most recently commited header

				mostRecentHeader := c.commit
				syncReq := types.SyncRequest{
					From: mostRecentHeader,
					Limit: uint64(0),
					Id: c.id,
				}
				c.sendMsg(NewSyncReqMsg(&syncReq), c.getLeader(c.view))
				c.vlog.Debug("made sync request for view " + fmt.Sprint(c.view) + ", missing parent: " + fmt.Sprint(msg.Header.Parent))
			} else {
				c.vlog.Debug("parent is already in proposalsToRevisit")
			}
		}
		return

	}

	// 6/13: remove proposal from `proposalsToRevisit` if needed, since the parent check passed
	for i:=0; i<= len(c.proposalsToRevisit)-1; i++ {
		if bytes.Compare(c.proposalsToRevisit[i].Header.Hash(), msg.Header.Hash()) == 0 {
			c.proposalsToRevisit = removeIndex(c.proposalsToRevisit, i)
		}
	}

	//--------------------------------------------------------------------------------------------------------------
	// old code below

	if msg.Timeout != nil {
		if !c.verifier.VerifyAggregated(HashSum(EncodeUint64(msg.Header.View-1)), msg.Timeout.Sig) {
			return
		}
	}

	leader := c.getLeader(msg.Header.View)
	if !c.verifier.Verify(leader, msg.Header.Hash(), msg.Sig) {
		log.Debug("proposal is not signed correctly", zap.Uint64("signer", leader))
		return
	}

	c.updatePrepare(parent, msg.ParentCert)
	c.syncView(parent, msg.ParentCert, msg.Timeout)
	c.update(parent, msg.ParentCert)

	if msg.Header.View != c.view {
		log.Debug("proposal view doesn't match local view")
		return
	}

	// IK question 6/6: is this an application-specific todo? yes

	// TODO after basic validation, state machine needs to validate Data included in the proposal
	// add Data to Progress and wait for a validation from state machine
	// everything after this comment should be done after receiving ack from app state machine
	c.persistProposal(msg)
	if msg.Header.View > c.voted && c.safeNode(msg.Header, msg.ParentCert) {
		log.Debug("proposal is safe to vote on")
		c.sendVote(msg.Header)
	}
}

func (c *consensus) persistProposal(msg *types.Proposal) {
	hash := msg.Header.Hash()

	// IK question 6/6: Is this similar to the above todo

	// TODO header and data for proposal must be tracked, as they can be pruned from store
	// e.g. add a separate bucket for tracking non-finalized blocks
	// remove from that bucket when block is commited
	// in background run a thread that will clear blocks that are in that bucket with a height <= commited height

	err := c.store.SaveHeader(msg.Header)
	if err != nil {
		c.vlog.Fatal("can't save a header", zap.Error(err))
	}
	err = c.store.SaveData(hash, msg.Data)
	if err != nil {
		c.vlog.Fatal("can't save a block data", zap.Error(err))
	}
	err = c.store.SaveCertificate(msg.ParentCert)
	if err != nil {
		c.vlog.Fatal("can't save a certificate", zap.Error(err))
	}
}

func (c *consensus) sendNewView() {
	// send new-view to the leader of this round.
	leader := c.getLeader(c.view)
	nview := &types.NewView{
		Voter: c.id,
		View:  c.view - 1,
		Cert:  c.prepareCert,
		// TODO prehash encoded uint
		Sig: c.signer.Sign(nil, HashSum(EncodeUint64(c.view-1))),
	}
	c.voted = c.view - 1
	err := c.store.SaveVoted(c.voted)
	if err != nil {
		c.vlog.Fatal("can't save voted", zap.Error(err))
	}

	c.vlog.Debug("sending new-view", zap.Uint64("previous view", nview.View), zap.Uint64("leader", leader))
	c.sendMsg(NewViewMsg(nview), leader)
}

func (c *consensus) sendVote(header *types.Header) {
	hash := header.Hash()

	c.voted = header.View
	err := c.store.SaveVoted(c.voted)
	if err != nil {
		c.vlog.Fatal("can't save voted", zap.Error(err))
	}

	// 	DONE: use string(hash) to be safer, and then string(vote.Block) in onVote method

	// new: need to check membership
	if _, ok := c.hashToVotes[header.View]; !ok {
		// fmt.Println("Node ", c.id, " received proposal before votes, hash is ", hash, " and key is, ", string(hash))
		// c.vlog.Debug("Node " + fmt.Sprint(c.id) + " received proposal before votes, key is, " + string(hash))
		c.hashToVotes[header.View] = &AllVotes{
			votes: NewVotes(c.verifier, 2*len(c.replicas)/3+1),
			pendingVotes: []*types.Vote{},
		}
	}

	keys := make([]uint64, 0, len(c.hashToVotes))
	for k := range c.hashToVotes {
		keys = append(keys, k)
	}
	// fmt.Println("Node ", c.id, " allVotes keys: ", keys)

	c.hashToVotes[header.View].votes.Start(header)

	// TODO: take care of any pending votes
	for _, vote := range c.hashToVotes[header.View].pendingVotes {
		c.vlog.Debug("Taking care of pending vote " + fmt.Sprint(vote.View))
		c.onVote(vote)
	}
	c.hashToVotes[header.View].pendingVotes = []*types.Vote{}

	vote := &types.Vote{
		Block: hash,
		View:  header.View,
		Voter: c.id,
		Sig:   c.signer.Sign(nil, hash),
	}

	c.sendMsg(NewVoteMsg(vote), c.getLeader(vote.View+1))
}

func (c *consensus) update(parent *types.Header, cert *types.Certificate) {
	// TODO if any node is missing in the chain we should switch to sync mode
	gparent, err := c.store.GetHeader(parent.Parent)
	if err != nil {
		c.vlog.Debug("could not find gparent header", zap.Uint64("view", parent.View), zap.Binary("hash", parent.Hash()))
		return
	}
	ggparent, err := c.store.GetHeader(gparent.Parent)
	if err != nil {
		c.vlog.Debug("could not find ggparent header", zap.Uint64("view", gparent.View), zap.Binary("hash", gparent.Hash()))
		return
	}

	// 2-chain locked, gaps are allowed
	if gparent.View > c.locked.View {
		c.vlog.Debug("new block locked", zap.Uint64("view", gparent.View), zap.Binary("hash", gparent.Hash()))
		c.locked = gparent
		err := c.store.SetTag(LockedTag, gparent.Hash())
		if err != nil {
			c.vlog.Fatal("can't set locked tag", zap.Error(err))
		}
	}
	// 3-chain commited must be without gaps
	if parent.View-gparent.View == 1 && gparent.View-ggparent.View == 1 && ggparent.View > c.commit.View {
		c.vlog.Info("new block commited", zap.Uint64("view", ggparent.View), zap.Binary("hash", ggparent.Hash()))
		c.commit = ggparent
		err := c.store.SetTag(DecideTag, ggparent.Hash())
		if err != nil {
			c.vlog.Fatal("can't set decided tag", zap.Error(err))
		}
		c.Progress.AddHeader(c.commit, true)
	}
}

func (c *consensus) updatePrepare(header *types.Header, cert *types.Certificate) {
	if header.View > c.prepare.View {
		c.vlog.Debug("new block certified",
			zap.Uint64("view", header.View),
			zap.Binary("hash", header.Hash()),
		)
		c.prepare = header
		c.prepareCert = cert
		err := c.store.SetTag(PrepareTag, header.Hash())
		if err != nil {
			c.vlog.Fatal("failed to set prepare tag", zap.Error(err))
		}
		c.Progress.AddHeader(header, false)
	}
}

func (c *consensus) syncView(header *types.Header, cert *types.Certificate, tcert *types.TimeoutCertificate) {
	if tcert == nil && header.View >= c.view {
		c.view = header.View + 1
		if err := c.store.SaveView(c.view); err != nil {
			c.vlog.Fatal("failed to store view", zap.Error(err))
		}
		c.nextRound(false)
	} else if tcert != nil && tcert.View >= c.view {
		c.view = tcert.View + 1
		if err := c.store.SaveView(c.view); err != nil {
			c.vlog.Fatal("failed to store view", zap.Error(err))
		}
		c.nextRound(false)
	}
}

func (c *consensus) getBlocksToReturn(from *types.Header) []*types.Block {
	blocksToReturn := []*types.Block{}
	// backtrack from c.commit to from
	lastFinHeader := c.commit
	lastFinBlock, err := c.store.GetBlock(lastFinHeader.Hash())
	if err != nil {
		blocksToReturn = append([]*types.Block{lastFinBlock}, blocksToReturn...)
	}
	for {
		lastFinHeader, err1 := c.store.GetHeader(lastFinHeader.GetParent())
		lastFinBlock, err2 := c.store.GetBlock(lastFinHeader.GetParent())

		if err1 != nil && err2 != nil && bytes.Compare(from.Hash(), lastFinHeader.Hash()) != 0 {
			blocksToReturn = append([]*types.Block{lastFinBlock}, blocksToReturn...)
		} else {
			break
		}
		// is this correct? should I check view number?
	}
	return blocksToReturn
}

// 6/9 NEW
func (c *consensus) onSyncReq(syncReq *types.SyncRequest) {
	from := syncReq.GetFrom()
	// limit := syncReq.GetLimit()
	id := syncReq.GetId()
	c.vlog.Debug("received sync request from node " + fmt.Sprint(id) + " for views since " + fmt.Sprint(syncReq.From.View))
	blocksToReturn := c.getBlocksToReturn(from)
	c.vlog.Debug("sending sync message with this many blocks: " + fmt.Sprint(len(blocksToReturn)))
	c.sendMsg(NewSyncMsg(blocksToReturn...), id)
	
	// DONE 6/10: add sender id/index as an extra syncRequest field.
	// answer: pass c.id
}

func (c *consensus) onSync(sync *types.Sync) {

	c.vlog.Debug("received sync message with this many blocks: " + fmt.Sprint(len(sync.Blocks)))
	for _, block := range sync.Blocks {
		if !c.syncBlock(block) {
			c.vlog.Debug("failed syncBlock, header: " + fmt.Sprint(block.Header))
			return
		}
	}
	// call onProposal() on all proposals that caused you to sync
	// optimization: don't ask for everything, check chain of proposals that caused you to sync

	for i:=0; i<= len(c.proposalsToRevisit)-1; i++ {
		// c.onProposal(c.proposalsToRevisit[i])
		fmt.Println("node ", c.id, " revisiting coded chunks")
		c.Step(NewProposalMsg(c.proposalsToRevisit[i]))
	}
	// TODO 6/13: remove proposal from array when onProposal succeeds
	// Idea: delete if passed the if statement inside `onProposal()`
}

// syncBlock returns false if block is invalid.
func (c *consensus) syncBlock(block *types.Block) bool {
	if block.Header == nil || block.Cert == nil || block.Data == nil {
		return false
	}
	if block.Header.View <= c.commit.View {
		return true
	}
	if !c.verifier.VerifyAggregated(block.Header.Hash(), block.Cert.Sig) {
		return false
	}
	log := c.log.With(
		zap.Uint64("block view", block.Header.View),
		zap.Binary("block hash", block.Header.Hash()),
	)
	log.Debug("syncing block")

	if err := c.store.SaveBlock(block); err != nil {
		log.Fatal("can't save block")
	}

	c.updatePrepare(block.Header, block.Cert)
	c.update(block.Header, block.Cert)
	return true
}

func (c *consensus) safeNode(header *types.Header, cert *types.Certificate) bool {
	// is safe if header extends locked or cert height is higher then the lock height
	parent, err := c.store.GetHeader(header.Parent)
	if err != nil {
		// TODO this could be out of order, and require synchronization
		return false
	}
	if bytes.Compare(header.Parent, c.locked.Hash()) == 0 {
		return true
	}
	if bytes.Compare(header.Parent, c.prepare.Hash()) == 0 {
		return true
	}

	// safe to vote since majority overwrote a lock
	// i think, only possible if leader didn't wait for 2f+1 new views from a prev rounds
	if parent.View > c.locked.View {
		return true
	}
	return false
}

func (c *consensus) getLeader(view uint64) uint64 {
	// TODO change to hash(view) % replicas
	return c.replicas[view%uint64(len(c.replicas))]
}

func (c *consensus) newTimeout() int {
	// double for each gap between current round and last round where quorum was collected
	return int(1 << (c.view - c.prepare.View))
}

func (c *consensus) onVote(vote *types.Vote) {
	// next leader is reponsible for aggregating votes from the previous round
	log := c.vlog.With(
		zap.String("msg", "vote"),
		zap.Uint64("voter", vote.Voter),
		zap.Uint64("view", c.view),
		zap.Binary("hash", vote.Block),
	)
	if c.id != c.getLeader(vote.View+1) {
		return
	}
	log.Debug("received vote")

	// New: if received vote before proposal, need to to save vote in pending votes (part I)
	if _, ok := c.hashToVotes[vote.View]; !ok {
		fmt.Println("Node ", c.id, " received vote before proposal, saving vote in pending votes, hash is ", vote.Block, " and string is, ", vote.View)
		c.vlog.Debug("Received vote before proposal, key is, " + fmt.Sprint(vote.View))
		c.hashToVotes[vote.View] = &AllVotes{
			votes: NewVotes(c.verifier, 2*len(c.replicas)/3+1),
			pendingVotes: []*types.Vote{},
		}
	}

	keys := make([]uint64, 0, len(c.hashToVotes))
	for k := range c.hashToVotes {
		keys = append(keys, k)
	}
	// fmt.Println("Node ", c.id, " allVotes keys: ", keys)

	allVotes := c.hashToVotes[vote.View]

	// New: if received vote before proposal, need to to save vote in pending votes (part II)
	if allVotes.votes.Cert == nil {
		// received vote before proposal from a valid leader
		allVotes.pendingVotes = append(allVotes.pendingVotes, vote)
		return
	}

	// new
	if !allVotes.votes.Collect(vote) {
		// do nothing if there is no majority
		return
	}

	// update leaf and certificate to collected and prepare for proposal
	if err := c.store.SaveCertificate(allVotes.votes.Cert); err != nil {
		c.log.Fatal("can't save new certificate",
			zap.Binary("cert for block", allVotes.votes.Cert.Block),
		)
	}
	c.updatePrepare(allVotes.votes.Header, allVotes.votes.Cert)
	c.syncView(allVotes.votes.Header, allVotes.votes.Cert, nil)
	c.nextRound(false)
}

func (c *consensus) onNewView(msg *types.NewView) {
	if c.id != c.getLeader(msg.View+1) {
		return
	}
	c.vlog.Debug("received new-view",
		zap.Uint64("local view", c.view),
		zap.Uint64("timedout view", msg.View),
		zap.Binary("certificate for", msg.Cert.Block),
		zap.Uint64("from", msg.Voter),
	)

	header, err := c.store.GetHeader(msg.Cert.Block)
	if err != nil {
		c.vlog.Debug("can't find block", zap.Binary("block", msg.Cert.Block))
		return
	}

	c.updatePrepare(header, msg.Cert)
	if !c.timeouts.Collect(msg) {
		return
	}

	c.vlog.Debug("collected enough new-views to propose a block",
		zap.Uint64("timedout view", msg.View))

	// if all new-views received before replica became a leader it must enter new round and then wait for data
	view := c.view

	c.timeoutCert = c.timeouts.Cert
	c.syncView(c.prepare, c.prepareCert, c.timeouts.Cert)
	if view == msg.View {
		c.nextRound(false)
	} else {
		c.waitingData = true
		c.Progress.WaitingData = true
	}
}
