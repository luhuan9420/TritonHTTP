package tritonhttp

import (
	"bufio"
	"reflect"
	"strings"
	"testing"
)

func checkGoodRequest(t *testing.T, readErr error, reqGot, reqWant *Request) {
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !reflect.DeepEqual(*reqGot, *reqWant) {
		t.Fatalf("\ngot: %v\nwant: %v", reqGot, reqWant)
	}
}

func checkBadRequest(t *testing.T, readErr error, reqGot *Request) {
	if readErr == nil {
		t.Errorf("\ngot unexpected request: %v\nwant: error", reqGot)
	}
}

func TestReadGoodRequest(t *testing.T) {
	var tests = []struct {
		name    string
		reqText string
		reqWant *Request
	}{
		{
			"Basic",
			"GET /index.html HTTP/1.1\r\n" +
				"Host: test\r\n" +
				"\r\n",
			&Request{
				Method: "GET",
				URL:    "/index.html",
				Proto:  "HTTP/1.1",
				Header: map[string]string{},
				Host:   "test",
				Close:  false,
			},
		},
		{
			"Close",
			"GET /index.html HTTP/1.1\r\n" +
				"Host: test\r\n" +
				"Connection: close\r\n" +
				"\r\n",
			&Request{
				Method: "GET",
				URL:    "/index.html",
				Proto:  "HTTP/1.1",
				Header: map[string]string{},
				Host:   "test",
				Close:  true,
			},
		},
		{
			"MiscHeaders",
			"GET /index.html HTTP/1.1\r\n" +
				"Host: test\r\n" +
				"Connection: close\r\n" +
				"Key1: val1\r\n" +
				"Key2:   val2\r\n" +
				"\r\n",
			&Request{
				Method: "GET",
				URL:    "/index.html",
				Proto:  "HTTP/1.1",
				Header: map[string]string{
					"Key1": "val1",
					"Key2": "val2",
				},
				Host:  "test",
				Close: true,
			},
		},
		{
			"DontExpandSlashHere",
			"GET / HTTP/1.1\r\n" +
				"Host: test\r\n" +
				"\r\n",
			&Request{
				Method: "GET",
				URL:    "/",
				Proto:  "HTTP/1.1",
				Header: map[string]string{},
				Host:   "test",
				Close:  false,
			},
		},
		{
			"EmptyValueInConnection",
			"GET /index.html HTTP/1.1\r\n" +
				"Host: test\r\n" +
				"Connection: \r\n" +
				"\r\n",
			&Request{
				Method: "GET",
				URL:    "/index.html",
				Proto:  "HTTP/1.1",
				Header: map[string]string{},
				Host:   "test",
				Close:  false,
			},
		},
		{
			"MiscHeaders2",
			"GET /index.html HTTP/1.1\r\n" +
				"Host: test\r\n" +
				"Connection: close\r\n" +
				"key1: val1\r\n" +
				"key2: val2  \r\n" +
				"\r\n",
			&Request{
				Method: "GET",
				URL:    "/index.html",
				Proto:  "HTTP/1.1",
				Header: map[string]string{
					"Key1": "val1",
					"Key2": "val2  ",
				},
				Host:  "test",
				Close: true,
			},
		},
		{
			"EmptyValueHeader",
			"GET /index.html HTTP/1.1\r\n" +
				"Host: test\r\n" +
				"Connection: close\r\n" +
				"key1:\r\n" +
				"key2:    \r\n" +
				"\r\n",
			&Request{
				Method: "GET",
				URL:    "/index.html",
				Proto:  "HTTP/1.1",
				Header: map[string]string{
					"Key1": "",
					"Key2": "",
				},
				Host:  "test",
				Close: true,
			},
		},
		{
			"AlternativeFormOfConnection",
			"GET /index.html HTTP/1.1\r\n" +
				"Host: test\r\n" +
				"connection: close\r\n" +
				"\r\n",
			&Request{
				Method: "GET",
				URL:    "/index.html",
				Proto:  "HTTP/1.1",
				Header: map[string]string{},
				Host:   "test",
				Close:  true,
			},
		},
		{
			"OtherValuesInConnection",
			"GET /index.html HTTP/1.1\r\n" +
				"Host: test\r\n" +
				"Connection: something-else\r\n" +
				"\r\n",
			&Request{
				Method: "GET",
				URL:    "/index.html",
				Proto:  "HTTP/1.1",
				Header: map[string]string{},
				Host:   "test",
				Close:  false,
			},
		},
		{
			"MultipleColons",
			"GET /index.html HTTP/1.1\r\n" +
				"Host: test\r\n" +
				"Connection: close\r\n" +
				"key1:123:\r\n" +
				"key2:456::ab:hello?:@\r\n" +
				"\r\n",
			&Request{
				Method: "GET",
				URL:    "/index.html",
				Proto:  "HTTP/1.1",
				Header: map[string]string{
					"Key1": "123:",
					"Key2": "456::ab:hello?:@",
				},
				Host:  "test",
				Close: true,
			},
		},
		{
			"SpaceAfterClose",
			"GET /index.html HTTP/1.1\r\n" +
				"Host: test\r\n" +
				"connection:close  \r\n" +
				"\r\n",
			&Request{
				Method: "GET",
				URL:    "/index.html",
				Proto:  "HTTP/1.1",
				Header: map[string]string{},
				Host:   "test",
				Close:  false,
			},
		},
		{
			"SpaceBeforeClose",
			"GET /index.html HTTP/1.1\r\n" +
				"Host: test\r\n" +
				"connection:   close\r\n" +
				"\r\n",
			&Request{
				Method: "GET",
				URL:    "/index.html",
				Proto:  "HTTP/1.1",
				Header: map[string]string{},
				Host:   "test",
				Close:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqGot, _, err := ReadRequest(bufio.NewReader(strings.NewReader(tt.reqText)))
			checkGoodRequest(t, err, reqGot, tt.reqWant)
		})
	}
}

func TestReadBadRequest(t *testing.T) {
	var tests = []struct {
		name string
		req  string
	}{
		{
			"Basic",
			"This is a bad request\r\n",
		},
		{
			"Empty",
			"\r\n",
		},
		{
			"LeadingAndTrailingWhiteSpace",
			"  GET /index.html HTTP/1.1  \r\n" +
				"Host: test\r\n" +
				"\r\n",
		},
		{
			"WrongHeaders",
			"GET /index.html HTTP/1.1\r\n" +
				"Host: test\r\n" +
				"Connection: close\r\n" +
				"key1 val1\r\n" +
				"\r\n",
		},
		{
			"WrongHeadersKey",
			"GET /index.html HTTP/1.1\r\n" +
				"Host: test\r\n" +
				"Connection: close\r\n" +
				"key?1: val1\r\n" +
				"\r\n",
		},
		{
			"MalformedURL",
			"GET subdir/ HTTP/1.1\r\nHost: test\r\n\r\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqGot, _, err := ReadRequest(bufio.NewReader(strings.NewReader(tt.req)))
			checkBadRequest(t, err, reqGot)
		})
	}
}

func TestReadMultipleRequests(t *testing.T) {
	var tests = []struct {
		name     string
		reqText  string
		reqsWant []*Request
	}{
		{
			"GoodGood",
			"GET /index.html HTTP/1.1\r\nHost: test\r\n\r\n" +
				"GET /index.html HTTP/1.1\r\nHost: test\r\n\r\n",
			[]*Request{
				{
					Method: "GET",
					URL:    "/index.html",
					Proto:  "HTTP/1.1",
					Header: map[string]string{},
					Host:   "test",
					Close:  false,
				},
				{
					Method: "GET",
					URL:    "/index.html",
					Proto:  "HTTP/1.1",
					Header: map[string]string{},
					Host:   "test",
					Close:  false,
				},
			},
		},
		{
			"GoodBad",
			"GET /index.html HTTP/1.1\r\nHost: test\r\n\r\n" +
				"GETT /index.html HTTP/1.1\r\nHost: test\r\n\r\n",
			[]*Request{
				{
					Method: "GET",
					URL:    "/index.html",
					Proto:  "HTTP/1.1",
					Header: map[string]string{},
					Host:   "test",
					Close:  false,
				},
				nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			br := bufio.NewReader(strings.NewReader(tt.reqText))
			for _, reqWant := range tt.reqsWant {
				reqGot, _, err := ReadRequest(br)
				if reqWant != nil {
					checkGoodRequest(t, err, reqGot, reqWant)
				} else {
					checkBadRequest(t, err, reqGot)
				}
			}
		})
	}
}
