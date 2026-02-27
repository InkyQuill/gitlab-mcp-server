package gitlab

import (
	"bytes"
	"encoding/json"
	"testing"
)

func BenchmarkBufferPool_NoPool(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := &bytes.Buffer{}
		buf.WriteString("test data")
		_ = buf.String()
	}
}

func BenchmarkBufferPool_WithPool(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		buf := GetBuffer()
		buf.WriteString("test data")
		_ = buf.String()
		PutBuffer(buf)
	}
}

func BenchmarkJSONMarshal_NoPool(b *testing.B) {
	data := map[string]interface{}{"key": "value", "nested": map[string]string{"a": "b"}}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(data)
	}
}

func BenchmarkJSONMarshal_WithPool(b *testing.B) {
	data := map[string]interface{}{"key": "value", "nested": map[string]string{"a": "b"}}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = MarshalJSONToString(data)
	}
}

func BenchmarkGetByteSlice_64Bytes(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		slice := GetByteSlice(64)
		PutByteSlice(slice)
	}
}

func BenchmarkGetByteSlice_256Bytes(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		slice := GetByteSlice(256)
		PutByteSlice(slice)
	}
}

func BenchmarkGetByteSlice_1024Bytes(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		slice := GetByteSlice(1024)
		PutByteSlice(slice)
	}
}

func BenchmarkGetByteSlice_Large(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = GetByteSlice(10000)
	}
}
