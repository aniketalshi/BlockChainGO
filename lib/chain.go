package blockchain

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/boltdb/bolt"
)

const dbFile = "blocks.db"
const blocksBucket = "Blocks"
const genesisCoinBaseData = "genesis Txn"

// BlockChain defines link of blocks
type BlockChain struct {
	tip []byte
	db  *bolt.DB
}

// Close exports close functionality of db
func (b *BlockChain) Close() {
	b.db.Close()
}

// AddBlock appends the block to end of blockchain
func (b *BlockChain) AddBlock(tx *Transaction) {
	// construct new block and prev hash will be current tip of db
	block := NewBlock([]*Transaction{tx}, b.tip)

	err := b.db.Update(func(tx *bolt.Tx) error {
		bckt := tx.Bucket([]byte(blocksBucket))
		if err := bckt.Put(block.Hash, block.Serialize()); err != nil {
			return err
		}
		if err := bckt.Put([]byte("l"), block.Hash); err != nil {
			return err
		}
		b.tip = block.Hash
		return nil
	})

	if err != nil {
		log.Fatal("AddBlock :", err)
	}
}

// MineBlock takes in new set of transactions and mines a new block and adds those txns to it
func (b *BlockChain) MineBlock(txns []*Transaction) {
	// construct new block and prev hash will be current tip of db
	block := NewBlock(txns, b.tip)

	err := b.db.Update(func(tx *bolt.Tx) error {
		bckt := tx.Bucket([]byte(blocksBucket))
		if err := bckt.Put(block.Hash, block.Serialize()); err != nil {
			return err
		}
		if err := bckt.Put([]byte("l"), block.Hash); err != nil {
			return err
		}
		b.tip = block.Hash
		return nil
	})

	if err != nil {
		log.Fatal("AddBlock :", err)
	}
}

//NewGenesisBlock creates first BlockChain block
func NewGenesisBlock(tx *Transaction) *Block {
	return NewBlock([]*Transaction{tx}, []byte{})
}

//NewBlockChain constructs a new block chain
func NewBlockChain(address string) *BlockChain {

	var tip []byte

	//db, err := bolt.Open(dbFile, 0600, &bolt.Options{Timeout: 1 * time.Minute})
	db, err := bolt.Open(dbFile, 0600, nil)

	if err != nil {
		log.Fatal(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))

		// if we dont already have bucket with blocks
		if b == nil {
			fmt.Println("Creating New Blockchain...")
			newTx := NewCoinBase(address, genesisCoinBaseData)
			bl := NewGenesisBlock(newTx)

			bckt, _ := tx.CreateBucket([]byte(blocksBucket))
			if newerr := bckt.Put(bl.Hash, bl.Serialize()); err != nil {
				return newerr
			}
			if newerr := bckt.Put([]byte("l"), bl.Hash); err != nil {
				return newerr
			}
			tip = bl.Hash
		} else {
			// set tip to last existing hash
			tip = b.Get([]byte("l"))
		}
		return nil
	})

	if err != nil {
		log.Fatal("NewBlockChain :", err)
	}
	return &BlockChain{tip, db}
}

// GetIterator returns pointer to iterate upon the blockchain
func (b *BlockChain) GetIterator() *Iterator {
	return &Iterator{b.tip, b.db}
}

// GetUnspentTxns returns all unspent transactions for given address
func (b *BlockChain) GetUnspentTxns(address string) []Transaction {
	var unspentTxns []Transaction
	var spentTxnMap = make(map[string][]int) // map txnID -> output index

	// go over blocks one by one
	iter := b.GetIterator()
	for {
		blck := iter.Next()

		// go over all Transactions in this block
		for _, txn := range blck.Transactions {
			// get string identifying this transaction
			txID := hex.EncodeToString(txn.ID)

		OutputLoop:
			// go over all outputs in this Txn
			for outIndex, output := range txn.Out {

				// check if this output is spent.
				if spentTxnMap[txID] != nil {
					for _, indx := range spentTxnMap[txID] {
						if indx == outIndex {
							continue OutputLoop
						}
					}
				}

				// check if this output belongs to this address
				if output.CheckOutputUnlock(address) {
					unspentTxns = append(unspentTxns, *txn)
				}

				// if this is not genesis block, go over all inputs
				// that refers to output that belongs to this address
				// and mark them as unspent
				if txn.IsCoinbase() == false {
					for _, inp := range txn.In {
						if inp.CheckInputUnlock(address) {
							spentTxnMap[txID] = append(spentTxnMap[txID], inp.Out)
						}
					}
				}
			}
		}

		if len(blck.PrevBlockHash) == 0 {
			break
		}
	}
	return unspentTxns
}

// GetUnspentOutputs gets unspent outputs
func (b *BlockChain) GetUnspentOutputs(address string) []TxOutput {
	var unspentOuts []TxOutput
	txns := b.GetUnspentTxns(address)

	// go over each txn and each output in it and collect ones which belongs to this address
	for _, txn := range txns {
		// iterate over all outputs
		for _, output := range txn.Out {
			if output.CheckOutputUnlock(address) {
				unspentOuts = append(unspentOuts, output)
			}
		}
	}

	return unspentOuts
}

// FindUnspendableOutputs finds all outputs we can use for given amount for given address
// Returns amt we can spend and map of txnID : output Indx
func (b *BlockChain) FindSpendableOutputs(address string, amount int) (int, map[string][]int) {

	// get all unspent txns
	unspentTxns := b.GetUnspentTxns(address)
	spendableOutputs := make(map[string][]int)

	accumulate := 0

Checkpoint:
	for _, txn := range unspentTxns {

		txID := hex.EncodeToString(txn.ID)
		// iterate over all outputs
		for outIndx, output := range txn.Out {
			if output.CheckOutputUnlock(address) && accumulate < amount {
				spendableOutputs[txID] = append(spendableOutputs[txID], outIndx)
				accumulate += int(output.Value)
			}

			if accumulate >= amount {
				break Checkpoint
			}
		}
	}

	return accumulate, spendableOutputs
}

// Print is utility to print info of blocks
func (b *BlockChain) Print() {

	iter := b.GetIterator()

	for {
		b := iter.Next()
		if b == nil || len(b.PrevBlockHash) == 0 {
			break
		}
		b.Print()
	}
}
