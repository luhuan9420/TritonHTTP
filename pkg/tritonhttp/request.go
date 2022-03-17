package tritonhttp

import (
	"bufio"
	"fmt"
	"strings"
)

type Request struct {
	Method string // e.g. "GET"
	URL    string // e.g. "/path/to/a/file"
	Proto  string // e.g. "HTTP/1.1"

	// Header stores misc headers excluding "Host" and "Connection",
	// which are stored in special fields below.
	// Header keys are case-incensitive, and should be stored
	// in the canonical format in this map.
	Header map[string]string

	Host  string // determine from the "Host" header
	Close bool   // determine from the "Connection" header
}

// ReadRequest tries to read the next valid request from br.
//
// If it succeeds, it returns the valid request read. In this case,
// bytesReceived should be true, and err should be nil.
//
// If an error occurs during the reading, it returns the error,
// and a nil request. In this case, bytesReceived indicates whether or not
// some bytes are received before the error occurs. This is useful to determine
// the timeout with partial request received condition.
func ReadRequest(br *bufio.Reader) (req *Request, bytesReceived bool, err error) {
	// assume request is sent
	bytesRec := false
	// Read start line
	line, err := ReadLine(br)
	if err != nil {
		return nil, len(line) != 0, err
	}
	bytesRec = true
	fields := strings.SplitN(line, " ", 3)
	if len(fields) != 3 {
		return nil, bytesRec, fmt.Errorf("could not parse the request line, got fields %v", fields)
	}
	// check method/url/proto valid or not
	// multiple spaces between, no space before or after (only between and only 1 space between)  (piazza)
	if fields[0] != "GET" {
		return nil, bytesRec, fmt.Errorf("invalid method %q", fields[0])
	}

	if len(fields[0]) == 0 || len(fields[1]) == 0 || len(fields[2]) == 0 {
		return nil, bytesRec, fmt.Errorf("Bad Request, empty field")
	}

	if strings.Contains(fields[0], " ") || strings.Contains(fields[1], " ") || strings.Contains(fields[2], " ") {
		return nil, bytesRec, fmt.Errorf("Bad Request, field contains spaces")
	}

	if !strings.HasPrefix(fields[1], "/") {
		return nil, bytesRec, fmt.Errorf("Bad Request, invalid URL starts: %v", fields[1])
	}

	if fields[2] != "HTTP/1.1" {
		return nil, bytesRec, fmt.Errorf("Bad Request, proto not HTTP/1.1, proto: %v", fields[2])
	}

	req = &Request{}
	req.Method = fields[0]
	req.Proto = fields[2]
	//req.Close = false

	req.URL = fields[1]
	// if strings.HasSuffix(req.URL, "/") {
	// 	req.URL = req.URL + "index.html"
	// }
	// fmt.Printf("url: %v\n", req.URL)

	// Read headers
	req.Header = make(map[string]string)
	checkConn := false
	checkHost := false
	// bytesRec = false
	for {
		line, err := ReadLine(br)
		if err != nil {
			fmt.Printf("Error while read line: %v\n", err)
			return nil, bytesRec, err
		}
		if line == "" {
			// header end
			break
		}
		// bytesRec = true
		fmt.Println("Read line from request", line)
		h := strings.SplitN(line, ":", 2)
		// check h valid
		if len(h) != 2 {
			return nil, bytesRec, fmt.Errorf("Bad Request, invalid header format: %v", h)
		}

		if strings.HasSuffix(h[0], " ") || strings.HasPrefix(h[0], " ") {
			return nil, bytesRec, fmt.Errorf("Bad Request, host has space")
		}
		if len(strings.TrimSpace(h[0])) == 0 {
			return nil, bytesRec, fmt.Errorf("Bad Request, host is empty")
		}

		for _, c := range h[0] {
			if (c < '0' || c > '9') && (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && c != '-' {
				return nil, bytesRec, fmt.Errorf("Bad Request, host contains not accepted char: %v\n", h[0])
			}
		}

		key := CanonicalHeaderKey(h[0])
		value := strings.TrimLeft(h[1], " ")

		if key == "Connection" {
			checkConn = true
		}

		// check if it is host
		if key == "Host" {
			req.Host = value
			checkHost = true
		}

		req.Header[key] = value
	}

	// Check required headers
	fmt.Printf("Header: %v\n", req.Header)
	// Handle special headers
	if checkConn {
		if req.Header["Connection"] == "close" {
			req.Close = true
		} else {
			req.Close = false
		}
		delete(req.Header, "Connection")
	}
	if checkHost {
		delete(req.Header, "Host")
	} else {
		return nil, bytesRec, fmt.Errorf("Bad Request: missing host")
	}

	return req, bytesRec, nil
}
