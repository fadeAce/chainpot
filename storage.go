package chainpot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type storage struct {
	*sync.Mutex
	Path string
	data []interface{}
}

func newStorage(path string) *storage {
	var obj = &storage{
		Mutex: &sync.Mutex{},
		Path:  path,
		data:  make([]interface{}, 0),
	}

	go setInterval(5*time.Second, obj.save)

	go func() {
		quit := make(chan os.Signal)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit
		obj.save()
	}()
	return obj
}

func (c *storage) save() {
	c.Lock()
	var t = time.Now().UTC()
	var filename = c.Path + fmt.Sprintf("/%d%02d%02d.log", t.Year(), t.Month(), t.Day())
	fd, _ := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)

	var buf = bytes.NewBuffer([]byte(""))
	for i, _ := range c.data {
		b, _ := json.Marshal(c.data[i])
		buf.Write(b)
		buf.Write([]byte("\n"))
	}
	fd.Write(buf.Bytes())
	c.data = make([]interface{}, 0)
	fd.Close()
	c.Unlock()
}

func (c *storage) append(v interface{}) {
	c.Lock()
	c.data = append(c.data, v)
	c.Unlock()
}
