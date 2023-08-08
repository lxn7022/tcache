package bitmap

import (
	"encoding/binary"
	"io"
	"math"
	"sync"

	"github.com/bits-and-blooms/bitset"
)

func init() {
	Register(EngineBitset, newBitset64)
}

type bitset64 struct {
	sets   map[uint32]*bitset.BitSet
	rwLock sync.RWMutex
}

func newBitset64() (API, error) {
	return &bitset64{make(map[uint32]*bitset.BitSet), sync.RWMutex{}}, nil
}

// Add 将整数x添加到位图
func (b *bitset64) Add(x uint64) {
	bset, value := b.getBitSetAndValue(x)
	bset.Set(value)
}

// Remove 将整数x从位图移除
func (b *bitset64) Remove(x uint64) {
	bset, value := b.getBitSetAndValue(x)
	bset.Clear(value)
}

// Contains 整数x是否包含在位图
func (b *bitset64) Contains(x uint64) bool {
	bset, value := b.getBitSetAndValue(x)
	return bset.Test(value)
}

// IsEmpty 判断位图是否为空
func (b *bitset64) IsEmpty() bool {
	for _, set := range b.sets {
		if !set.None() {
			return false
		}
	}
	return true
}

// Len 返回位图中存储的元素个数
func (b *bitset64) Len() uint64 {
	var count uint64
	for _, set := range b.sets {
		count += uint64(set.Count())
	}
	return count
}

// WriteTo 将位图写到一个流里面
func (b *bitset64) WriteTo(stream io.Writer) (int64, error) {
	var totalLen int64
	// 先写入有多少个set
	if err := binary.Write(stream, binary.BigEndian, uint32(len(b.sets))); err != nil {
		return 0, err
	}
	totalLen += int64(binary.Size(uint32(0)))

	for high, set := range b.sets {
		// 写入key
		if err := binary.Write(stream, binary.BigEndian, high); err != nil {
			return totalLen, err
		}
		totalLen += int64(binary.Size(uint32(0)))

		// 写入set
		len, err := set.WriteTo(stream)
		if err != nil {
			return totalLen, err
		}
		totalLen += len
	}
	return totalLen, nil
}

// ReadFrom 从流里面加载一个位图
func (b *bitset64) ReadFrom(stream io.Reader) (int64, error) {
	// 先清空，防止脏数据污染
	b.Clear()

	var (
		totalSets uint32
		totalLen  int64
	)

	// 先读有多少个set
	if err := binary.Read(stream, binary.BigEndian, &totalSets); err != nil {
		return 0, err
	}
	totalLen += int64(binary.Size(uint32(0)))

	for i := uint32(0); i < totalSets; i++ {
		// 读高32位key
		var high32bits uint32
		if err := binary.Read(stream, binary.BigEndian, &high32bits); err != nil {
			return totalLen, err
		}
		set := &bitset.BitSet{}
		b.sets[high32bits] = set
		// 读set
		len, err := set.ReadFrom(stream)
		if err != nil {
			return totalLen, err
		}
		totalLen += len
	}
	return totalLen, nil
}

// Clear 清空位图
func (b *bitset64) Clear() {
	b.sets = make(map[uint32]*bitset.BitSet)
}

func (b *bitset64) getBitSetAndValue(x uint64) (*bitset.BitSet, uint) {
	high32bits, low32bits := highbits(x), lowbits(x)
	b.rwLock.RLock()
	set, ok := b.sets[high32bits]
	b.rwLock.RUnlock()
	if ok {
		return set, uint(low32bits)
	}
	newSet := &bitset.BitSet{}
	b.rwLock.Lock()
	b.sets[high32bits] = newSet
	b.rwLock.Unlock()
	return newSet, uint(low32bits)
}

func highbits(x uint64) uint32 {
	return uint32(x >> 32)
}

func lowbits(x uint64) uint32 {
	return uint32(x & math.MaxUint32)
}
