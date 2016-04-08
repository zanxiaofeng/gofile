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
	reqBuffLen    = 2 * 1024
	reqMaxBuffLen = 64 * 1024
)

var (
	SocketCounter = 0
)

func Serve(optPort string, requestCallback func(Request, *Response)) {
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
		go handleConnection(req, &res, requestCallback)
	}
}

func readRequest(req Request, res *Response) (requestBuff []byte, err error) {
	requestBuff = make([]byte, 0, 8*1024)
	var reqLen int

	for {
		buff := make([]byte, reqBuffLen)
		reqLen, err = res.Conn.Read(buff)
		requestBuff = append(requestBuff, buff[:reqLen]...)

		if len(requestBuff) > reqMaxBuffLen {
			log.Normal("Request is too big, ignoring the rest.")
			break
		}

		if err != nil && err != io.EOF {
			log.Err("Connection error:", err)
			break
		}

		if err == io.EOF || reqLen < reqBuffLen {
			break
		}
	}
	return
}

func handleConnection(req Request, res *Response, requestCallback func(Request, *Response)) {
	defer func() {
		SocketCounter--
		log.Debug(fmt.Sprintf("Closing socket:%d. Total connections:%d", res.ConnID, SocketCounter))
		// res.Conn.Close()
		// FIXME this might be called before response is done writing
	}()

	for {
		requestBuff, err := readRequest(req, res)

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

		res.BodyChan = make(chan []byte)
		go res.RespondOther(req)

		if err != nil {
			res.Status = 400
			res.ContentType = "text/plain"

			res.BodyChan <- []byte(err.Error() + "\n")
			close(res.BodyChan)

			// if req.Headers["Connection"] == "close" {
			// 	// res.Conn.Close()
			// 	break
			// } else {
			// }
			continue
		}

		requestIsValid := true
		log.Normal(fmt.Sprintf("%s sock:%v %s %s",
			time.Now().Format("2006-01-02 15:04:05-0700"),
			res.ConnID,
			req.LocalAddr,
			requestLines[0],
		))

		if len(req.Headers["Host"]) == 0 {
			res.ContentType = "text/plain"
			res.Status = 400
			close(res.BodyChan)
			requestIsValid = false
			// continue
		}

		switch req.Method {
		case "GET", "HEAD":
		default:
			res.ContentType = "text/plain"
			res.Status = 501
			close(res.BodyChan)
			requestIsValid = false
		}

		if requestIsValid {
			if req.Method == "HEAD" {
				close(res.BodyChan)
			} else {
				requestCallback(req, res)
			}
		}

		log.Normal(fmt.Sprintf("%s sock:%v %s Completed in %v",
			time.Now().Format("2006-01-02 15:04:05-0700"),
			res.ConnID,
			req.LocalAddr,
			time.Since(resStartTime),
		))

		if req.Headers["Connection"] == "close" {
			break
		}
	}
}
