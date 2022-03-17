package tritonhttp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	statusOK         = 200
	statusBadRequest = 400
	statusNotFound   = 404
)

var statusText = map[int]string{
	statusOK:         "OK",
	statusBadRequest: "Bad Request",
	statusNotFound:   "Not Found",
}

type Server struct {
	// Addr specifies the TCP address for the server to listen on,
	// in the form "host:port". It shall be passed to net.Listen()
	// during ListenAndServe().
	Addr string // e.g. ":0"

	// DocRoot specifies the path to the directory to serve static files from.
	DocRoot string
}

// ListenAndServe listens on the TCP network address s.Addr and then
// handles requests on incoming connections.
func (s *Server) ListenAndServe() error {

	// Validate the configuration of the server
	if err := s.ValidateServerSetup(); err != nil {
		return fmt.Errorf("server is not up correctly %v", err)
	}
	fmt.Println("Server setup valid!")

	// Server should now start to listen on the configured address
	ln, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	fmt.Println("Listening on", ln.Addr())

	// Making sure the listener is closed when exit
	defer func() {
		err = ln.Close()
		if err != nil {
			fmt.Println("error in closing listener", err)
		}
	}()

	//accept connections forever
	for {
		conn, err := ln.Accept()
		if err != nil {
			continue
		}
		fmt.Println("Accepted connection", conn.RemoteAddr())
		go s.HandleConnection(conn)
	}

	// Hint: call HandleConnection
}

// HandleConnection reads requests from the accepted conn and handles them.
func (s *Server) HandleConnection(conn net.Conn) {
	br := bufio.NewReader(conn)
	for {
		// Set timeout
		if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
			log.Printf("Failed to set timeout for connection %v", conn)
			_ = conn.Close()
			return
		}

		// Try to read next request
		req, bytesReceived, err := ReadRequest(br)

		// Handle EOF
		if errors.Is(err, io.EOF) {
			log.Printf("Connection closed by %v", conn.RemoteAddr())
			_ = conn.Close()
			return
		}

		fmt.Printf("%v\n", bytesReceived)
		// Handle timeout
		// just close the connection (need more)
		if err, ok := err.(net.Error); ok && err.Timeout() {
			if !bytesReceived {
				log.Printf("Connection to %v timed out", conn.RemoteAddr())
				_ = conn.Close()
				return
			}
			if bytesReceived {
				res := &Response{}
				log.Printf("Connection to %v timed out with part of a request sent", conn.RemoteAddr())
				res.HandleBadRequest()
				_ = res.Write(conn)
				_ = conn.Close()
				return
			}
		}

		// Handle bad request
		// request is not a GET
		if err != nil {
			res := &Response{}
			log.Printf("Handle bad request for error %v", err)
			res.HandleBadRequest()
			_ = res.Write(conn)
			_ = conn.Close()
			return
		}

		// Handle good request
		log.Printf("Handle good request: %v", req)
		res := s.HandleGoodRequest(req)
		fmt.Printf("Good request response: %v\n", res)
		// call response write function
		err = res.Write(conn)
		if err != nil {
			fmt.Println(err)
		}

		if req.Close || res.StatusCode != 200 {
			log.Printf("Request close connection")
			_ = conn.Close()
			return
		}

		// Close conn if requested
	}
}

// HandleGoodRequest handles the valid req and generates the corresponding res.
func (s *Server) HandleGoodRequest(req *Request) (res *Response) {
	// validate url: error 404
	res = &Response{}

	if strings.HasSuffix(req.URL, "/") {
		req.URL = req.URL + "index.html"
	}
	fmt.Printf("url: %v\n", req.URL)

	if req.URL == "" {
		res.HandleNotFound(req)
		return res
	}
	// path := filepath.Clean(filepath.Join(s.DocRoot, req.URL))
	// fmt.Printf("Full path: %v\n", s.DocRoot+req.URL)
	path := filepath.Clean(s.DocRoot + req.URL)
	// fmt.Printf("Doc root: %v\n", s.DocRoot)
	fmt.Printf("File path: %v\n", path)

	// fmt.Printf("Has Doc Root prefix? %v\n", strings.HasPrefix(path, s.DocRoot))
	if strings.HasPrefix(path, s.DocRoot) == false {
		res.HandleNotFound(req)
		return res
	}

	fi, err := os.Stat(path)
	if os.IsNotExist(err) {
		res.HandleNotFound(req)
	} else if fi.IsDir() {
		res.HandleNotFound(req)
	} else {
		// fmt.Println("Handle OK")
		res.HandleOK(req, path)
	}
	// fmt.Printf("Response: %v\n", res)
	return res
}

// HandleOK prepares res to be a 200 OK response
// ready to be written back to client.
func (res *Response) HandleOK(req *Request, path string) {
	// fmt.Printf("Handle OK")
	// edit response object value
	res.Proto = req.Proto
	res.StatusCode = statusOK

	file, err := os.Stat(path)
	if err != nil {
		fmt.Println(err)
	}

	// res.Header = req.Header
	res.Header = make(map[string]string)
	res.Header["Date"] = FormatTime(time.Now())
	res.Header["Last-Modified"] = FormatTime(file.ModTime())
	ext := "." + strings.SplitN(path, ".", 2)[1]
	res.Header["Content-Type"] = MIMETypeByExtension(ext)
	res.Header["Content-Length"] = strconv.Itoa(int(file.Size()))
	if req.Close {
		res.Header["Connection"] = "close"
	}

	res.FilePath = path
	res.Request = req
}

// HandleBadRequest prepares res to be a 400 Bad Request response
// ready to be written back to client
func (res *Response) HandleBadRequest() {
	res.Proto = "HTTP/1.1"
	res.StatusCode = statusBadRequest
	res.FilePath = ""

	response_header := make(map[string]string)
	response_header["Date"] = FormatTime(time.Now())
	response_header["Connection"] = "close"
	res.Header = response_header

	res.Request = nil
}

// HandleNotFound prepares res to be a 404 Not Found response
// ready to be written back to client.
func (res *Response) HandleNotFound(req *Request) {
	// fmt.Println("Handle Not Found Error")
	// fmt.Printf("status code got: %v\n", statusNotFound)
	res.StatusCode = statusNotFound
	res.FilePath = ""
	res.Proto = "HTTP/1.1"
	res.Request = nil

	res.Header = req.Header
	res.Header["Date"] = FormatTime(time.Now())
	if req.Close {
		res.Header["Connection"] = "close"
	}
}

func (s *Server) ValidateServerSetup() error {
	fi, err := os.Stat(s.DocRoot)

	if os.IsNotExist(err) {
		return err
	}

	if !fi.IsDir() {
		return fmt.Errorf("doc root %q is not a directory", s.DocRoot)
	}

	return nil
}
