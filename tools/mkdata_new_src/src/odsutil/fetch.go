package odsutil

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

func FetchUrlAndPipe(url string, pipeFunc func(r io.ReaderAt, size int64) error) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	// Read everything into memory to satisfy ReaderAt
	data, err := readAll(resp.Body)
	if err != nil {
		return err
	}

	size := int64(len(data))
	readerAt := bytes.NewReader(data)

	return pipeFunc(readerAt, size)
}

// copy of io.ReadAll
func readAll(r io.Reader) ([]byte, error) {
	b := make([]byte, 0, 512)
	for {
		n, err := r.Read(b[len(b):cap(b)])
		b = b[:len(b)+n]
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return b, err
		}
		if len(b) == cap(b) {
			// Add more capacity (let append pick how much).
			b = append(b, 0)[:len(b)]
		}
	}
}

