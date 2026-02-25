package bufferedvsunbuffered

import (
	"bufio"
	"io"
	"os"
)

// WriteUnbuffered writes totalBytes to f in chunks of chunkSize using os.File directly.
// Each Write call is a syscall.
func WriteUnbuffered(f *os.File, chunkSize, totalBytes int) error {
	chunk := make([]byte, chunkSize)
	for i := range chunk {
		chunk[i] = 'A'
	}
	written := 0
	for written < totalBytes {
		n := chunkSize
		if written+n > totalBytes {
			n = totalBytes - written
		}
		if _, err := f.Write(chunk[:n]); err != nil {
			return err
		}
		written += n
	}
	return nil
}

// WriteBuffered writes totalBytes to f in chunks of chunkSize through a bufio.Writer.
// Small writes are coalesced in the buffer, reducing syscall count.
func WriteBuffered(f *os.File, chunkSize, totalBytes int) error {
	bw := bufio.NewWriter(f)
	chunk := make([]byte, chunkSize)
	for i := range chunk {
		chunk[i] = 'A'
	}
	written := 0
	for written < totalBytes {
		n := chunkSize
		if written+n > totalBytes {
			n = totalBytes - written
		}
		if _, err := bw.Write(chunk[:n]); err != nil {
			return err
		}
		written += n
	}
	return bw.Flush()
}

// ReadUnbuffered reads from f one byte at a time using os.File directly.
func ReadUnbuffered(f *os.File) (int64, error) {
	buf := make([]byte, 1)
	var total int64
	for {
		n, err := f.Read(buf)
		total += int64(n)
		if err == io.EOF {
			break
		}
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// ReadBuffered reads from f one byte at a time through a bufio.Reader.
func ReadBuffered(f *os.File) (int64, error) {
	br := bufio.NewReader(f)
	var total int64
	for {
		_, err := br.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return total, err
		}
		total++
	}
	return total, nil
}
