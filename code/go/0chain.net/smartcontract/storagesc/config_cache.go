package storagesc

import "sync"

type cache struct {
	config *Config
	l      sync.RWMutex
	err    error
}

var cfg = &cache{
	l: sync.RWMutex{},
}

func (*cache) update(conf *Config, err error) {
	cfg.l.Lock()
	cfg.config = conf
	cfg.err = err
	cfg.l.Unlock()
}
