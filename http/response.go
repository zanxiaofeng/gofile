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
	BodyChan    chan []byte
	ContentType string
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

func (res *Response) RespondHTML(req Request) {
	res.ContentType = "text/html"
	respondChan(req, res)
}

func (res *Response) RespondPlain(req Request) {
	res.ContentType = "text/plain"
	respondChan(req, res)
}

func (res *Response) RespondOther(req Request) {
	respondChan(req, res)
}

func respondChan(req Request, res *Response) {
	var firstLine string
	firstLine = fmt.Sprintf("HTTP/1.1 %d %s", res.Status, responsePhrases[res.Status])
	if len(res.ContentType) == 0 {
		res.ContentType = "text/plain"
	}

	headers := []string{
		firstLine,
		"Connection: keep-alive",
		"Accept-Ranges: byte",
		fmt.Sprintf("Content-Type: %s", res.ContentType),
		fmt.Sprintf("Server: Gofile/%s %s", Version, runtime.Version()),
		fmt.Sprintf("Date: %s", time.Now().UTC().Format(HTTPTimeFormat)),
	}

	headers = append(headers, fmt.Sprintf("Transfer-Encoding: %s", "chunked"))

	res.Conn.Write(([]byte)(strings.Join(headers, crlf) + crlf + crlf))

	if req.Method == "HEAD" {
		return
	}

	if res.Status == 304 {
		return
	}

	if res.Status == 501 {
		return
	}

	from := 0
	for content := range res.BodyChan {
		to := from + len(content)
		res.Conn.Write(([]byte)(fmt.Sprintf("%x%s", to-from, crlf)))

		res.Conn.Write(content)

		res.Conn.Write(([]byte)(fmt.Sprintf("%s", crlf)))
		from = to
	}
	res.Conn.Write(([]byte)(fmt.Sprintf("%d%s%s", 0, crlf, crlf)))
}
