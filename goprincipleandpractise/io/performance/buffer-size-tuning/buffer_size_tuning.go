package buffersizetuning

import (
	"bufio"
	"io"
	"os"
)

// ReadWithBufferSize reads the entire file using bufio.NewReaderSize with the given buffer size.
// Returns total bytes read.
func ReadWithBufferSize(f *os.File, bufSize int) (int64, error) {
	br := bufio.NewReaderSize(f, bufSize)
	return io.Copy(io.Discard, br)
}

// CreateTestFile creates a temp file filled with the specified number of bytes.
func CreateTestFile(size int) (*os.File, error) {
	f, err := os.CreateTemp("", "bench-bufsize-*")
	if err != nil {
		return nil, err
	}
	chunk := make([]byte, 32*1024)
	for i := range chunk {
		chunk[i] = 'A'
	}
	written := 0
	for written < size {
		n := len(chunk)
		if written+n > size {
			n = size - written
		}
		if _, err := f.Write(chunk[:n]); err != nil {
			f.Close()
			os.Remove(f.Name())
			return nil, err
		}
		written += n
	}
	f.Seek(0, 0)
	return f, nil
}
