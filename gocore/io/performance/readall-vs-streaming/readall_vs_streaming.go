package readallvsstreaming

import (
	"bytes"
	"io"
)

// ProcessReadAll reads the entire content into memory, then processes it.
// Returns a simple checksum (sum of all bytes).
func ProcessReadAll(r io.Reader) (int64, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	var sum int64
	for _, b := range data {
		sum += int64(b)
	}
	return sum, nil
}

// ProcessStreaming reads and processes data in chunks of bufSize.
// Uses constant O(bufSize) memory regardless of input size.
func ProcessStreaming(r io.Reader, bufSize int) (int64, error) {
	buf := make([]byte, bufSize)
	var sum int64
	for {
		n, err := r.Read(buf)
		for i := 0; i < n; i++ {
			sum += int64(buf[i])
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return sum, err
		}
	}
	return sum, nil
}

// NewRepeatingReader returns a Reader that yields the given byte pattern
// repeated until size bytes have been read.
func NewRepeatingReader(pattern []byte, size int) io.Reader {
	if len(pattern) == 0 {
		pattern = []byte{0x42}
	}
	full := make([]byte, size)
	for i := range full {
		full[i] = pattern[i%len(pattern)]
	}
	return bytes.NewReader(full)
}
