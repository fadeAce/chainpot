package chainpot

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"math"
	"strconv"
	"sync"
)

type storage struct {
	*sync.Mutex
	DBS   []*bolt.DB
	Path  string
	Chain string
}

func newStorage(path string, chain string) *storage {
	var obj = &storage{
		DBS:   make([]*bolt.DB, 1000),
		Path:  path + "/" + chain,
		Chain: chain,
		Mutex: &sync.Mutex{},
	}
	return obj
}

func (c *storage) getDB(height int64) (db *bolt.DB, err error) {
	var idx = int(math.Ceil(float64(height) / 1000000))
	if c.DBS[idx] == nil {
		var filename = fmt.Sprintf("%s/%s_%04d.db", c.Path, c.Chain, idx)
		db, err = bolt.Open(filename, 0755, nil)
		if err == nil {
			c.Lock()
			c.DBS[idx] = db
			c.Unlock()
		}
		return
	} else {
		return c.DBS[idx], nil
	}
}

func (c *storage) saveBlock(height int64, block []*BlockMessage) error {
	if len(block) == 0 {
		return nil
	}

	var db, err = c.getDB(height)
	if err != nil {
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(c.Chain))
		if err != nil {
			return err
		}

		k := []byte(strconv.Itoa(int(height)))
		if oldBlock, err := c.getBlock(height); err == nil {
			var m = make(map[string]*BlockMessage)
			for _, item := range oldBlock {
				m[item.Hash] = item
			}
			for _, item := range block {
				m[item.Hash] = item
			}
			block = make([]*BlockMessage, 0)
			for _, item := range m {
				block = append(block, item)
			}
		}
		return bucket.Put(k, encode(block))
	})
}

func (c *storage) getBlock(height int64) ([]*BlockMessage, error) {
	var bs = make([]byte, 0)
	var db, err = c.getDB(height)
	if err != nil {
		return nil, err
	}

	db.View(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(c.Chain))
		if err != nil {
			return err
		}

		k := []byte(strconv.Itoa(int(height)))
		bs = bucket.Get(k)
		return nil
	})
	if len(bs) == 0 {
		return nil, errors.New("not exist")
	}
	return decode(bs), nil
}

func encode(block []*BlockMessage) []byte {
	data, _ := json.Marshal(block)
	return data
}

func decode(data []byte) []*BlockMessage {
	var obj = make([]*BlockMessage, 0)
	json.Unmarshal(data, &obj)
	return obj
}
