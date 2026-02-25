package scannervsreadline

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

// GenerateLines creates a reader with the specified number of lines, each of lineLen bytes.
func GenerateLines(numLines, lineLen int) io.Reader {
	var buf bytes.Buffer
	line := strings.Repeat("X", lineLen)
	for i := 0; i < numLines; i++ {
		buf.WriteString(line)
		buf.WriteByte('\n')
	}
	return bytes.NewReader(buf.Bytes())
}

// ReadWithScanner reads all lines using bufio.Scanner.
func ReadWithScanner(r io.Reader, maxTokenSize int) (int, error) {
	scanner := bufio.NewScanner(r)
	if maxTokenSize > bufio.MaxScanTokenSize {
		scanner.Buffer(make([]byte, 0, maxTokenSize), maxTokenSize)
	}
	count := 0
	for scanner.Scan() {
		_ = scanner.Text()
		count++
	}
	return count, scanner.Err()
}

// ReadWithReadString reads all lines using bufio.Reader.ReadString('\n').
func ReadWithReadString(r io.Reader) (int, error) {
	br := bufio.NewReader(r)
	count := 0
	for {
		_, err := br.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return count, err
		}
		count++
	}
	return count, nil
}

// ReadWithReadLine reads all lines using bufio.Reader.ReadLine().
func ReadWithReadLine(r io.Reader) (int, error) {
	br := bufio.NewReaderSize(r, 64*1024)
	count := 0
	for {
		_, isPrefix, err := br.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return count, err
		}
		if !isPrefix {
			count++
		}
	}
	return count, nil
}
