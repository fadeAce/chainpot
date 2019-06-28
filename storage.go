package chainpot

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/rs/zerolog/log"
	"math"
	"strconv"
	"strings"
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
	EventID  int64
}

var (
	cachePath string
	cfgDB     *bolt.DB
)

func initStorage(path string) {
	cachePath = path
	db, openError := bolt.Open(cachePath+"/cache.db", 0755, nil)
	if openError != nil {
		log.Fatal().Msg(openError.Error())
		return
	}

	updateError := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("config"))
		return err
	})
	if updateError != nil {
		log.Error().Msg(updateError.Error())
	}
	cfgDB = db
}

func newStorage(chain string) *storage {
	var obj = &storage{
		DBS:   make([]*bolt.DB, 1000),
		Path:  cachePath + "/" + chain,
		Chain: chain,
		Mutex: &sync.Mutex{},
	}

	err := cfgDB.Update(func(tx *bolt.Tx) error {
		bucket := fmt.Sprintf("%s_addrs", strings.ToLower(chain))
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		return err
	})
	if err != nil {
		log.Error().Msgf("BoltDB Register Chain Error: %s", err.Error())
	}

	return obj
}

func (c *storage) getDB(height int64) (db *bolt.DB, err error) {
	var idx = int(math.Ceil(float64(height) / 1000000))
	if c.DBS[idx] == nil {
		var filename = fmt.Sprintf("%s/%s_%04d.db", c.Path, c.Chain, idx)
		db, err = bolt.Open(filename, 0755, nil)
		if err != nil {
			log.Fatal().Msg(err.Error())
			return
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

	putError := db.Update(func(tx *bolt.Tx) error {
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

	if putError != nil {
		log.Error().Msg(putError.Error())
	}
	return putError
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

func getCacheConfig(chain string) (cfg *cacheConfig, addrs map[string]bool) {
	cfg = &cacheConfig{}
	cfgDB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("config"))
		bs := bucket.Get([]byte(chain))
		return json.Unmarshal(bs, cfg)
	})
	if cfg.EventID == 0 {
		cfg.EventID = 1
	}

	addrs = make(map[string]bool)
	cfgDB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(fmt.Sprintf("%s_addrs", strings.ToLower(chain))))
		return bucket.ForEach(func(k, v []byte) error {
			key := string(k)
			addrs[key] = true
			return nil
		})
	})
	return
}

func saveCacheConfig(chain string, cfg *cacheConfig, addrs map[string]bool) {
	bs, _ := json.Marshal(cfg)
	err1 := cfgDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("config"))
		err := bucket.Put([]byte(chain), bs)
		if err != nil {
			log.Error().Msg(err.Error())
		}
		return err
	})
	if err1 != nil {
		log.Error().Msgf(err1.Error())
	}

	err2 := cfgDB.Update(func(tx *bolt.Tx) error {
		bucketName := []byte(fmt.Sprintf("%s_addrs", strings.ToLower(chain)))
		bucket := tx.Bucket(bucketName)
		var hasError error
		for key, _ := range addrs {
			err := bucket.Put([]byte(key), []byte(""))
			if err != nil {
				hasError = err
				log.Error().Msgf("BoltDB Put Error: %s", err.Error())
			}
		}
		return hasError
	})
	if err2 != nil {
		log.Error().Msgf(err2.Error())
	}
}

func addAddr(chain string, height int64, addrs []string) error {
	h := strconv.Itoa(int(height))
	err := cfgDB.Update(func(tx *bolt.Tx) error {
		bucketName := []byte(fmt.Sprintf("%s_addrs", strings.ToLower(chain)))
		bucket := tx.Bucket(bucketName)
		var hasError error
		for _, addr := range addrs {
			err := bucket.Put([]byte(addr), []byte(h))
			if err != nil {
				hasError = err
				log.Error().Msgf("BoltDB Put Error: %s", err.Error())
			}
		}
		return hasError
	})
	if err != nil {
		log.Error().Msgf(err.Error())
	}
	return err
}

func clearCacheConfig(chain string) (err error) {
	err1 := cfgDB.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("config"))
		return bucket.Delete([]byte(chain))
	})
	if err1 != nil {
		log.Error().Msg(err1.Error())
	}

	err2 := cfgDB.Update(func(tx *bolt.Tx) error {
		bucketName := []byte(fmt.Sprintf("%s_addrs", strings.ToLower(chain)))
		err := tx.DeleteBucket(bucketName)
		tx.CreateBucketIfNotExists(bucketName)
		return err
	})
	if err2 != nil {
		err = err2
		log.Error().Msg(err2.Error())
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
