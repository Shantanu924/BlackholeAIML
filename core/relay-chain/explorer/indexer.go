package explorer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/Shivam-Patel-G/blackhole-blockchain/core/relay-chain/chain"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

// Indexer manages the indexing of blockchain data for the explorer
type Indexer struct {
	blockchain *chain.Blockchain
	db         *leveldb.DB
	lastIndex  uint64
}

// NewIndexer creates a new indexer instance
func NewIndexer(bc *chain.Blockchain, dbPath string) (*Indexer, error) {
	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open explorer db: %v", err)
	}

	return &Indexer{
		blockchain: bc,
		db:         db,
		lastIndex:  0,
	}, nil
}

// Start runs the indexer in the background
func (idx *Indexer) Start(ctx context.Context) {
	// Restore last indexed height
	lastHeightData, err := idx.db.Get([]byte("meta_last_height"), nil)
	if err == nil {
		var h uint64
		if err := json.Unmarshal(lastHeightData, &h); err == nil {
			idx.lastIndex = h
			log.Printf("📊 Explorer Indexer resumed from height %d\n", h)
		}
	} else {
		log.Printf("📊 Explorer Indexer starting fresh\n")
	}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	defer idx.db.Close()

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Explorer Indexer shutting down")
			return
		case <-ticker.C:
			idx.sync()
		}
	}
}

func (idx *Indexer) sync() {
	idx.blockchain.GetLatestBlock() // just to see it works
	// For simplicity, we just ask the blockchain for blocks greater than lastIndex
	// In a real scenario, we'd use blockchain.GetBlockByIndex if it existed.
	// We'll iterate the blockchain in memory to find missing blocks.

	latest := idx.blockchain.GetLatestBlock()
	if latest == nil {
		return
	}
	latestHeight := latest.Header.Index

	for i := idx.lastIndex; i <= latestHeight; i++ {
		// find block i in blockchain.Blocks
		var found *chain.Block
		// this is O(n) but okay for our test scenario.
		for _, b := range idx.blockchain.Blocks {
			if b.Header.Index == i {
				found = b
				break
			}
		}

		if found != nil {
			idx.indexBlock(found)
			idx.lastIndex = i

			b, _ := json.Marshal(i)
			idx.db.Put([]byte("meta_last_height"), b, nil)
		}
	}
}

func (idx *Indexer) indexBlock(block *chain.Block) {
	// Index block by hash
	blockData, _ := json.Marshal(block)
	idx.db.Put([]byte(fmt.Sprintf("block:%s", block.Hash)), blockData, nil)
	
	// Index block by height
	idx.db.Put([]byte(fmt.Sprintf("block_height:%d", block.Header.Index)), []byte(block.Hash), nil)

	// Index transactions
	for _, tx := range block.Transactions {
		txData, _ := json.Marshal(tx)
		idx.db.Put([]byte(fmt.Sprintf("tx:%s", tx.ID)), txData, nil)
		
		// Index by sender
		if tx.From != "" {
			idx.db.Put([]byte(fmt.Sprintf("address_tx:%s:%d:%s", tx.From, block.Header.Index, tx.ID)), []byte(tx.ID), nil)
		}
		
		// Index by receiver
		if tx.To != "" {
			idx.db.Put([]byte(fmt.Sprintf("address_tx:%s:%d:%s", tx.To, block.Header.Index, tx.ID)), []byte(tx.ID), nil)
		}
	}
}

// GetBlockByHash retrieves a block from index
func (idx *Indexer) GetBlockByHash(hash string) (*chain.Block, error) {
	data, err := idx.db.Get([]byte(fmt.Sprintf("block:%s", hash)), nil)
	if err != nil {
		return nil, err
	}
	var b chain.Block
	json.Unmarshal(data, &b)
	return &b, nil
}

// GetBlockByHeight retrieves a block from index
func (idx *Indexer) GetBlockByHeight(height uint64) (*chain.Block, error) {
	hash, err := idx.db.Get([]byte(fmt.Sprintf("block_height:%d", height)), nil)
	if err != nil {
		return nil, err
	}
	return idx.GetBlockByHash(string(hash))
}

// GetTransaction retrieves a tx from index
func (idx *Indexer) GetTransaction(hash string) (*chain.Transaction, error) {
	data, err := idx.db.Get([]byte(fmt.Sprintf("tx:%s", hash)), nil)
	if err != nil {
		return nil, err
	}
	var tx chain.Transaction
	json.Unmarshal(data, &tx)
	return &tx, nil
}

// GetAddressTransactions retrieves all txs for an address
func (idx *Indexer) GetAddressTransactions(address string) ([]*chain.Transaction, error) {
	var txs []*chain.Transaction
	prefix := []byte(fmt.Sprintf("address_tx:%s:", address))
	iter := idx.db.NewIterator(util.BytesPrefix(prefix), nil)
	defer iter.Release()
	
	for iter.Next() {
		txHash := string(iter.Value())
		tx, err := idx.GetTransaction(txHash)
		if err == nil {
			txs = append(txs, tx)
		}
	}
	return txs, nil
}
