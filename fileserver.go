package main

import (
	"errors"
	"fmt"
	http "github.com/siadat/gofile/http"
	"io/ioutil"
	"mime"
	"os"
	"path"
	fp "path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	rootDir   = ""
	startTime = time.Now()
	history   []string
)

func htmlLayout(content string) string {
	return `<!DOCTYPE html>
    <html>
    <head>
      <title>Gofile</title>
      <meta charset="utf-8" />
	  <style>
	    body { font-family:monospace; padding:0; margin:0; }
	    h1,h2,h3,h4,h5,h6 { margin:10px; padding:10px; }
	    ul { padding:0; margin:0; }
	    li { padding:0; margin:1px 0; background-color:#eee; list-style:none; }
	    li.dir { background-color:#B8D2E8; font-weight:bold; /*eea;*/ }
	    li a { color:#29414E; text-decoration:none; display:block; padding:5px 20px; }
	  </style>
    </head>
    <body>` + content + `</body></html>`
}

func htmlLink(href, text string) string {
	return fmt.Sprintf("<a href='%s'>%s</a>", href, text)
}

func filesLis(fileInfos []os.FileInfo, url string) (content string) {
	content = "<ul>"
	for _, fi := range fileInfos {
		filename := fi.Name()
		class := ""
		if fi.IsDir() {
			class = " class='dir' "
			filename = filename + "/"
		}

		fullPath := strings.Join([]string{url, fi.Name()}, "/")
		// url could end with a "/" or with no "/", so when joined with
		// something else using "/" there could be a double slash ie "//"
		fullPath = strings.Replace(fullPath, "//", "/", 1)

		content += fmt.Sprintf("<li%s>%s</li>\n", class, htmlLink(fullPath, filename))
	}
	content += "</ul>"
	return
}

func listDir(url string, filepath string) (content string, err error) {
	content += fmt.Sprintf("<h1>Directory Listing for %s</h1>", url)
	content += fmt.Sprintf("<h3>Uptime:%s OpenSockets:%d Goroutines:%d Requests:%d</h3>",
		time.Now().Sub(startTime),
		http.SocketCounter,
		runtime.NumGoroutine(),
		len(history),
	)

	fileInfos, status, errMsg := getFilesInDir(filepath)
	content += filesLis(fileInfos, url)
	content = htmlLayout(content)

	if status != 200 {
		err = errors.New(errMsg)
		content = ""
	}

	return
}

func getRootDir() (root string) {
	wd, _ := os.Getwd()
	root, err := fp.Abs(wd)

	if err != nil {
		panic(err)
	}
	return
}

func getFilepath(requestURI string) (requestedFilepath string, retErr error) {
	// Note: path.Join removes '..', so the HasPrefix check is safe for paths
	// that try to traverse parent directory using '..'.
	if len(rootDir) == 0 {
		rootDir = getRootDir()
	}
	requestedFilepath = path.Join(rootDir, requestURI)

	if !strings.HasPrefix(requestedFilepath, rootDir) {
		retErr = errors.New("Requested URI is not allowed")
		requestedFilepath = ""
		return
	}
	return
}

func getFilesInDir(requestedFilepath string) (fileInfos []os.FileInfo, status int, errMsg string) {
	status = 200
	fileInfos, err := ioutil.ReadDir(requestedFilepath)

	if err != nil {
		errMsg = "Requested URI was not found."
		status = 404
	}

	return
}

type ByteRange struct {
	start int64
	end   int64
}

// func parseInt(str string) int {
// 	strconv.Parse()
// }

func parseRangeHeader(headerValue string) (byteRanges []ByteRange) {
	rangePrefix := "bytes="
	byteRanges = make([]ByteRange, 0)

	if !strings.HasPrefix(headerValue, rangePrefix) {
		byteRanges = append(byteRanges, ByteRange{start: 0, end: -1})
		return
	}

	headerValue = headerValue[len(rangePrefix):]
	// regexp.MustCompile(`^(-\d+|\d+-|\d+-\d+)$`)
	for _, value := range strings.Split(headerValue, ",") {

		// Let's say we have 10 bytes: [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]
		// -10 => [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]
		// -7 => [3, 4, 5, 6, 7, 8, 9]
		// -2 => [8, 9]
		// -1 => [9]
		if val, err := strconv.ParseInt(value, 10, 0); err == nil && val < 0 {
			byteRanges = append(byteRanges, ByteRange{start: val, end: -1})
			continue
		}

		// 0- => [0, 1, 2, 3, 4, 5, 6, 7, 8, 9]
		// 3- => [3, 4, 5, 6, 7, 8, 9]
		if strings.HasSuffix(value, "-") {
			if val, err := strconv.ParseInt(value[:len(value)-1], 10, 0); err == nil {
				byteRanges = append(byteRanges, ByteRange{start: val, end: -1})
			}
			continue
		}

		// 1-1 => [1, 1]
		// 3-6 => [3, 4, 5, 6]
		rangeVals := strings.Split(value, "-")
		val1, err1 := strconv.ParseInt(rangeVals[0], 10, 0)
		val2, err2 := strconv.ParseInt(rangeVals[1], 10, 0)
		if err1 == nil && err2 == nil {
			byteRanges = append(byteRanges, ByteRange{start: val1, end: val2})
		}
	}
	return
}

func downloadFile(filepath string, ranges []ByteRange) (buff []byte, contentType string, err error) {
	// TODO at the moment we are respecting the first range only
	rangeFrom := ranges[0].start
	rangeTo := ranges[0].end

	f, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer f.Close()

	var fileSize int64

	if fi, err := f.Stat(); err == nil {
		fileSize = fi.Size()
	}

	if rangeTo < 0 {
		rangeTo = fileSize + rangeTo // + 1
	}

	if rangeFrom < 0 {
		rangeFrom = fileSize + rangeFrom // + 1
	}

	rangeFromNew, err := f.Seek(rangeFrom, 0)
	if err != nil {
		return
	}

	if rangeTo < rangeFromNew {
		rangeTo = rangeFromNew
	}

	buff = make([]byte, rangeTo-rangeFromNew+1)
	_, err = f.Read(buff)
	if err != nil {
		return
	}

	contentType = mime.TypeByExtension(fp.Ext(filepath))
	return
}

func fileServerHandleRequest(req http.Request, res http.Response) {
	history = append(history, req.URL)
	filepath, err := getFilepath(req.URL)
	if err != nil {
		res.Status = 401
		res.Body = err.Error() + "\n"
		res.RespondPlain(req)
		return
	}

	file, err := os.Stat(filepath)
	if err != nil {
		res.Status = 404
		res.Body = err.Error() + "\n"
		res.RespondPlain(req)
		return
	}

	if file.IsDir() {
		res.Body, err = listDir(req.URL, filepath)
		res.Status = 200
		if err != nil {
			res.Status = 400
		}
		res.RespondHTML(req)
		return
	}

	res.Status = 200
	fileIsModified := true
	if len(req.Headers["If-Modified-Since"]) > 0 {
		ifModifiedSince := http.ParseDate(req.Headers["If-Modified-Since"])
		if !file.ModTime().After(ifModifiedSince) {
			fileIsModified = false
		}
	}
	if fileIsModified {
		ranges := parseRangeHeader(req.Headers["Range"])
		res.BodyBytes, res.ContentType, err = downloadFile(filepath, ranges) // ranges[0].start, ranges[0].end)
	} else {
		res.Status = 304
	}

	if err != nil {
		res.Status = 400
	}

	res.RespondOther(req)
}