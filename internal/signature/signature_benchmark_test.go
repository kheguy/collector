package signature

import (
	"bytes"
	"testing"
)

func BenchmarkCalculate(b *testing.B) {
	data := bytes.Repeat([]byte(`{"id":"HeapAlloc","type":"gauge","value":16777216},`), 32)
	data[len(data)-1] = ']'
	data = append([]byte{'['}, data...)
	const key = "secret"
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		Calculate(data, key)
	}
}

func BenchmarkValid(b *testing.B) {
	data := bytes.Repeat([]byte(`{"id":"HeapAlloc","type":"gauge","value":16777216},`), 32)
	data[len(data)-1] = ']'
	data = append([]byte{'['}, data...)
	const key = "secret"
	encoded := Calculate(data, key)
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	for b.Loop() {
		if !Valid(data, key, encoded) {
			b.Fatal("invalid signature")
		}
	}
}
