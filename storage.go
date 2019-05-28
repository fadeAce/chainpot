package chainpot

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/boltdb/bolt"
	"strconv"
)

type storage struct {
	DB    *bolt.DB
	Chain string
}

func newStorage(path string, chain string) *storage {
	var filepath = fmt.Sprintf("%s/%s.db", path, chain)
	db, err := bolt.Open(filepath, 0644, nil)
	if err != nil {
		panic(err)
	}
	var obj = &storage{
		DB:    db,
		Chain: chain,
	}
	return obj
}

func (c *storage) saveBlock(height int64, block []*BlockMessage) error {
	if len(block) == 0 {
		return nil
	}

	return c.DB.Update(func(tx *bolt.Tx) error {
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
	c.DB.View(func(tx *bolt.Tx) error {
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
