package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	neturl "net/url"
	"os"
	"path"
	fp "path/filepath"
	"runtime"
	"strings"
	"time"

	http "github.com/siadat/gofile/http"
)

var (
	rootDir   = ""
	startTime = time.Now()
	history   []string
)

func htmlLayoutTop(title string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
    <html>
    <head>
      <title>%s | Gofile</title>
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
    <body>`, title)
}
func htmlLayoutBottom() string {
	return `</body></html>`
}

func htmlLink(href, text string, isDir bool) string {
	if isDir {
		return fmt.Sprintf("<a href='%s'>%s</a>", href, text)
	}
	return fmt.Sprintf("<a>%s</a>", text)
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

		bodyChan <- []byte(fmt.Sprintf("<li%s>%s</li>\n", class, htmlLink(fullPath, filename, fi.IsDir())))
	}
	bodyChan <- []byte("</ul>")
	return
}

func listDirChan(url string, urlUnescaped string, filepath string, res *http.Response) (err error) {
	res.Body <- []byte(htmlLayoutTop(url))
	res.Body <- []byte(fmt.Sprintf("<h1>Directory Listing for %s</h1>", urlUnescaped))
	if false {
		res.Body <- []byte(fmt.Sprintf("<h3>Uptime:%s Goroutines:%d Requests:%d</h3>",
			time.Since(startTime),
			runtime.NumGoroutine(),
			len(history),
		))
	}

	fileInfos, status, errMsg := getFilesInDir(filepath)
	filesLis(fileInfos, url, urlUnescaped, res.Body)
	res.Body <- []byte(htmlLayoutBottom())

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

func tryFilepaths(filepath string) (filepathRet string, file os.FileInfo, err error) {
	filepaths := []string{
		filepath,
		fmt.Sprintf("%s.html", filepath),
		fmt.Sprintf("%s.htm", filepath),
		fmt.Sprintf("%s/index.html", filepath),
		fmt.Sprintf("%s/index.htm", filepath),
	}

	for _, filepathRet = range filepaths {
		file, err = os.Stat(filepathRet)
		if err == nil {
			return filepathRet, file, nil
		}
	}

	return filepathRet, file, err
}

func urlToFilepath(requestURI string) (requestedFilepath string, retErr error) {
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

func getFilesize(filepath string) (fileSize int64) {
	f, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer f.Close()

	if fi, err := f.Stat(); err == nil {
		fileSize = fi.Size()
	}
	return
}

func fileServerHandleRequestGen(optRoot string) func(http.Request, *http.Response) {
	rootDir = getRootDir(optRoot)
	return fileServerHandleRequest
}

func fileServerHandleRequest(req http.Request, res *http.Response) {
	history = append(history, req.URL.Path)

	unescapedURL, _ := neturl.QueryUnescape(req.URL.Path)
	filepath, err := urlToFilepath(unescapedURL)
	if err != nil {
		res.Status = 401
		defer close(res.Body)
		res.Body <- []byte(err.Error() + "\n")
		return
	}

	filepath, file, err := tryFilepaths(filepath)
	if err != nil {
		res.Status = 404
		defer close(res.Body)
		res.Body <- []byte("")
		return
	}

	if file.IsDir() {
		res.Status = 200
		res.ContentType = "text/html"
		defer close(res.Body)
		err = listDirChan(req.URL.Path, unescapedURL, filepath, res)
		// if err != nil { res.Status = 400 }
		return
	}

	res.Status = 400
	defer close(res.Body)
}
