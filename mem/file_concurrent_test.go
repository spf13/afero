package mem

import (
	"bytes"
	"encoding/binary"
	"io"
	"sync"
	"testing"
)

func TestFileConcurrentWrite(t *testing.T) {
	t.Parallel()
	const workers, writes = 16, 256
	data := CreateFile("concurrent-write")
	f := NewFileHandle(data)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(value byte) {
			defer wg.Done()
			<-start
			for j := 0; j < writes; j++ {
				if n, err := f.Write([]byte{value}); n != 1 || err != nil {
					t.Errorf("Write = %d, %v; want 1, nil", n, err)
					return
				}
			}
		}(byte(i))
	}
	close(start)
	wg.Wait()
	if len(data.data) != workers*writes {
		t.Fatalf("file size = %d; want %d", len(data.data), workers*writes)
	}
	for i := 0; i < workers; i++ {
		if count := bytes.Count(data.data, []byte{byte(i)}); count != writes {
			t.Errorf("byte %d written %d times; want %d", i, count, writes)
		}
	}
	if offset, err := f.Seek(0, io.SeekCurrent); offset != workers*writes || err != nil {
		t.Errorf("offset = %d, %v; want %d, nil", offset, err, workers*writes)
	}
}

func TestFileConcurrentRead(t *testing.T) {
	t.Parallel()
	const workers, reads = 16, 256
	data := CreateFile("concurrent-read")
	data.data = make([]byte, workers*reads*4)
	for i := 0; i < workers*reads; i++ {
		binary.LittleEndian.PutUint32(data.data[i*4:], uint32(i))
	}
	f := NewReadOnlyFileHandle(data)
	start := make(chan struct{})
	values := make(chan uint32, workers*reads)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			var b [4]byte
			for j := 0; j < reads; j++ {
				if n, err := f.Read(b[:]); n != len(b) || err != nil {
					t.Errorf("Read = %d, %v; want %d, nil", n, err, len(b))
					return
				}
				values <- binary.LittleEndian.Uint32(b[:])
			}
		}()
	}
	close(start)
	wg.Wait()
	close(values)
	seen := make(map[uint32]bool)
	for value := range values {
		if seen[value] {
			t.Fatalf("record %d read more than once", value)
		}
		seen[value] = true
	}
	if len(seen) != workers*reads {
		t.Fatalf("read %d distinct records; want %d", len(seen), workers*reads)
	}
	if n, err := f.Read(make([]byte, 4)); n != 0 || err != io.EOF {
		t.Errorf("Read at EOF = %d, %v; want 0, EOF", n, err)
	}
}

func TestFileConcurrentSeek(t *testing.T) {
	t.Parallel()
	const workers, seeks = 16, 256
	f := NewFileHandle(CreateFile("concurrent-seek"))
	start := make(chan struct{})
	positions := make(chan int64, workers*seeks)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < seeks; j++ {
				position, err := f.Seek(1, io.SeekCurrent)
				if err != nil {
					t.Error(err)
					return
				}
				positions <- position
			}
		}()
	}
	close(start)
	wg.Wait()
	close(positions)
	seen := make(map[int64]bool)
	for position := range positions {
		if position < 1 || position > workers*seeks || seen[position] {
			t.Fatalf("Seek returned out-of-range or duplicate offset %d", position)
		}
		seen[position] = true
	}
	if len(seen) != workers*seeks {
		t.Errorf("got %d distinct positions; want %d", len(seen), workers*seeks)
	}
}
