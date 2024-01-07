package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	webserver "github.com/zhongjie-cai/web-server"
)

type myCustomization struct {
	webserver.DefaultCustomization
}

func (customization *myCustomization) ServerCert() *tls.Certificate {
	var cert, err = tls.LoadX509KeyPair("/data/v2ray.crt", "/data/v2ray.key")
	if err != nil {
		return nil
	}
	return &cert
}

func (customization *myCustomization) Routes() []webserver.Route {
	return []webserver.Route{
		{
			Endpoint:   "Root",
			Method:     http.MethodGet,
			Path:       "/",
			ActionFunc: indexAction,
		},
		{
			Endpoint:   "Process",
			Method:     http.MethodPost,
			Path:       "/",
			ActionFunc: processAction,
		},
	}
}

const INDEX_PAGE_CONTENT string = `<html>
  <header>
    <title>Uploader v` + APP_VERSION + `</title>
  </header>
  <body>
    <form method="POST" enctype="multipart/form-data">
      <label>Source Image:&nbsp;</label>
      <input type="file" id="source_image" name="source_image" />
      <br />
      <label>Target image:&nbsp;</label>
      <input type="file" id="target_image" name="target_image"
	    multiple="multiple" />
      <br />
      <label>Name prefix:&nbsp;</label>
      <input type="text" id="name_prefix"
	    name="name_prefix" value="IMG" />
      <br />
      <label>Reactor API:&nbsp;</label>
      <input type="text" id="reactor_api"
	    name="reactor_api" value="http://localhost:7860/reactor/image" />
      <br />
      <label>Quality:&nbsp;</label>
      <input type="text" id="quality"
	    name="quality" value="100" />
      <br />
      <input type="submit" />
	  <br />
	  <br />
	  <label>App Version = ` + APP_VERSION + `</label>
    </form>
  </body>
</html>`

func indexAction(session webserver.Session) (interface{}, error) {
	var request = session.GetRequest()
	var responseWriter = session.GetResponseWriter()
	http.ServeContent(
		responseWriter,
		request,
		"index.html",
		time.Now(),
		strings.NewReader(
			INDEX_PAGE_CONTENT,
		),
	)
	return webserver.SkipResponseHandling()
}

func getImageBytes(multipartForm *multipart.Form, filename string) ([]imageBytes, error) {
	var files, found = multipartForm.File[filename]
	if !found || len(files) < 1 {
		return nil, nil
	}
	var allBytes = make([]imageBytes, 0, len(files))
	for _, file := range files {
		var imageFile, imageErr = file.Open()
		if imageErr != nil {
			return nil, imageErr
		}
		defer imageFile.Close()
		var buffer bytes.Buffer
		var _, bufferErr = buffer.ReadFrom(imageFile)
		if bufferErr != nil {
			return nil, bufferErr
		}
		allBytes = append(allBytes, imageBytes{
			bytes: buffer.Bytes(),
			name:  file.Filename,
		})
	}
	return allBytes, nil
}

func getNamePrefix(multipartForm *multipart.Form) string {
	var namePrefixes, found = multipartForm.Value["name_prefix"]
	if !found || len(namePrefixes) == 0 {
		namePrefixes = []string{"IMG"}
	}
	return namePrefixes[0]
}

func getReactorAPI(multipartForm *multipart.Form) string {
	var reactorAPI, found = multipartForm.Value["reactor_api"]
	if !found || len(reactorAPI) == 0 {
		reactorAPI = []string{"http://localhost:7860/reactor/image"}
	}
	return reactorAPI[0]
}

func getImageQuality(multipartForm *multipart.Form) int {
	var qualities, found = multipartForm.Value["quality"]
	if !found || len(qualities) == 0 {
		return 100
	}
	var quality, err = strconv.Atoi(qualities[0])
	if err != nil {
		return 100
	}
	return quality
}

func processAction(session webserver.Session) (interface{}, error) {
	var request = session.GetRequest()
	var parseErr = request.ParseMultipartForm(2097152)
	if parseErr != nil {
		return nil, parseErr
	}
	var sourceImageBytes, sourceImageErr = getImageBytes(
		request.MultipartForm,
		"source_image",
	)
	if sourceImageErr != nil {
		return nil, sourceImageErr
	}
	var targetImageBytes, targetImageErr = getImageBytes(
		request.MultipartForm,
		"target_image",
	)
	if targetImageErr != nil {
		return nil, targetImageErr
	}
	var namePrefix = getNamePrefix(request.MultipartForm)
	var reactorAPI = getReactorAPI(request.MultipartForm)
	var quality = getImageQuality(request.MultipartForm)
	var outImageBytes, outImageErr = processImage(
		sourceImageBytes[0],
		targetImageBytes,
		namePrefix,
		reactorAPI,
		quality,
	)
	if outImageErr != nil {
		return nil, outImageErr
	}
	var archiveBytes, archiveName, archiveErr = generateArchive(
		outImageBytes,
		namePrefix,
	)
	if archiveErr != nil {
		return nil, archiveErr
	}
	var responseWriter = session.GetResponseWriter()
	responseWriter.Header().Set(
		"Content-Type",
		"application/octet-stream",
	)
	responseWriter.Header().Set(
		"Content-Length",
		strconv.Itoa(len(archiveBytes)),
	)
	responseWriter.Header().Set(
		"Content-Disposition",
		fmt.Sprint("attachment;filename=", archiveName),
	)
	responseWriter.WriteHeader(http.StatusOK)
	responseWriter.Write(archiveBytes)
	return webserver.SkipResponseHandling()
}
