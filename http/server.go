// Package http implements an HTTP/1.1 server.
package http

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"strings"
	"time"
)

const (
	reqBuffLen    = 2 * 1024
	reqMaxBuffLen = 64 * 1024
)

var (
	socketCounter = 0
	verbose       = false
)

// Server defines the Handler used by Serve.
type Server struct {
	Handler func(Request, *Response)
}

// Serve starts the HTTP server listening on port. For each request, handle is
// called with the parsed request and response in their own goroutine.
func (s Server) Serve(ln net.Listener) error {
	r := rand.New(rand.NewSource(99))

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Error while accepting new connection", err)
			continue
		}

		socketCounter++
		if verbose {
			log.Println("handleConnection #", socketCounter)
		}
		req := Request{Headers: make(map[string]string)}
		res := Response{conn: conn, connID: r.Uint32()}
		go handleConnection(req, &res, s.Handler)
	}
	return nil
}

func readRequest(req Request, res *Response) (requestBuff []byte, err error) {
	requestBuff = make([]byte, 0, 8*1024)
	var reqLen int

	for {
		buff := make([]byte, reqBuffLen)
		reqLen, err = res.conn.Read(buff)
		requestBuff = append(requestBuff, buff[:reqLen]...)

		if len(requestBuff) > reqMaxBuffLen {
			log.Println("Request is too big, ignoring the rest.")
			break
		}

		if err != nil && err != io.EOF {
			log.Println("Connection error:", err)
			break
		}

		if err == io.EOF || reqLen < reqBuffLen {
			break
		}
	}
	return
}

func handleConnection(req Request, res *Response, handle func(Request, *Response)) {
	defer func() {
		socketCounter--
		if verbose {
			log.Println(fmt.Sprintf("Closing socket:%d. Total connections:%d", res.connID, socketCounter))
		}
	}()

	for {
		requestBuff, err := readRequest(req, res)

		resStartTime := time.Now()

		if len(requestBuff) == 0 {
			return
		}

		if err != nil && err != io.EOF {
			log.Println("Error while reading socket:", err)
			return
		}

		if verbose {
			log.Println(string(requestBuff[0:]))
		}

		requestLines := strings.Split(string(requestBuff[0:]), crlf)
		req.parseHeaders(requestLines[1:])
		err = req.parseInitialLine(requestLines[0])

		res.Body = make(chan []byte)
		go res.respondOther(req)

		if err != nil {
			res.Status = 400
			res.ContentType = "text/plain"

			res.Body <- []byte(err.Error() + "\n")
			close(res.Body)

			continue
		}

		requestIsValid := true
		log.Println(fmt.Sprintf("sock:%v %s",
			res.connID,
			requestLines[0],
		))

		if len(req.Headers["Host"]) == 0 {
			res.ContentType = "text/plain"
			res.Status = 400
			close(res.Body)
			requestIsValid = false
		}

		switch req.Method {
		case "GET", "HEAD":
		default:
			res.ContentType = "text/plain"
			res.Status = 501
			close(res.Body)
			requestIsValid = false
		}

		if requestIsValid {
			if req.Method == "HEAD" {
				close(res.Body)
			} else {
				handle(req, res)
			}
		}

		log.Println(fmt.Sprintf("sock:%v Completed in %v",
			res.connID,
			time.Since(resStartTime),
		))

		if req.Headers["Connection"] == "close" {
			break
		}
	}
}
