package buffer

import "sync"

type Pool struct {
	p sync.Pool
}

func (p *Pool) Get() []byte {
	return p.p.Get().([]byte)
}

func (p *Pool) Put(i []byte) {
	p.p.Put(i)
}

func MakeBufPool(size int) *Pool {
	return &Pool{p: sync.Pool{
		New: func() interface{} {
			return make([]byte, size)
		},
	}}
}

var Pool5 = MakeBufPool(5)
var Pool9 = MakeBufPool(9)
var PoolK = MakeBufPool(1024)
