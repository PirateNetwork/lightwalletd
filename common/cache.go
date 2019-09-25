package common

import "github.com/pkg/errors"

type BlockCache struct {
	MaxEntries int

	FirstBlock int
	LastBlock  int

	m map[int][]byte
}

func New(maxEntries int) *BlockCache {
	return &BlockCache{
		MaxEntries: maxEntries,
		FirstBlock: -1,
		LastBlock:  -1,
		m:          make(map[int][]byte),
	}
}

func (c *BlockCache) Add(height int, bytes []byte) error {

	if c.FirstBlock == -1 && c.LastBlock == -1 {
		// If this is the first block, prep the data structure
		c.FirstBlock = height
		c.LastBlock = height - 1
	} else if height >= c.FirstBlock && height <= c.LastBlock {
		// Overwriting an existing entry. If so, then remove all
		// subsequent blocks, since this might be a reorg
		for i := height; i <= c.LastBlock; i++ {
			delete(c.m, i)
		}
	}

	if height != c.LastBlock+1 {
		return errors.New("Blocks need to be added sequentially")
	}

	// Add the entry and update the counters
	c.m[height] = bytes
	c.LastBlock = height

	// If the cache is full, remove the oldest block
	if c.LastBlock-c.FirstBlock+1 > c.MaxEntries {
		delete(c.m, c.FirstBlock)
		c.FirstBlock = c.FirstBlock + 1
	}

	return nil
}

func (c *BlockCache) Get(height int) ([]byte, error) {

	if c.LastBlock == -1 || c.FirstBlock == -1 {
		return nil, errors.New("Map is empty")
	}

	if height < c.FirstBlock || height > c.LastBlock {
		return nil, errors.New("Index out of range")
	}

	return c.m[height], nil
}
