package etcdconfig

import (
	"encoding/json"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/coreos/etcd/clientv3"
	"golang.org/x/net/context"
)

// Config ...
type Config struct {
	kv    map[string]string
	mutex sync.RWMutex

	cli    *clientv3.Client
	prefix string
	rch    clientv3.WatchChan
}

// New ...
func New(etcdEndpoints []string, cluster string) (*Config, error) {
	// init config
	var err error
	c := &Config{
		kv:     map[string]string{},
		prefix: cluster + "/",
	}

	// new etcd client
	c.cli, err = clientv3.New(clientv3.Config{
		Endpoints:   etcdEndpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, err
	}

	// init and watch
	err = c.initAndWatch()
	if err != nil {
		c.cli.Close()
		return nil, err
	}

	// loop to update
	go func() {
		for {
			for wresp := range c.rch {
				for _, ev := range wresp.Events {
					c.set(ev.Kv.Key, ev.Kv.Value)
				}
			}
			log.Print("etcd-config watch channel closed")
			for {
				err = c.initAndWatch()
				if err == nil {
					break
				}
				log.Print("etcd-config get failed: ", err)
				time.Sleep(time.Second)
			}
		}
	}()

	return c, nil
}

// Get ...
func (c *Config) Get(key string) string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.kv[key]
}

func (c *Config) String() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	b, _ := json.MarshalIndent(c.kv, "", "  ")
	return string(b)
}

func (c *Config) initAndWatch() error {
	c.rch = c.cli.Watch(context.TODO(), c.prefix, clientv3.WithPrefix())
	resp, err := c.cli.Get(context.TODO(), c.prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}
	for _, kv := range resp.Kvs {
		c.set(kv.Key, kv.Value)
	}
	return nil
}

func (c *Config) set(key, value []byte) {
	strKey := strings.TrimPrefix(string(key), c.prefix)
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if len(value) == 0 {
		delete(c.kv, string(strKey))
	} else {
		c.kv[string(strKey)] = string(value)
	}
}
