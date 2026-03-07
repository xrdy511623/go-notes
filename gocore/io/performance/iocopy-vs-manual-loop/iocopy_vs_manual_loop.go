package iocopyvsmanualloop

import (
	"io"
)

// CopyWithIOCopy uses io.Copy which checks for WriterTo/ReaderFrom optimizations.
func CopyWithIOCopy(dst io.Writer, src io.Reader) (int64, error) {
	return io.Copy(dst, src)
}

// CopyWithIOCopyBuffer uses io.CopyBuffer with a caller-provided buffer.
func CopyWithIOCopyBuffer(dst io.Writer, src io.Reader, bufSize int) (int64, error) {
	buf := make([]byte, bufSize)
	return io.CopyBuffer(dst, src, buf)
}

// CopyManualLoop copies data using a hand-written read/write loop.
func CopyManualLoop(dst io.Writer, src io.Reader, bufSize int) (int64, error) {
	buf := make([]byte, bufSize)
	var total int64
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[:nr])
			total += int64(nw)
			if ew != nil {
				return total, ew
			}
		}
		if er != nil {
			if er == io.EOF {
				break
			}
			return total, er
		}
	}
	return total, nil
}

// onlyReader wraps a Reader to strip any WriterTo interface.
// This forces io.Copy to use its fallback buffer path.
type onlyReader struct {
	io.Reader
}

// StripWriterTo wraps r so that io.Copy cannot detect WriterTo.
func StripWriterTo(r io.Reader) io.Reader {
	return onlyReader{r}
}
