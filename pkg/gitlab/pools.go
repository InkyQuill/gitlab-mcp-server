package gitlab

import (
	"bytes"
	"encoding/json"
)

// bufferPool maintains a pool of bytes.Buffer objects for reuse
// This reduces allocations when building strings or JSON
var bufferPool = syncBufferPool{
	 buffers: make([]*bytes.Buffer, 0, 16),
}

// syncBufferPool is a simple pool for bytes.Buffer objects
type syncBufferPool struct {
	buffers []*bytes.Buffer
}

// Get retrieves a buffer from the pool or creates a new one
func (p *syncBufferPool) Get() *bytes.Buffer {
	if len(p.buffers) == 0 {
		return &bytes.Buffer{}
	}
	buf := p.buffers[len(p.buffers)-1]
	p.buffers = p.buffers[:len(p.buffers)-1]
	return buf
}

// Put returns a buffer to the pool for reuse
func (p *syncBufferPool) Put(buf *bytes.Buffer) {
	if buf.Cap() > 64*1024 { // Don't pool buffers larger than 64KB
		return
	}
	buf.Reset()
	p.buffers = append(p.buffers, buf)
}

// GetBuffer gets a buffer from the pool
// Remember to call PutBuffer when done
func GetBuffer() *bytes.Buffer {
	return bufferPool.Get()
}

// PutBuffer returns a buffer to the pool
func PutBuffer(buf *bytes.Buffer) {
	bufferPool.Put(buf)
}

// jsonMarshalBufferPool maintains buffers specifically for JSON marshaling
var jsonBufferPool = syncBufferPool{
	buffers: make([]*bytes.Buffer, 0, 32),
}

// MarshalJSONToString marshals v to JSON string using a pooled buffer
// This reduces allocations compared to json.Marshal + string conversion
func MarshalJSONToString(v interface{}) (string, error) {
	buf := jsonBufferPool.Get()
	defer jsonBufferPool.Put(buf)

	if err := json.NewEncoder(buf).Encode(v); err != nil {
		return "", err
	}
	// Trim the newline added by Encoder
	result := buf.String()
	if len(result) > 0 && result[len(result)-1] == '\n' {
		result = result[:len(result)-1]
	}
	return result, nil
}

// byteSlicePool maintains byte slices for temporary use
var byteSlicePool = struct {
	s64  [][]byte  // 64-byte slices
	s256 [][]byte  // 256-byte slices
	s1k  [][]byte  // 1KB slices
}{
	s64:  make([][]byte, 0, 8),
	s256: make([][]byte, 0, 8),
	s1k:  make([][]byte, 0, 4),
}

// GetByteSlice gets a byte slice of approximately the requested size
func GetByteSlice(size int) []byte {
	switch {
	case size <= 64:
		if len(byteSlicePool.s64) == 0 {
			return make([]byte, 64)
		}
		slice := byteSlicePool.s64[len(byteSlicePool.s64)-1]
		byteSlicePool.s64 = byteSlicePool.s64[:len(byteSlicePool.s64)-1]
		return slice[:size]
	case size <= 256:
		if len(byteSlicePool.s256) == 0 {
			return make([]byte, 256)
		}
		slice := byteSlicePool.s256[len(byteSlicePool.s256)-1]
		byteSlicePool.s256 = byteSlicePool.s256[:len(byteSlicePool.s256)-1]
		return slice[:size]
	case size <= 1024:
		if len(byteSlicePool.s1k) == 0 {
			return make([]byte, 1024)
		}
		slice := byteSlicePool.s1k[len(byteSlicePool.s1k)-1]
		byteSlicePool.s1k = byteSlicePool.s1k[:len(byteSlicePool.s1k)-1]
		return slice[:size]
	default:
		// Don't pool large slices
		return make([]byte, size)
	}
}

// PutByteSlice returns a byte slice to the appropriate pool
func PutByteSlice(slice []byte) {
	capacity := cap(slice)
	switch {
	case capacity == 64:
		byteSlicePool.s64 = append(byteSlicePool.s64, slice)
	case capacity == 256:
		byteSlicePool.s256 = append(byteSlicePool.s256, slice)
	case capacity == 1024:
		byteSlicePool.s1k = append(byteSlicePool.s1k, slice)
	// Don't pool other sizes
	}
}
