package main

import (
	http "./http"
	"errors"
	"fmt"
	"io/ioutil"
	"mime"
	"os"
	"path"
	"path/filepath"
	"runtime"
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
      <title>Gottp</title>
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

func listDir(url string, localFilepath string) (content string, err error) {
	content += fmt.Sprintf("<h1>Directory Listing for %s</h1>", url)
	content += fmt.Sprintf("<h3>Uptime:%s OpenSockets:%d Goroutines:%d Requests:%d</h3>",
		time.Now().Sub(startTime),
		http.SocketCounter,
		runtime.NumGoroutine(),
		len(history),
	)
	content += "<ul>"

	files, status, errMsg := getFilesInDir(localFilepath)
	for _, file := range files {
		filename := file.Name()
		class := ""
		if file.IsDir() {
			class = " class='dir' "
			filename = filename + "/"
		}

		fullPath := strings.Join([]string{url, file.Name()}, "/")
		// url could end with a "/" or with no "/", so when joined with
		// something else using "/" there could be a double slash ie "//"
		fullPath = strings.Replace(fullPath, "//", "/", 1)

		content += fmt.Sprintf("<li%s>%s</li>\n", class, htmlLink(fullPath, filename))
	}

	content += "</ul>"
	content = htmlLayout(content)

	if status != 200 {
		err = errors.New(errMsg)
		content = ""
	}

	return
}

func getRootDir() (root string) {
	wd, _ := os.Getwd()
	root, err := filepath.Abs(wd)

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

func getFilesInDir(requestedFilepath string) (files []os.FileInfo, status int, errMsg string) {
	status = 200
	files, err := ioutil.ReadDir(requestedFilepath)

	if err != nil {
		errMsg = "Requested URI was not found."
		status = 404
	}

	return
}

func downloadFile(file string) (buff []byte, contentType string, err error) {
	buff, err = ioutil.ReadFile(file)
	if err != nil {
		return
	}
	contentType = mime.TypeByExtension(filepath.Ext(file))
	return
}

func fileServerHandleRequest(req http.Request, res http.Response) {
	history = append(history, req.URL)
	localFilepath, err := getFilepath(req.URL)
	if err != nil {
		res.Status = 401
		res.Body = err.Error() + "\n"
		res.RespondPlain(req)
		return
	}

	file, err := os.Stat(localFilepath)
	if err != nil {
		res.Status = 404
		res.Body = err.Error() + "\n"
		res.RespondPlain(req)
		return
	}

	if file.IsDir() {
		res.Body, err = listDir(req.URL, localFilepath)
		res.Status = 200
		if err != nil {
			res.Status = 400
		}
		res.RespondHTML(req)
	} else {
		res.BodyBytes, res.ContentType, err = downloadFile(localFilepath)
		res.Status = 200
		if err != nil {
			res.Status = 400
		}
		res.RespondOther(req)
	}
}
