package hotstuff

import (
	"context"
	"time"
	"math/rand"
	"testing"

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

func TestFull(t *testing.T) {
	nodes := createExampleNodes(1, 100 * time.Millisecond)
	node := nodes[0] 
	node.Start()

	// any message from the network
	node.Step(context.Background(), &types.Message{})

	select {
	case <-node.Ready():
		node.Send(context.Background(), Data{
			State: []byte{},
			Root:  []byte{},
			Data:  &types.Data{},
		})
	case msgs := <-node.Messages():
		_ = msgs
		// broadcast message or send it to a peer if specified
	case blocks := <-node.Blocks():
		_ = blocks
		// each block will appear up to two times
		// first time non-finalized, for speculative execution
		// second time finalized, execution can be persisted on disk
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
