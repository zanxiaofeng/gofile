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

func Serve(optPort string, requestCallback func(Request, Response)) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", fmt.Sprintf(":%s", optPort))
	ln, err := net.ListenTCP("tcp", tcpAddr)
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
		go handleConnection(req, res, requestCallback)
	}
}

func handleConnection(req Request, res Response, requestCallback func(Request, Response)) {
	defer func() {
		SocketCounter--
		log.Debug(fmt.Sprintf("Closing socket:%d. Total connections:%d", res.ConnID, SocketCounter))
		res.Conn.Close()
	}()

	for {
		requestBuff := make([]byte, 0, 8*1024)

		var reqLen int
		var err error

		for {
			buff := make([]byte, buffSize)
			reqLen, err = res.Conn.Read(buff)
			requestBuff = append(requestBuff, buff[:reqLen]...)

			if len(requestBuff) > maxBuffSize {
				log.Normal("Request is too big, ignoring the rest.")
				break
			}

			if err != nil && err != io.EOF {
				log.Err("Connection error:", err)
				break
			}

			if err == io.EOF || reqLen < buffSize {
				break
			}
		}

		resStartTime := time.Now()

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
			res.BodyChan = make(chan []byte)
			go func() {
				res.BodyChan <- []byte(err.Error() + "\n")
			}()
			res.Status = 400
			res.RespondPlain(req)
			continue
		}

		// ---------
		requestIsValid := true
		log.Normal(fmt.Sprintf("%s sock:%v %s %s",
			time.Now().Format("2006-01-02 15:04:05-0700"),
			res.ConnID,
			req.LocalAddr,
			requestLines[0],
		))

		if len(req.Headers["Host"]) == 0 {
			res.BodyChan = make(chan []byte)
			res.Status = 400
			close(res.BodyChan)
			res.RespondPlain(req)
			requestIsValid = false
		}

		switch req.Method {
		case "GET", "HEAD":
		default:
			res.BodyChan = make(chan []byte)
			res.Status = 501
			close(res.BodyChan)
			res.RespondPlain(req)
			requestIsValid = false
		}

		if requestIsValid {
			res.BodyChan = make(chan []byte)
			requestCallback(req, res)
		}

		log.Normal(fmt.Sprintf("%s sock:%v %s Completed in %v",
			time.Now().Format("2006-01-02 15:04:05-0700"),
			res.ConnID,
			req.LocalAddr,
			time.Since(resStartTime),
		))

		if req.Headers["Connection"] == "close" {
			res.Conn.Close()
			break
		}
	}
}
