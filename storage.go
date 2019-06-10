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

type cacheConfig struct {
	EndPoint int64
}

var (
	cachePath string
	cfgDB     *bolt.DB
)

func initStorage(path string) {
	cachePath = path
	db, err := bolt.Open(cachePath+"/cache.db", 0755, nil)
	if err != nil {
		reportError(err)
		panic(err)
	}

	DisplayError(db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("config"))
		return err
	}))
	cfgDB = db
}

func newStorage(chain string) *storage {
	var obj = &storage{
		DBS:   make([]*bolt.DB, 1000),
		Path:  cachePath + "/" + chain,
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
		if err != nil {
			panic(err)
		}

		c.Lock()
		c.DBS[idx] = db
		c.Unlock()
		db.Update(func(tx *bolt.Tx) error {
			_, err = tx.CreateBucketIfNotExists([]byte(c.Chain))
			return err
		})
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
		bucket := tx.Bucket([]byte(c.Chain))
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
		bucket := tx.Bucket([]byte(c.Chain))
		k := []byte(strconv.Itoa(int(height)))
		bs = bucket.Get(k)
		return nil
	})
	if len(bs) == 0 {
		return nil, errors.New("not exist")
	}
	return decode(bs), nil
}

func getCacheConfig(chain string) (cfg *cacheConfig) {
	cfg = &cacheConfig{}
	cfgDB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("config"))
		bs := bucket.Get([]byte(chain))
		return json.Unmarshal(bs, cfg)
	})
	return
}

func saveCacheConfig(chain string, cfg *cacheConfig) {
	bs, _ := json.Marshal(cfg)
	cfgDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("config"))
		return bucket.Put([]byte(chain), bs)
	})
}

func clearCacheConfig(chain string) error {
	err := cfgDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("config"))
		return bucket.Delete([]byte(chain))
	})
	if err != nil {
		reportError(err)
	}
	return err
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
