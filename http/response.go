package http

import (
	"fmt"
	"net"
	"runtime"
	"strings"
	"time"
)

type Response struct {
	Conn        net.Conn
	ConnID      uint32
	Status      int
	Body        string
	BodyBytes   []byte
	ContentType string
}

const (
	crlf           = "\r\n"
	HTTPTimeFormat = "Mon, 02 Jan 2006 15:04:05 MST"
	ChunkLength    = 1024
	EmptyLine      = ""
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

func (res *Response) RespondHTML(req Request) {
	res.ContentType = "text/html"
	respond(req, res)
}

func (res *Response) RespondPlain(req Request) {
	res.ContentType = "text/plain"
	respond(req, res)
}

func (res *Response) RespondOther(req Request) {
	respond(req, res)
}

func respond(req Request, res *Response) {
	if len(res.BodyBytes) == 0 && len(res.Body) > 0 {
		res.BodyBytes = ([]byte)(res.Body)
	}
	contentLength := len(res.BodyBytes)
	isChunked := contentLength > ChunkLength
	var firstLine string
	firstLine = fmt.Sprintf("HTTP/1.1 %d %s", res.Status, responsePhrases[res.Status])
	if len(res.ContentType) == 0 {
		res.ContentType = "text/plain"
	}

	headers := []string{
		firstLine,
		"Connection: keep-alive",
		fmt.Sprintf("Content-Type: %s", res.ContentType),
		fmt.Sprintf("Server: Gofile/0.1.0 %s", runtime.Version()),
		fmt.Sprintf("Date: %s", time.Now().UTC().Format(HTTPTimeFormat)),
	}
	if isChunked {
		headers = append(headers, fmt.Sprintf("Transfer-Encoding: %s", "chunked"))
	} else {
		headers = append(headers, fmt.Sprintf("Content-Length: %d", contentLength))
	}

	res.Conn.Write(([]byte)(strings.Join(headers, crlf) + crlf + crlf))

	if req.Method == "HEAD" {
		return
	}

	if res.Status == 304 {
		// Do not send body because it is not modified.
		return
	}

	if res.Status == 501 {
		return
	}

	for i := 0; i < contentLength; i += ChunkLength {
		to := i + ChunkLength
		if to > contentLength {
			to = contentLength
		}

		if isChunked {
			res.Conn.Write(([]byte)(fmt.Sprintf("%x%s", to-i, crlf)))
		}

		res.Conn.Write(res.BodyBytes[i:to])

		if isChunked {
			res.Conn.Write(([]byte)(fmt.Sprintf("%s", crlf)))
		}
	}
	if isChunked {
		res.Conn.Write(([]byte)(fmt.Sprintf("%d%s%s", 0, crlf, crlf)))
	}
}
