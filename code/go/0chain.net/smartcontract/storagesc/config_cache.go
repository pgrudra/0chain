package storagesc

import "sync"

type cache struct {
	config *Config
	l      sync.RWMutex `msg:"-"`
	err    error        `msg:"-"`
}

var cfgwtf = &cache{
	l: sync.RWMutex{},
}
