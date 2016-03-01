package http

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
)

type Request struct {
	Method    string
	URL       string
	protocol  string
	Headers   map[string]string
	LocalAddr net.Addr
}

var (
	reAbsURL = regexp.MustCompile("^https?://[^/]+")
)

func ParseDate(date string) (t time.Time) {
	t, err := time.Parse(HTTPTimeFormat, date)
	if err != nil {
		fmt.Println("error parsing", err)
	}
	return
}

func (req *Request) ParseInitialLine(line string) (err error) {
	words := strings.SplitN(line, " ", 3)
	if len(words) != 3 {
		err = errors.New("Invalid initial request line.")
		return
	}
	req.Method = words[0]
	req.URL = words[1]
	req.protocol = words[2]

	if reAbsURL.MatchString(req.URL) {
		req.URL = reAbsURL.ReplaceAllString(req.URL, "")
	}

	return
}

func (req *Request) ParseHeaders(headerLines []string) {
	for _, headerLine := range headerLines {
		headerPair := strings.SplitN(headerLine, ": ", 2)
		if len(headerPair) == 2 {
			req.Headers[headerPair[0]] = headerPair[1]
		}
	}
}
