package stl

import (
	"math"
	"sync"
)

const (
	MapSize    = 8
	ChunckSize = 128
)

type chunck []interface{}

var pool = &sync.Pool{
	New: func() interface{} {
		return make(chunck, ChunckSize)
	},
}

//iterator is the index of bucket
type iterator struct {
	chunck int
	index  int
}

//next return next iterator
func (i *iterator) Next() {
	i.chunck, i.index = i.chunck+(i.index+1)/ChunckSize, (i.index+1)%ChunckSize
}

func (i *iterator) Less(o *iterator) bool {
	if i.chunck < o.chunck {
		return true
	} else if i.chunck == o.chunck && i.index < o.index {
		return true
	} else {
		return false
	}

}

type Deque struct {
	mmap []chunck

	begin *iterator
	end   *iterator
}

//the Deque is a double-ended queue
//the init size is MapSize,which is a const value
func NewDeque() *Deque {
	return &Deque{
		mmap: make([]chunck, MapSize),

		begin: &iterator{
			chunck: (MapSize - 1) / 2,
			index:  ChunckSize,
		},
		end: &iterator{
			chunck: (MapSize + 1) / 2,
			index:  -1,
		},
	}
}

//reallocmmap malloc memory and revoke memory
func (d *Deque) reallocmmap() {

	//end has no space and begin has space
	if d.end.chunck == len(d.mmap)-1 && d.begin.chunck >= 1 {
		//cal the offset
		offset := (d.begin.chunck + 1) / 2

		//copy all between d.begin.chunck and d.end.chunck
		copy(d.mmap[d.begin.chunck-offset:d.end.chunck+1-offset], d.mmap[d.begin.chunck:d.end.chunck+1])

		//set nil
		for i := range d.mmap[len(d.mmap)-offset:] {
			d.mmap[i+len(d.mmap)-offset] = nil
		}

		//reindex
		d.begin.chunck -= offset
		d.end.chunck -= offset

		//begin has no space and end has space
	} else if d.begin.chunck == 0 && d.end.chunck <= len(d.mmap)-2 {

		//cal the offset
		offset := (len(d.mmap) - d.end.chunck) / 2

		//copy all between d.begin.chunck and d.end.chunck
		copy(d.mmap[offset:d.end.chunck+offset+1], d.mmap[:d.end.chunck+1])

		//set nil
		for i := range d.mmap[:offset] {
			d.mmap[i] = nil
		}

		//reindex
		d.begin.chunck += offset
		d.end.chunck += offset

	} else if d.end.chunck == len(d.mmap)-1 && d.begin.chunck == 0 {

		//we need to relloc a new map,add two node
		var mmap []chunck

		//if size of d.mmap is smaller than 1024,we double every time
		//if size of d.mmap is bigger than 1024,we add 25% every time
		if len(d.mmap) < 1024 {
			mmap = make([]chunck, 2*len(d.mmap))
		} else {
			mmap = make([]chunck, len(d.mmap)/4+len(d.mmap))
		}

		//cal offset
		offset := float64(len(mmap)-len(d.mmap)) / 2
		d.begin.chunck = int(math.Floor(offset))
		d.end.chunck = len(d.mmap) + int(math.Ceil(offset)) - 1

		//copy all into mmap
		copy(mmap[d.begin.chunck:d.end.chunck+1], d.mmap)

		d.mmap = mmap

		//revoke unused memory when there are only half chuncks have been used
	} else if d.end.chunck-d.begin.chunck < len(d.mmap)/2 && len(d.mmap) > MapSize {
		//new mmap
		mmap := make([]chunck, (d.end.chunck - d.begin.chunck + 3))

		//copy
		copy(mmap[1:len(mmap)-1], d.mmap[d.begin.chunck:d.end.chunck+1])

		//reindex
		d.begin.chunck = 1
		d.end.chunck = len(mmap) - 2

		d.mmap = mmap
	} else {
		//do nothing
	}

}

//pushback add a new val in the back
func (d *Deque) Pushback(val interface{}) {
	//realloc if need
	if d.end.chunck == len(d.mmap)-1 && d.end.index == ChunckSize-1 {
		d.reallocmmap()
	}

	//find the new position
	d.end.chunck, d.end.index = d.end.chunck+(d.end.index+1)/ChunckSize, (d.end.index+1)%ChunckSize

	//alloc memory
	if d.mmap[d.end.chunck] == nil {
		d.mmap[d.end.chunck] = pool.Get().(chunck)
	}

	//add new val
	d.mmap[d.end.chunck][d.end.index] = val
}

//Pushfront add a new val in the front
func (d *Deque) Pushfront(val interface{}) {
	//realloc if need
	if d.begin.chunck == 0 && d.begin.index == 0 {
		d.reallocmmap()
	}

	//find the new position
	d.begin.chunck, d.begin.index = d.begin.chunck+(d.begin.index-ChunckSize)/ChunckSize, (d.begin.index-1+ChunckSize)%ChunckSize

	//alloc memory
	if d.mmap[d.begin.chunck] == nil {
		d.mmap[d.begin.chunck] = pool.Get().(chunck)
	}
	//add new val
	d.mmap[d.begin.chunck][d.begin.index] = val
}

//Popback delete a new val in the back
func (d *Deque) Popback() (interface{}, bool) {
	//revoke used memory if need
	defer d.reallocmmap()

	if d.end.chunck >= d.begin.chunck && d.end.index >= d.begin.index {
		val := d.mmap[d.end.chunck][d.end.index]
		chunck, index := d.end.chunck+(d.end.index-ChunckSize)/ChunckSize, (d.end.index-1+ChunckSize)%ChunckSize

		//if old chunck does not been used ,revoke it
		if chunck != d.end.chunck {
			pool.Put(d.mmap[d.end.chunck])
			d.mmap[d.end.chunck] = nil
		}

		d.end.chunck = chunck
		d.end.index = index
		return val, true
	}

	return nil, false
}

//Popfront delete a new val in the front
func (d *Deque) Popfront() (interface{}, bool) {
	//revoke used memory if need
	defer d.reallocmmap()

	if d.begin.chunck <= d.end.chunck && d.begin.index <= d.end.index {
		val := d.mmap[d.begin.chunck][d.begin.index]
		chunck, index := d.begin.chunck+(d.begin.index+1)/ChunckSize, (d.begin.index+1)%ChunckSize

		//if old chunck does not been used ,revoke it
		if chunck != d.begin.chunck {
			pool.Put(d.mmap[d.begin.chunck])
			d.mmap[d.begin.chunck] = nil
		}

		d.begin.chunck = chunck
		d.begin.index = index
		return val, true
	}

	return nil, false
}

func (d *Deque) Begin() *iterator {
	return &iterator{
		chunck: d.begin.chunck,
		index:  d.begin.index,
	}
}

func (d *Deque) End() *iterator {
	return &iterator{
		chunck: d.end.chunck,
		index:  d.end.index,
	}
}

func (d *Deque) Get(i *iterator) interface{} {
	return d.mmap[i.chunck][i.index]
}
