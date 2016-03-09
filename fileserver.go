package main

import (
	"errors"
	"fmt"
	http "github.com/siadat/gofile/http"
	"io/ioutil"
	"mime"
	neturl "net/url"
	"os"
	"path"
	fp "path/filepath"
	"runtime"
	"strings"
	"time"
)

var (
	rootDir   = ""
	startTime = time.Now()
	history   []string
)

func htmlLayoutTop() string {
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
    <body>`
}
func htmlLayoutBottom() string {
	return `</body></html>`
}

func htmlLink(href, text string) string {
	return fmt.Sprintf("<a href='%s'>%s</a>", href, text)
}

func filesLis(fileInfos []os.FileInfo, url string, urlUnescaped string, bodyChan chan []byte) {
	bodyChan <- []byte("<ul>")
	for _, fi := range fileInfos {
		filename := fi.Name()
		class := ""
		if fi.IsDir() {
			class = " class='dir' "
			filename = filename + "/"
		}

		fullPath := strings.Join([]string{url, neturl.QueryEscape(fi.Name())}, "/")
		// url could end with a "/" or with no "/", so when joined with
		// something else using "/" there could be a double slash ie "//"
		fullPath = strings.Replace(fullPath, "//", "/", 1)

		bodyChan <- []byte(fmt.Sprintf("<li%s>%s</li>\n", class, htmlLink(fullPath, filename)))
	}
	bodyChan <- []byte("</ul>")
	return
}

func listDirChan(url string, urlUnescaped string, filepath string, bodyChan chan []byte) (err error) {
	bodyChan <- []byte(htmlLayoutTop())
	bodyChan <- []byte(fmt.Sprintf("<h1>Directory Listing for %s</h1>", urlUnescaped))
	bodyChan <- []byte(fmt.Sprintf("<h3>Uptime:%s OpenSockets:%d Goroutines:%d Requests:%d</h3>",
		time.Now().Sub(startTime),
		http.SocketCounter,
		runtime.NumGoroutine(),
		len(history),
	))

	fileInfos, status, errMsg := getFilesInDir(filepath)
	filesLis(fileInfos, url, urlUnescaped, bodyChan)
	bodyChan <- []byte(htmlLayoutBottom())

	if status != 200 {
		err = errors.New(errMsg)
		// content = ""
	}

	return
}

func getRootDir(optRoot string) (root string) {
	var wd string
	if len(optRoot) == 0 {
		wd, _ = os.Getwd()
	} else {
		wd = optRoot
	}

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
		rootDir = getRootDir("")
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

func downloadFileChan(filepath string, ranges []http.ByteRange, bodyChan chan []byte) (contentType string, err error) {
	// NOTE at the moment we are respecting the first range only
	rangeFrom := ranges[0].Start
	rangeTo := ranges[0].End

	f, err := os.Open(filepath)
	if err != nil {
		return
	}

	var fileSize int64

	if fi, err := f.Stat(); err == nil {
		fileSize = fi.Size()
	}

	if rangeTo < 0 {
		rangeTo = fileSize + rangeTo
	}

	if rangeFrom < 0 {
		rangeFrom = fileSize + rangeFrom
	}

	rangeFromNew, err := f.Seek(rangeFrom, 0)
	if err != nil {
		f.Close()
		return
	}

	if rangeTo < rangeFromNew {
		rangeTo = rangeFromNew
	}

	defer f.Close()
	buff := make([]byte, rangeTo-rangeFromNew+1)
	_, err = f.Read(buff)
	if err != nil {
		return
	}
	bodyChan <- buff

	return
}

func fileServerHandleRequestGen(optRoot string) func(http.Request, http.Response) {
	rootDir = getRootDir(optRoot)
	return fileServerHandleRequest
}

func fileServerHandleRequest(req http.Request, res http.Response) {
	history = append(history, req.URL.Path)

	req.UnescapedURL, _ = neturl.QueryUnescape(req.URL.Path)
	filepath, err := getFilepath(req.UnescapedURL)
	if err != nil {
		go func() {
			defer close(res.BodyChan)
			res.BodyChan <- []byte(err.Error() + "\n")
		}()
		res.Status = 401
		res.RespondPlain(req)
		return
	}

	file, err := os.Stat(filepath)
	if err != nil {
		go func() {
			defer close(res.BodyChan)
		}()
		res.Status = 404
		res.RespondPlain(req)
		return
	}

	if file.IsDir() {
		go func() {
			defer close(res.BodyChan)
			err = listDirChan(req.URL.Path, req.UnescapedURL, filepath, res.BodyChan)
		}()

		// if err != nil { res.Status = 400 }
		res.Status = 200
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
		ranges := http.ParseByteRangeHeader(req.Headers["Range"])
		res.ContentType = mime.TypeByExtension(fp.Ext(filepath))
		go func() {
			defer close(res.BodyChan)
			res.ContentType, err = downloadFileChan(filepath, ranges, res.BodyChan)
		}()
	} else {
		res.Status = 304
	}

	if err != nil {
		res.Status = 400
	}

	res.RespondOther(req)
}
