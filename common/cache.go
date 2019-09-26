package common

import (
	"bytes"

	"github.com/adityapk00/lightwalletd/walletrpc"
	"github.com/pkg/errors"
)

type BlockCache struct {
	MaxEntries int

	FirstBlock int
	LastBlock  int

	m map[int]*walletrpc.CompactBlock
}

func New(maxEntries int) *BlockCache {
	return &BlockCache{
		MaxEntries: maxEntries,
		FirstBlock: -1,
		LastBlock:  -1,
		m:          make(map[int]*walletrpc.CompactBlock),
	}
}

func (c *BlockCache) Add(height int, block *walletrpc.CompactBlock) error {
	//println("Cache add", height)
	if c.FirstBlock == -1 && c.LastBlock == -1 {
		// If this is the first block, prep the data structure
		c.FirstBlock = height
		c.LastBlock = height - 1
	} else if height >= c.FirstBlock && height <= c.LastBlock {
		// Overwriting an existing entry. If so, then remove all
		// subsequent blocks, since this might be a reorg
		for i := height; i <= c.LastBlock; i++ {
			//println("Deleteing at height", i)
			delete(c.m, i)
		}
		c.LastBlock = height - 1
	}

	if height != c.LastBlock+1 {
		return errors.New("Blocks need to be added sequentially")
	}

	if c.m[height-1] != nil && !bytes.Equal(block.PrevHash, c.m[height-1].Hash) {
		return errors.New("Prev hash of the block didn't match")
	}

	// Add the entry and update the counters
	c.m[height] = block

	c.LastBlock = height

	// If the cache is full, remove the oldest block
	if c.LastBlock-c.FirstBlock+1 > c.MaxEntries {
		//println("Deleteing at height", c.FirstBlock)
		delete(c.m, c.FirstBlock)
		c.FirstBlock = c.FirstBlock + 1
	}

	//println("Cache size is ", len(c.m))
	return nil
}

func (c *BlockCache) Get(height int) *walletrpc.CompactBlock {
	//println("Cache get", height)
	if c.LastBlock == -1 || c.FirstBlock == -1 {
		return nil
	}

	if height < c.FirstBlock || height > c.LastBlock {
		//println("Cache miss: index out of range")
		return nil
	}

	//println("Cache returned")
	return c.m[height]
}
