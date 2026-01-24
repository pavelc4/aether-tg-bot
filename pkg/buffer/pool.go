package buffer

import (
	"sync"
)


const DefaultSize = 512 * 1024
type Pool struct {
	pool sync.Pool
	size int
}


func NewPool(size int) *Pool {
	return &Pool{
		size: size,
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

func (p *Pool) Get() []byte {
	return p.pool.Get().([]byte)
}
func (p *Pool) Put(b []byte) {
	if cap(b) < p.size {
		return
	}
	b = b[:p.size]
	p.pool.Put(b)
}

var Default = NewPool(DefaultSize)
func Get() []byte {
	return Default.Get()
}
func Put(b []byte) {
	Default.Put(b)
}
