package chainpot

import (
	"encoding/json"
	"fmt"
	"github.com/boltdb/bolt"
	"github.com/rs/zerolog/log"
	"path/filepath"
	"strconv"
	"strings"
)

type ConfigCache struct {
	EndPoint int64
	EventID  int64
}

type Storage interface {
	GetConfig() (cache *ConfigCache, addrs map[string]int64)
	SaveConfig(cache *ConfigCache, addrs map[string]int64) error
	SaveAddrs(records map[string]int64) error
	ClearConfig() error
}

type BoltStorage struct {
	Chain    string
	Database *bolt.DB
}

func NewBoltStorage(dbPath string, chain string) Storage {
	var obj = &BoltStorage{
		Chain: strings.ToLower(chain),
	}
	absPath, err := filepath.Abs(dbPath)
	if err != nil {
		panic("db_path error")
	}

	var filename = fmt.Sprintf("%s/%s.db", absPath, chain)
	if db, err := bolt.Open(filename, 0755, nil); err == nil {
		obj.Database = db
	} else {
		log.Fatal().Msgf("Open DB Error: %s", err.Error())
	}

	if err := obj.Database.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("config"))
		return err
	}); err != nil {
		log.Fatal().Msgf("Create Bucket Error: %s", err.Error())
	}

	if err := obj.Database.Update(func(tx *bolt.Tx) error {
		if _, err := tx.CreateBucketIfNotExists([]byte("addrs")); err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Fatal().Msgf("Create Bucket Error: %s", err.Error())
	}

	return obj
}

func (c *BoltStorage) GetConfig() (cfg *ConfigCache, addrs map[string]int64) {
	cfg = &ConfigCache{}
	c.Database.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("config"))
		bs := bucket.Get([]byte(c.Chain))
		return json.Unmarshal(bs, cfg)
	})

	addrs = make(map[string]int64)
	c.Database.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("addrs"))
		return bucket.ForEach(func(k, v []byte) error {
			key := string(k)
			val, _ := strconv.Atoi(string(v))
			addrs[key] = int64(val)
			return nil
		})
	})
	return
}

func (c *BoltStorage) SaveConfig(cfg *ConfigCache, addrs map[string]int64) error {
	bs, _ := json.Marshal(cfg)
	err1 := c.Database.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("config"))
		err := bucket.Put([]byte(c.Chain), bs)
		if err != nil {
			log.Error().Msg(err.Error())
		}
		return err
	})
	if err1 != nil {
		log.Error().Msgf(err1.Error())
	}

	err2 := c.Database.Update(func(tx *bolt.Tx) error {
		bucketName := []byte(fmt.Sprintf("%s_addrs", strings.ToLower(c.Chain)))
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

func (c *BoltStorage) SaveAddrs(records map[string]int64) error {
	err := c.Database.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("addrs"))
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

func (c *BoltStorage) ClearConfig() error {
	err := c.Database.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("config"))
		return bucket.Delete([]byte(c.Chain))
	})
	if err != nil {
		return err
	}

	err = c.Database.Update(func(tx *bolt.Tx) error {
		bucketName := []byte("addrs")
		err := tx.DeleteBucket(bucketName)
		tx.CreateBucketIfNotExists(bucketName)
		return err
	})
	return err
}
