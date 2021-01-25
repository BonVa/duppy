package duppy

import (
"math"
"sync"
)

type scaledPool struct {
	size int
	pool sync.Pool
}

func newScaledPool(size int) *scaledPool {
	return &scaledPool{
		size: size,
		pool: sync.Pool{
			New: func() interface{} { return makeSlicePointer(size) },
		},
	}
}

type PoolGroup struct {
	minSize int
	maxSize int
	pools   []*scaledPool
}

func New(minSize, maxSize int) *PoolGroup {
	if maxSize < minSize {
		panic("maxSize can't be less than minSize")
	}
	const multiplier = 2
	var pools []*scaledPool
	curSize := minSize
	for curSize < maxSize {
		pools = append(pools, newScaledPool(curSize))
		curSize *= multiplier
	}
	pools = append(pools, newScaledPool(maxSize))
	return &PoolGroup{
		minSize: minSize,
		maxSize: maxSize,
		pools:   pools,
	}
}

func (p *PoolGroup) findPool(size int) *scaledPool {
	if size > p.maxSize {
		return nil
	}
	idx := int(math.Ceil(math.Log2(float64(size) / float64(p.minSize))))
	if idx < 0 {
		idx = 0
	}
	if idx > len(p.pools)-1 {
		return nil
	}
	return p.pools[idx]
}


func (p *PoolGroup) Get(size int) *[]byte {
	sp := p.findPool(size)
	if sp == nil {
		return makeSlicePointer(size)
	}
	buf := sp.pool.Get().(*[]byte)
	*buf = (*buf)[:size]
	return buf
}

func (p *PoolGroup) Put(b *[]byte) {
	sp := p.findPool(cap(*b))
	if sp == nil {
		return
	}
	*b = (*b)[:cap(*b)]
	sp.pool.Put(b)
}

func makeSlicePointer(size int) *[]byte {
	data := make([]byte, size)
	return &data
}
