package http

import (
	"fmt"
	log "github.com/siadat/gofile/log"
	"io"
	"math/rand"
	"net"
	"strings"
	"time"
)

const (
	buffSize    = 2 * 1024
	maxBuffSize = 64 * 1024
)

var (
	SocketCounter = 0
)

func Serve(optPort string, callback func(Request, Response)) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%s", optPort))
	if err != nil {
		panic(err)
	}

	r := rand.New(rand.NewSource(99))

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Err("Error while accepting new connection", err)
			continue
		}

		SocketCounter++
		log.Debug("handleConnection #", SocketCounter)
		req := Request{Headers: make(map[string]string), LocalAddr: conn.LocalAddr()}
		res := Response{Conn: conn, ConnID: r.Uint32()}
		go handleConnection(req, res, callback)
	}
}

func handleConnection(req Request, res Response, callback func(Request, Response)) {
	defer func() {
		SocketCounter--
		log.Debug(fmt.Sprintf("Closing socket:%d. Total connections:%d", res.ConnID, SocketCounter))
		res.Conn.Close()
	}()

	for {
		startTime := time.Now()
		requestBuff := make([]byte, 0, 8*1024)
		buff := make([]byte, buffSize)

		var reqLen int
		var err error

		for {
			reqLen, err = res.Conn.Read(buff)
			requestBuff = append(requestBuff, buff[:reqLen]...)

			if len(requestBuff) > maxBuffSize {
				break
			}

			if err != nil || reqLen < buffSize {
				break
			}
		}

		if len(requestBuff) == 0 {
			return
		}

		if err != nil && err != io.EOF {
			log.Err("Error while reading socket:", err)
			return
		}

		log.Debug(string(requestBuff[0:]))

		requestLines := strings.Split(string(requestBuff[0:]), crlf)
		req.ParseHeaders(requestLines[1:])
		err = req.ParseInitialLine(requestLines[0])

		if err != nil {
			res.Status = 400
			res.Body = err.Error()
			res.RespondPlain(req)
			continue
		}

		// ---------
		requestIsValid := true
		log.Normal(fmt.Sprintf("%s sock:%v %s %s",
			time.Now().Format("2006-01-02@15:04:05-0700"),
			res.ConnID,
			req.LocalAddr,
			requestLines[0],
		))

		if len(req.Headers["Host"]) == 0 {
			res.Status = 400
			res.Body = ""
			res.RespondPlain(req)
			requestIsValid = false
		}

		if req.Method != "GET" && req.Method != "HEAD" {
			// Other methods are Not Implemented, and not required by the
			// specs.
			res.Status = 501
			res.RespondPlain(req)
			requestIsValid = false
		}

		if requestIsValid {
			callback(req, res)
		}

		log.Normal(fmt.Sprintf("%s sock:%v %s Completed in %v",
			time.Now().Format("2006-01-02@15:04:05-0700"),
			res.ConnID,
			req.LocalAddr,
			time.Now().Sub(startTime),
		))

		if req.Headers["Connection"] == "close" {
			res.Conn.Close()
			break
		}
	}
}
