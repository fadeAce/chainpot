package chainpot

import (
	"bytes"
	"encoding/gob"
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
		println(err.Error())
		return err
	}

	return db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte(c.Chain))
		if err != nil {
			return err
		}

		k := []byte(strconv.Itoa(int(height)))
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
	var buf = bytes.NewBuffer([]byte(""))
	gob.NewEncoder(buf).Encode(block)
	return buf.Bytes()
}

func decode(data []byte) []*BlockMessage {
	var obj []*BlockMessage
	var buf = bytes.NewBuffer(data)
	gob.NewDecoder(buf).Decode(&obj)
	return obj
}
