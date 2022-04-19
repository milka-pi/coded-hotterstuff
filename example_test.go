package hotstuff

import (
	"context"
	"fmt"
	"math/rand"
	"sync"

	//"sync/atomic"
	"testing"
	"time"

	"github.com/dshulyak/go-hotstuff/crypto"
	"github.com/dshulyak/go-hotstuff/types"
	"go.uber.org/zap"
)

func createExampleNodes(n int, interval time.Duration) []*Node {
	genesis := randGenesis()
	rng := rand.New(rand.NewSource(*seed))

	logger, err := zap.NewDevelopment()
	must(err)

	replicas := []Replica{}
	pubs, privs, err := crypto.GenerateKeys(rng, n)
	must(err)

	verifier := crypto.NewBLS12381Verifier(2*len(pubs)/3+1, pubs)
	for id, pub := range pubs {
		replicas = append(replicas, Replica{ID: pub})

		signer := crypto.NewBLS12381Signer(privs[id])
		sig := signer.Sign(nil, genesis.Header.Hash())
		verifier.Merge(genesis.Cert.Sig, uint64(id), sig)
	}

	okay := verifier.VerifyAggregated(genesis.Header.Hash(), genesis.Cert.Sig)
	if !okay {
		panic("verifier failed")
	}

	nodes := make([]*Node, n)
	for i, priv := range privs {
		db := NewMemDB()
		store := NewBlockStore(db)
		must(ImportGenesis(store, genesis))

		node := NewNode(logger, store, priv, Config{
			Replicas: replicas,
			ID:       replicas[i].ID,
			Interval: interval,
		})
		nodes[i] = node
	}
	return nodes
}

// func TestFull(t *testing.T) {
// 	// create 4 nodes, where 4 = 4f+1 for f=1
// 	nodes := createExampleNodes(4, 100 * time.Millisecond)
// 	// TODO: modify this to work with all nodes created
// 	node := nodes[0]
// 	node.Start()

// 	// any message from the network
// 	node.Step(context.Background(), &types.Message{})

// 	node.logger.Debug("finished STEP, entering for loop") // extra

// 	proposed := 0
// 	// should be < 10
// 	for proposed < 1 {
// 		select {
// 		case <-node.Ready():
// 			node.logger.Debug("CASE <- READY") // extra
// 			proposed += 1
// 			node.Send(context.Background(), Data{
// 				State: []byte{},
// 				Root:  []byte{},
// 				Data:  &types.Data{},
// 			})
// 		case msgs := <-node.Messages():
// 			node.logger.Debug("CASE <- MESSAGES") // extra
// 			_ = msgs
// 			// broadcast message or send it to a peer if specified
// 		case blocks := <-node.Blocks():
// 			node.logger.Debug("CASE <- BLOCKS") // extra
// 			_ = blocks
// 			// each block will appear up to two times
// 			// first time non-finalized, for speculative execution
// 			// second time finalized, execution can be persisted on disk
// 		}
// 	}
// }

func must(err error) {
	if err != nil {
		panic(err)
	}
}

// func run_node(node *Node) {

// 	node.Start()

// 	// any message from the network
// 	node.Step(context.Background(), &types.Message{})

// 	node.logger.Debug("finished STEP, entering for loop") // extra

// 	// proposed := 0
// 	// should be < 10
// 	for  {
// 		// nprop := <- numProposed
// 		// if nprop >= 1 {
// 		// 	break
// 		// }
// 		select {
// 		case <-node.Ready():
// 			node.logger.Debug("CASE <- READY") // extra
// 			proposed := <- numProposed

// 			println("----------------------------------------------------Number proposed: ", 1) // extra

// 			node.Send(context.Background(), Data{
// 				State: []byte{},
// 				Root:  []byte{},
// 				Data:  &types.Data{},
// 			})

// 		case msgs := <-node.Messages():
// 			node.logger.Debug("CASE <- MESSAGES") // extra
// 			_ = msgs
// 			// broadcast message or send it to a peer if specified
// 		case blocks := <-node.Blocks():
// 			node.logger.Debug("CASE <- BLOCKS") // extra
// 			_ = blocks
// 			// each block will appear up to two times
// 			// first time non-finalized, for speculative execution
// 			// second time finalized, execution can be persisted on disk
// 		}
// 	}
// }

func TestFull(t *testing.T) {
	// create 4 nodes, where 4 = 4f+1 for f=1
	numNodes := 4
	nodes := createExampleNodes(numNodes, 100*time.Millisecond)
	// TODO: modify this to work with all nodes created --> almost there

	var totalProposals int
	totalProposals = 3
	// wait group -- for concurrent goroutines
	var wg sync.WaitGroup

	lock := &sync.Mutex{}
	totalConfirmed := make([]int, numNodes)

	for idx := 0; idx < numNodes; idx++ {
		wg.Add(1)
		go func(i int) {
			node := nodes[i]

			defer wg.Done()

			node.Start()
			// any message from the network
			node.Step(context.Background(), &types.Message{})
			// node.logger.Debug("entering for loop") // extra

			for {
				lock.Lock()
				canExit := true
				for j := 0; j < numNodes; j++ {
					if ! (totalConfirmed[j] >= totalProposals) {
						canExit = false
					}
				}
				lock.Unlock()
				if canExit {
					break
				}

				select {
				case <-node.Ready():
					node.logger.Debug("CASE <- READY") // extra
					node.Send(context.Background(), Data{
						State: []byte{},
						Root:  []byte{},
						Data:  &types.Data{},
					})

				case msgs := <-node.Messages():
					node.logger.Debug("CASE <- MESSAGES") // extra
					// broadcast message or send it to a peer if specified
					for _, m := range msgs {
						if len(m.Recipients) == 0 {
							for rx := 0; rx < numNodes; rx++ {
								rxNode := nodes[int(rx)]
								// sender:
								bytes, _ := m.Message.Marshal()
								// receiver:
								rxMsg := &types.Message{}
								rxMsg.Unmarshal(bytes)
								// pretend network
								rxNode.Step(context.Background(), rxMsg)
							}
						} else {
							for _, rx := range m.Recipients {
								rxNode := nodes[int(rx)]
								bytes, _ := m.Message.Marshal()
								rxMsg := &types.Message{}
								rxMsg.Unmarshal(bytes)
								rxNode.Step(context.Background(), rxMsg)
							}
						}
					}
				case blocks := <-node.Blocks():
					node.logger.Debug("CASE <- BLOCKS") // extra
					for _, b := range blocks {
						if b.Finalized {
							lock.Lock()
							totalConfirmed[i] += 1
							fmt.Println("----------------------------------------")
							fmt.Println(i, " TOTAL CONFIRMED: ", totalConfirmed[i])
							fmt.Println("----------------------------------------")
							lock.Unlock()
						}
					}
				}
			}
		}(idx)
	}
	wg.Wait()
}
