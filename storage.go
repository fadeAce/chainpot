package chainpot

import (
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/rs/zerolog/log"
	"strconv"
	"strings"
	"sync"
)

type storage struct {
	*sync.Mutex
	DB    *bolt.DB
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
		Chain: chain,
		Mutex: &sync.Mutex{},
	}
	var filename = fmt.Sprintf("%s/%s.db", cachePath, chain)
	if db, err := bolt.Open(filename, 0755, nil); err == nil {
		obj.DB = db
	} else {
		log.Fatal().Msgf("Open DB Error: %s", err.Error())
	}

	if err := obj.DB.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("height_eventid"))
		return err
	}); err != nil {
		log.Fatal().Msgf("Create Bucket Error: %s", err.Error())
	}

	err := cfgDB.Update(func(tx *bolt.Tx) error {
		bucket := fmt.Sprintf("%s_addrs", strings.ToLower(chain))
		if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Error().Msgf("BoltDB Register Chain Error: %s", err.Error())
	}

	return obj
}

func getCacheConfig(chain string) (cfg *cacheConfig, addrs map[string]int64) {
	cfg = &cacheConfig{}
	cfgDB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("config"))
		bs := bucket.Get([]byte(chain))
		return json.Unmarshal(bs, cfg)
	})

	addrs = make(map[string]int64)
	cfgDB.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(fmt.Sprintf("%s_addrs", strings.ToLower(chain))))
		return bucket.ForEach(func(k, v []byte) error {
			key := string(k)
			val, _ := strconv.Atoi(string(v))
			addrs[key] = int64(val)
			return nil
		})
	})
	return
}

func saveCacheConfig(chain string, cfg *cacheConfig, addrs map[string]int64) error {
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
		for key, val := range addrs {
			h := strconv.Itoa(int(val))
			err := bucket.Put([]byte(key), []byte(h))
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
	return nil
}

func saveAddrs(chain string, records map[string]int64) error {
	err := cfgDB.Update(func(tx *bolt.Tx) error {
		bucketName := []byte(fmt.Sprintf("%s_addrs", strings.ToLower(chain)))
		bucket := tx.Bucket(bucketName)
		var hasError error
		for addr, height := range records {
			h := strconv.Itoa(int(height))
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
