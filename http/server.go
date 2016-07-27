// Package http implements an HTTP/1.1 server.
package http

import (
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	log "github.com/siadat/gofile/log"
)

const (
	reqBuffLen    = 2 * 1024
	reqMaxBuffLen = 64 * 1024
)

var (
	socketCounter = 0
)

// Serve starts the HTTP server listening on port. For each request, handle is
// called with the parsed request and response in their own goroutine.
func Serve(port int, handle func(Request, *Response)) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf(":%d", port))
	ln, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		log.Error(err)
		os.Exit(1)
		return
	}

	r := rand.New(rand.NewSource(99))

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Error("Error while accepting new connection", err)
			continue
		}

		socketCounter++
		log.Info("handleConnection #", socketCounter)
		req := Request{Headers: make(map[string]string)}
		res := Response{conn: conn, connID: r.Uint32()}
		go handleConnection(req, &res, handle)
	}
}

func readRequest(req Request, res *Response) (requestBuff []byte, err error) {
	requestBuff = make([]byte, 0, 8*1024)
	var reqLen int

	for {
		buff := make([]byte, reqBuffLen)
		reqLen, err = res.conn.Read(buff)
		requestBuff = append(requestBuff, buff[:reqLen]...)

		if len(requestBuff) > reqMaxBuffLen {
			log.Normal("Request is too big, ignoring the rest.")
			break
		}

		if err != nil && err != io.EOF {
			log.Error("Connection error:", err)
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
		log.Info(fmt.Sprintf("Closing socket:%d. Total connections:%d", res.connID, socketCounter))
		// res.conn.Close()
		// FIXME this might be called before response is done writing
	}()

	for {
		requestBuff, err := readRequest(req, res)

		resStartTime := time.Now()

		if len(requestBuff) == 0 {
			return
		}

		if err != nil && err != io.EOF {
			log.Error("Error while reading socket:", err)
			return
		}

		log.Info(string(requestBuff[0:]))

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

			// if req.Headers["Connection"] == "close" {
			// 	// res.conn.Close()
			// 	break
			// } else {
			// }
			continue
		}

		requestIsValid := true
		log.Normal(fmt.Sprintf("%s sock:%v %s",
			time.Now().Format("2006-01-02 15:04:05-0700"),
			res.connID,
			requestLines[0],
		))

		if len(req.Headers["Host"]) == 0 {
			res.ContentType = "text/plain"
			res.Status = 400
			close(res.Body)
			requestIsValid = false
			// continue
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

		log.Normal(fmt.Sprintf("%s sock:%v Completed in %v",
			time.Now().Format("2006-01-02 15:04:05-0700"),
			res.connID,
			time.Since(resStartTime),
		))

		if req.Headers["Connection"] == "close" {
			break
		}
	}
}
