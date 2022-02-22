package hotstuff

import (
	"context"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
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
	nodes := createExampleNodes(numNodes, 100 * time.Millisecond)
	// TODO: modify this to work with all nodes created --> almost there

	var totalProposals uint32
	totalProposals = 1

	// declare and initialize global counter to 0
	var proposed uint32
	proposed = 0

	// wait group -- for concurrent goroutines
	var wg sync.WaitGroup

	i := 0
	for i < numNodes {
		wg.Add(1)
		node := nodes[i]
		i += 1

		go func() {

			defer wg.Done()
			
			node.Start()
			// any message from the network
			node.Step(context.Background(), &types.Message{})
			// node.logger.Debug("entering for loop") // extra

			for  {	

				if atomic.LoadUint32(&proposed) >= totalProposals {
					break
				}

				select {
				case <-node.Ready():
					node.logger.Debug("CASE <- READY") // extra
					// increment global counter by 1
					atomic.AddUint32(&proposed, 1)

					node.logger.Debug("PROPOSED: " + strconv.Itoa(int(atomic.LoadUint32(&proposed)))) // extra

					node.Send(context.Background(), Data{
						State: []byte{},
						Root:  []byte{},
						Data:  &types.Data{},
					})

				case msgs := <-node.Messages():
					node.logger.Debug("CASE <- MESSAGES") // extra
					_ = msgs
					// broadcast message or send it to a peer if specified
				case blocks := <-node.Blocks():
					node.logger.Debug("CASE <- BLOCKS") // extra
					_ = blocks
					// each block will appear up to two times
					// first time non-finalized, for speculative execution
					// second time finalized, execution can be persisted on disk
				}
			}

		}()
		
	}

	wg.Wait()

	
}
