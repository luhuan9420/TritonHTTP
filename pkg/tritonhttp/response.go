package tritonhttp

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
)

type Response struct {
	StatusCode int    // e.g. 200
	Proto      string // e.g. "HTTP/1.1"

	// Header stores all headers to write to the response.
	// Header keys are case-incensitive, and should be stored
	// in the canonical format in this map.
	Header map[string]string

	// Request is the valid request that leads to this response.
	// It could be nil for responses not resulting from a valid request.
	Request *Request

	// FilePath is the local path to the file to serve.
	// It could be "", which means there is no file to serve.
	FilePath string
}

// Write writes the res to the w.
func (res *Response) Write(w io.Writer) error {
	if err := res.WriteStatusLine(w); err != nil {
		return err
	}
	if err := res.WriteSortedHeaders(w); err != nil {
		return err
	}
	if err := res.WriteBody(w); err != nil {
		return err
	}
	return nil
}

// WriteStatusLine writes the status line of res to w, including the ending "\r\n".
// For example, it could write "HTTP/1.1 200 OK\r\n".
func (res *Response) WriteStatusLine(w io.Writer) error {
	bw := bufio.NewWriter(w)
	// s := &Server{}
	// desc := s.statusText[res.StatusCode]
	str := fmt.Sprintf("%v %v %v\r\n", res.Proto, res.StatusCode, statusText[res.StatusCode])
	_, err := w.Write([]byte(str))
	if err != nil {
		return err
	}

	if err := bw.Flush(); err != nil {
		return err
	}
	return err
}

// WriteSortedHeaders writes the headers of res to w, including the ending "\r\n".
// For example, it could write "Connection: close\r\nDate: foobar\r\n\r\n".
// For HTTP, there is no need to write headers in any particular order.
// TritonHTTP requires to write in sorted order for the ease of testing.
func (res *Response) WriteSortedHeaders(w io.Writer) error {
	// sort headers
	header_keys := make([]string, 0, len(res.Header))
	for k := range res.Header {
		header_keys = append(header_keys, k)
	}
	sort.Strings(header_keys)

	bw := bufio.NewWriter(w)
	for _, key := range header_keys {
		value := res.Header[key]
		str := fmt.Sprintf("%v: %v\r\n", key, value)
		_, err := bw.Write([]byte(str))
		if err != nil {
			return err
		}
	}
	_, err := bw.Write([]byte("\r\n"))
	if err != nil {
		return err
	}
	if err := bw.Flush(); err != nil {
		return err
	}
	return nil
}

// WriteBody writes res' file content as them  response body to w.
// It doesn't write anything if there is no file to serve.
func (res *Response) WriteBody(w io.Writer) error {

	if res.FilePath == "" {
		return nil
	}

	bw := bufio.NewWriter(w)

	data, err := os.ReadFile(res.FilePath)
	if err != nil {
		return err
	}

	_, err = bw.Write(data)
	if err != nil {
		return err
	}

	if err := bw.Flush(); err != nil {
		return err
	}
	return nil
}
