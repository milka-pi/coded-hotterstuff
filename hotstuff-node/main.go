package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"strconv"
	"strings"

	//"sync/atomic"

	"time"

	"github.com/dshulyak/go-hotstuff" // use it as "hotstuff", e.g., hotstuff.Node{}
	"github.com/dshulyak/go-hotstuff/crypto"
	"github.com/dshulyak/go-hotstuff/types"

	"go.uber.org/zap"
)

const (
	NETWORK_TYPE_LISTEN = "tcp" // changed to support IPv4
	NETWORK_TYPE_DIAL = "tcp" // changed to support IPv4
	DEFAULT_ADDRESS_NUMBER = 9000
	DEFAULT_MESSAGE = "hello"
	NUMBER_OF_NODES = 4
	SEED = 0
)

var (
	// all nodes must use the same seed
	seed = int64(SEED)
)

// indexed by node index
func createInMsgsChannel() (inMsgsChan chan *types.Message){
	return make(chan *types.Message, 1000)
}

// indexed by node index
func createArrayOfChannels() (arrayOfChannels []chan *types.Message){
	var _arrayOfChannels []chan *types.Message
	for i := 0; i < NUMBER_OF_NODES; i++ {
		_arrayOfChannels = append(_arrayOfChannels, make(chan *types.Message, 1000))
	}
	return _arrayOfChannels
}

// indexed by node index
func getAddressList() [NUMBER_OF_NODES]string {
	addressList := [NUMBER_OF_NODES]string{}
	for i := 0; i < NUMBER_OF_NODES; i++ {
		addressList[i] = ":" + strconv.Itoa(DEFAULT_ADDRESS_NUMBER + i)
	}
	return addressList
}

// indexed by node index
func getIPAddressList() [NUMBER_OF_NODES]string {
	addressList := [NUMBER_OF_NODES]string{}
	for i := 0; i < NUMBER_OF_NODES; i++ {
		addressList[i] = "127.0.0.1:" + strconv.Itoa(DEFAULT_ADDRESS_NUMBER + i)
	}
	return addressList
}


func randGenesis(rng *rand.Rand) *types.Block {
	header := &types.Header{
		DataRoot: randRoot(rng),
	}
	return &types.Block{
		Header: header,
		Cert: &types.Certificate{
			Block: header.Hash(),
			Sig:   &types.AggregatedSignature{},
		},
		Data: &types.Data{},
	}
}

func randRoot(rng *rand.Rand) []byte {
	root := make([]byte, 32)
	rng.Read(root)
	return root
}

// creates one hotstuff node instance
func createExampleNode(idx int, n int, interval time.Duration) *hotstuff.Node {
	rng := rand.New(rand.NewSource(seed))
	genesis := randGenesis(rng)
	fmt.Println("node", idx, "genesis block hash", genesis.Header.Hash())

	logger, err := zap.NewDevelopment()
	must(err)

	replicas := []hotstuff.Replica{}
	pubs, privs, err := crypto.GenerateKeys(rng, n)
	must(err)

	verifier := crypto.NewBLS12381Verifier(2*len(pubs)/3+1, pubs)
	for id, pub := range pubs {
		replicas = append(replicas, hotstuff.Replica{ID: pub})

		signer := crypto.NewBLS12381Signer(privs[id])
		sig := signer.Sign(nil, genesis.Header.Hash())
		verifier.Merge(genesis.Cert.Sig, uint64(id), sig)
	}

	okay := verifier.VerifyAggregated(genesis.Header.Hash(), genesis.Cert.Sig)
	if !okay {
		panic("verifier failed")
	}

	db := hotstuff.NewMemDB()
	store := hotstuff.NewBlockStore(db)
	must(hotstuff.ImportGenesis(store, genesis))

	node := hotstuff.NewNode(logger, store, privs[idx], hotstuff.Config{
		Replicas: replicas, // just empty array? no
		ID:       replicas[idx].ID,
		Interval: interval,
	})

	return node
}


func must(err error) {
	if err != nil {
		panic(err)
	}
}

// interim layer
func dispatchMessage(node *hotstuff.Node, idx int, m hotstuff.MsgTo, arrayOfChannels []chan *types.Message) {
	// if need to broadcast to all nodes
	if len(m.Recipients) == 0 {
		for rx := 0; rx < NUMBER_OF_NODES; rx++ {
			if (rx == idx) {
				node.Step(context.Background(), m.Message)
			} else {
				arrayOfChannels[rx] <- m.Message
			}
		}
	// if need to send to specific nodes
	} else {
		for _, rx := range m.Recipients {
			if (int(rx) == idx) {
				node.Step(context.Background(), m.Message)
			} else {
				arrayOfChannels[int(rx)] <- m.Message
			}
		}
	}
}

// interim layer: NOT USED
func collectMessages(node *hotstuff.Node, inMsgsChan <-chan *types.Message) {
	for {
		select{
			case msg := <- inMsgsChan:
				node.Step(context.Background(), msg)
		}
	}
}


func entryPoint(ctx context.Context, index int, ipAddressList [NUMBER_OF_NODES]string, confirmedChannel chan int) {

	// addressList := getAddressList()
	arrayOfChannels := createArrayOfChannels()
	inMsgsChan := createInMsgsChannel()

	// nodeAddress := addressList[index]
	myIpAddress := ipAddressList[index]

	go listenForConnections(myIpAddress, index, arrayOfChannels, inMsgsChan)
 
	for i := 0; i < index; i++ {
		fmt.Println("initiating connection to node with index:", i)
		ipAddress_i := ipAddressList[i]
		go initiateConnection(ipAddress_i, index, arrayOfChannels, inMsgsChan)
	}

	//------------------------------------------------------------------------------------------

	// ATTENTION: use same genesis and seed
	node := createExampleNode(index, NUMBER_OF_NODES, 100*time.Millisecond)
	node.Start()
	// any message from the network
	node.Step(context.Background(), &types.Message{})

	for {
		select {
		case <-node.Ready():
			// node.logger.Debug("CASE <- READY") // extra
			fmt.Println("Node ", index, "--> ", "CASE <- READY")
			node.Send(context.Background(), hotstuff.Data{
				State: []byte{},
				Root:  []byte{},
				Data:  &types.Data{},
			})

		case msgs := <-node.Messages():
			// node.logger.Debug("CASE <- MESSAGES") // extra
			fmt.Println("Node ", index, "--> ", "CASE <- MESSAGES")
			// broadcast message to all nodes or send it to a node if specified
			for _, m := range msgs {
				// if need to broadcast to all nodes
				dispatchMessage(node, index, m, arrayOfChannels)
			}

		case blocks := <-node.Blocks():
			// node.logger.Debug("CASE <- BLOCKS") // extra
			fmt.Println("Node ", index, "--> ", "CASE <- BLOCKS")
			for _, b := range blocks {
				if b.Finalized {
					// lock.Lock()
					confirmedChannel <- 1
					// lock.Unlock()
					fmt.Println("+1 block finalized")
				}
			}
		
		// note: can use extra case select instead of go routine
		case msg := <- inMsgsChan:
			node.Step(context.Background(), msg)

		// if cancelFunction() executes
		case <- ctx.Done(): 
			fmt.Println("entryPoint: Time to return")
				return

		}
	}
}


func main() {

	var index int
	var ipAddresses string
	flag.IntVar(&index, "index", 0, "node index")
	flag.StringVar(&ipAddresses, "ipAddresses", "", "List of IP addresses of all nodes") // TODO: use in ECE2 instance
	flag.Parse()
	fmt.Println("Node index:", index)
	ipAddressList := strings.Split(ipAddresses, ",")

	if (len(ipAddressList) != NUMBER_OF_NODES) {
		panic("IP address list does not have correct size!")
	}

	// this piece of code is to go arounfd the []string vs [4]string mismatch
	IPaddressList := [NUMBER_OF_NODES]string{}
	for i := 0; i < NUMBER_OF_NODES; i++ {
		IPaddressList[i] = ipAddressList[i]
	}

	totalConfirmed := make(chan int, 100)

	ctx := context.Background()
	//Derive a context with cancel: NOT USED
	// ctxWithCancel, _ := context.WithCancel(ctx)

	entryPoint(ctx, index, IPaddressList, totalConfirmed) 

	//------------------------------------------------------------------------------------------

	// select{}
}






