package gws

import (
	"bytes"
	"encoding/binary"
	"github.com/klauspost/compress/flate"
	"github.com/lxzan/gws/internal"
	"io"
	"math"
	"sync"
	"sync/atomic"
)

const compressionRate = 3

type compressors struct {
	serial      uint64
	size        uint64
	compressors []*compressor
}

func (c *compressors) initialize(num int, level int) *compressors {
	c.size = uint64(internal.ToBinaryNumber(num))
	for i := uint64(0); i < c.size; i++ {
		c.compressors = append(c.compressors, newCompressor(level))
	}
	return c
}

func (c *compressors) Select() *compressor {
	var j = atomic.AddUint64(&c.serial, 1) & (c.size - 1)
	return c.compressors[j]
}

func newCompressor(level int) *compressor {
	fw, _ := flate.NewWriter(nil, level)
	return &compressor{fw: fw, level: level}
}

// 压缩器
type compressor struct {
	sync.Mutex
	level int
	fw    *flate.Writer
}

// Compress 压缩
func (c *compressor) Compress(src []byte, dst *bytes.Buffer) error {
	c.Lock()
	defer c.Unlock()

	c.fw.Reset(dst)
	if err := internal.WriteN(c.fw, src); err != nil {
		return err
	}
	if err := c.fw.Flush(); err != nil {
		return err
	}
	if n := dst.Len(); n >= 4 {
		compressedContent := dst.Bytes()
		if tail := compressedContent[n-4:]; binary.BigEndian.Uint32(tail) == math.MaxUint16 {
			dst.Truncate(n - 4)
		}
	}
	return nil
}

type decompressors struct {
	serial        uint64
	size          uint64
	decompressors []*decompressor
}

func (c *decompressors) initialize(num int, level int) *decompressors {
	c.size = uint64(internal.ToBinaryNumber(num))
	for i := uint64(0); i < c.size; i++ {
		c.decompressors = append(c.decompressors, newDecompressor())
	}
	return c
}

func (c *decompressors) Select() *decompressor {
	var j = atomic.AddUint64(&c.serial, 1) & (c.size - 1)
	return c.decompressors[j]
}

func newDecompressor() *decompressor {
	return &decompressor{fr: flate.NewReader(nil)}
}

type decompressor struct {
	sync.Mutex
	fr io.ReadCloser
}

// Decompress 解压
func (c *decompressor) Decompress(src *bytes.Buffer) (*bytes.Buffer, int, error) {
	c.Lock()
	defer c.Unlock()

	_, _ = src.Write(internal.FlateTail)
	resetter := c.fr.(flate.Resetter)
	_ = resetter.Reset(src, nil) // must return a null pointer
	var dst, idx = myBufferPool.Get(src.Len() * compressionRate)
	_, err := c.fr.(io.WriterTo).WriteTo(dst)
	return dst, idx, err
}
