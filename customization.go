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
		namePrefixes = []string{"image-out"}
	}
	return namePrefixes[0]
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
	var outImageBytes, outImageErr = processImage(
		sourceImageBytes[0],
		targetImageBytes,
		namePrefix,
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
