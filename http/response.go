package http

import (
	"fmt"
	log "github.com/siadat/gofile/log"
	"net"
	"runtime"
	"strings"
	"time"
)

type Response struct {
	Conn          net.Conn
	ConnID        uint32
	Status        int
	BodyChan      chan []byte
	ContentType   string
	ContentLength int64
}

const (
	crlf           = "\r\n"
	HTTPTimeFormat = "Mon, 02 Jan 2006 15:04:05 MST"
	ChunkLength    = 1024
	EmptyLine      = ""
	Version        = "0.3.0"
)

var responsePhrases = map[int]string{
	100: "Continue",
	101: "Switching Protocols",
	200: "OK",
	201: "Created",
	202: "Accepted",
	203: "Non-Authoritative Information",
	204: "No Content",
	205: "Reset Content",
	206: "Partial Content",
	300: "Multiple Choices",
	301: "Moved Permanently",
	302: "Found",
	303: "See Other",
	304: "Not Modified",
	305: "Use Proxy",
	307: "Temporary Redirect",
	400: "Bad Request",
	401: "Unauthorized",
	402: "Payment Required",
	403: "Forbidden",
	404: "Not Found",
	405: "Method Not Allowed",
	406: "Not Acceptable",
	407: "Proxy Authentication Required",
	408: "Request Time-out",
	409: "Conflict",
	410: "Gone",
	411: "Length Required",
	412: "Precondition Failed",
	413: "Request Entity Too Large",
	414: "Request-URI Too Large",
	415: "Unsupported Media Type",
	416: "Requested range not satisfiable",
	417: "Expectation Failed",
	500: "Internal Server Error",
	501: "Not Implemented",
	502: "Bad Gateway",
	503: "Service Unavailable",
	504: "Gateway Time-out",
	505: "HTTP Version not supported",
}

func (res *Response) RespondOther(req Request) {
	respond(req, res)
}

func respondHead(req Request, res *Response) {
	var headers []string

	if res.Status == 0 {
		res.Status = 200
	}

	if req.RangedReq && res.Status == 200 {
		res.Status = 206
	}

	r := req.Ranges[0]
	if res.ContentLength > 0 {
		if r.End < 0 {
			r.End = res.ContentLength + r.End
		}
		if r.Start < 0 {
			r.Start = res.ContentLength + r.Start
		}
		if r.Start > r.End {
			res.Status = 416
		}
	}

	headers = append(headers, fmt.Sprintf("HTTP/1.1 %d %s", res.Status, responsePhrases[res.Status]))

	if req.RangedReq && res.ContentLength > 0 {
		headers = append(headers, fmt.Sprintf("Content-Range: %s-%s/%d",
			fmt.Sprintf("%d", r.Start),
			fmt.Sprintf("%d", r.End),
			res.ContentLength))
	}

	if len(res.ContentType) == 0 {
		res.ContentType = "text/plain"
	}

	headers = append(headers,
		"Connection: keep-alive",
		"Accept-Ranges: byte",
		fmt.Sprintf("Content-Type: %s", res.ContentType),
		fmt.Sprintf("Server: Gofile/%s %s", Version, runtime.Version()),
		fmt.Sprintf("Date: %s", time.Now().UTC().Format(HTTPTimeFormat)),
	)

	headers = append(headers, fmt.Sprintf("Transfer-Encoding: %s", "chunked"))

	if res.ContentLength > 0 {
		headers = append(headers, fmt.Sprintf("Content-Length: %d", r.Length()))
	}

	log.Debug(strings.Join(headers, crlf) + crlf + crlf)
	res.Conn.Write(([]byte)(strings.Join(headers, crlf) + crlf + crlf))
}

func respond(req Request, res *Response) {
	from := 0
	var chunkBuff []byte
	noWriteYet := true

	for content := range res.BodyChan {
		if noWriteYet {
			noWriteYet = false
			respondHead(req, res)
			switch res.Status {
			case 304, 501:
				break
			}
		}

		if len(chunkBuff)+len(content) > ChunkLength && len(chunkBuff) > 0 {
			to := from + len(chunkBuff)
			err := writeToConn(res.Conn, chunkBuff, from, to)
			if err != nil {
				fmt.Println("Socket Write Error > ", err)
				break
			}
			from = 0
			chunkBuff = []byte{}
		}
		chunkBuff = append(chunkBuff, content...)
	}

	if len(chunkBuff) > 0 {
		writeToConn(res.Conn, chunkBuff, 0, len(chunkBuff))
	}

	if noWriteYet {
		respondHead(req, res)
	} else {
		res.Conn.Write(([]byte)(fmt.Sprintf("%d%s%s", 0, crlf, crlf)))
	}

	if req.Headers["Connection"] == "close" {
		res.Conn.Close()
	}
}

func writeToConn(conn net.Conn, content []byte, from int, to int) (err error) {
	written := []byte(fmt.Sprintf("%x%s", to-from, crlf))
	written = append(written, content...)
	written = append(written, []byte(fmt.Sprintf("%s", crlf))...)
	_, err = conn.Write(written)
	return
}
