package kernel

import (
	"bytes"
	"encoding/json"
	"errors"
	"sort"
	"strings"

	"github.com/number571/gopeer/crypto"
)

var (
	_ Chain = &ChainT{}
)

type ChainT struct {
	length BigInt
	blocks []Block
}

type chainJSON struct {
	Blocks [][]byte `json:"blocks"`
}

// TODO: LevelDB -> Create DB
func NewChain(priv crypto.PrivKey, txs []Transaction) Chain {
	genesis := NewBlock([]byte(ChainID))
	for _, tx := range txs {
		err := genesis.Append(tx)
		if err != nil {
			return nil
		}
	}

	err := genesis.Accept(priv)
	if err != nil {
		return nil
	}

	if !genesis.IsValid() {
		return nil
	}

	return &ChainT{
		blocks: []Block{genesis},
		length: NewInt("1"),
	}
}

// TODO: LevelDB -> Gets range of blocks
func (chain *ChainT) Range(x, y BigInt) Objects {
	return chain.blocks[x.Uint64():y.Uint64()]
}

func (chain *ChainT) Length() BigInt {
	return chain.length
}

// TODO: LevelDB -> Get last block
func (chain *ChainT) LastHash() Hash {
	last := chain.length.Uint64() - 1
	return chain.blocks[last].Hash()
}

// TODO: LevelDB -> Push block
func (chain *ChainT) Append(obj Object) error {
	block := obj.(Block)
	if block == nil {
		return errors.New("block is null")
	}

	if !block.IsValid() {
		return errors.New("block is invalid")
	}

	if !bytes.Equal(block.LastHash(), chain.LastHash()) {
		return errors.New("relation is invalid")
	}

	chain.blocks = append(chain.blocks, block)
	chain.length = chain.length.Inc()

	return nil
}

// TODO: LevelDB -> Search blocks
func (chain *ChainT) Find(hash Hash) Object {
	for _, block := range chain.blocks {
		if bytes.Equal(hash, block.Hash()) {
			return block
		}
	}
	return nil
}

// TODO: LevelDB -> Search blocks
func (chain *ChainT) IsValid() bool {
	for _, block := range chain.blocks {
		if !block.IsValid() {
			return false
		}
		if !bytes.Equal(block.LastHash(), chain.LastHash()) {
			return false
		}
	}
	return true
}

// TODO: LevelDB -> Wrap() N blocks
func (chain *ChainT) Wrap() []byte {
	chainConv := &chainJSON{}

	for _, block := range chain.blocks {
		chainConv.Blocks = append(chainConv.Blocks, block.Wrap())
	}

	chainBytes, err := json.Marshal(chainConv)
	if err != nil {
		return nil
	}

	return chainBytes
}

func (chain *ChainT) SelectLazy(validators []PubKey) PubKey {
	var (
		finds []PubKey
		diff  = NewInt("0")
	)

	for _, pub := range validators {
		lazyLevel := chain.LazyInterval(pub)

		if lazyLevel.Cmp(diff) > 0 {
			diff = lazyLevel
			finds = []PubKey{pub}
			continue
		}

		if lazyLevel.Cmp(diff) == 0 {
			finds = append(finds, pub)
			continue
		}
	}

	lenpub := uint64(len(finds))

	if lenpub > 1 {
		sort.SliceStable(finds, func(i, j int) bool {
			return strings.Compare(finds[i].Address(), finds[j].Address()) < 0
		})

		rnum := LoadInt(chain.LastHash()).Uint64()
		finds[0] = finds[rnum%lenpub]
	}

	return finds[0]
}

// TODO: LevelDB -> Search blocks
func (chain *ChainT) LazyInterval(pub PubKey) BigInt {
	var (
		block = chain.Find(chain.LastHash()).(Block)
		diff  = NewInt("0")
	)

	for {
		if pub.Equal(block.Validator()) {
			return diff
		}

		objects := block.Range(NewInt("0"), block.Length())
		if objects == nil {
			return NewInt("-1")
		}

		txs := objects.([]Transaction)
		for _, tx := range txs {
			if pub.Equal(tx.Validator()) {
				return diff
			}
		}

		object := chain.Find(block.LastHash())
		if object == nil {
			return NewInt("-1")
		}
		block = object.(Block)
		diff = diff.Inc()
	}
}
