package main

import (
	"bytes"
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

func (customization *myCustomization) Routes() []webserver.Route {
	return []webserver.Route{
		{
			Endpoint: "Root",
			Method: http.MethodGet,
			Path: "/",
			ActionFunc: indexAction,
		},
		{
			Endpoint: "Process",
			Method: http.MethodPost,
			Path: "/",
			ActionFunc: processAction,
		},
	}
}

const INDEX_PAGE_CONTENT string = `<html>
  <header>
    <title>Uploader</title>
  </header>
  <body>
    <form method="POST" enctype="multipart/form-data">
      <label>Left image:&nbsp;</label>
      <input type="file" id="left_image" name="left_image" />
      <br />
      <label>Right image:&nbsp;</label>
      <input type="file" id="right_image" name="right_image" />
      <br />
      <label>Name prefix:&nbsp;</label>
      <input
        type="text"
        id="name_prefix"
        name="name_prefix"
        value="image-out"
      />
      <br />
      <label>Quality:&nbsp;</label>
      <input type="text" id="quality" name="quality" value="90" />
      <br />
      <input type="submit" />
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

func getImageBytes(multipartForm *multipart.Form, filename string) ([]byte, error) {
	var files, found = multipartForm.File[filename]
	if !found || len(files) < 1 {
		return nil, fmt.Errorf("unable to load file for %v", filename)
	}
	var imageFile, imageErr = files[0].Open()
	if imageErr != nil {
		return nil, imageErr
	}
	defer imageFile.Close()
	var buffer bytes.Buffer
	var _, bufferErr = buffer.ReadFrom(imageFile)
	if bufferErr != nil {
		return nil, bufferErr
	}
	return buffer.Bytes(), nil
}

func getImageQuality(multipartForm *multipart.Form) int {
	var qualities, found = multipartForm.Value["quality"]
	if !found || len(qualities) == 0 {
		return 90
	}
	var quality, err = strconv.Atoi(qualities[0])
	if err != nil {
		return 90
	}
	return quality
}

func getImageName(multipartForm *multipart.Form) string {
	var namePrefixes, found = multipartForm.Value["name_prefix"]
	if !found || len(namePrefixes) == 0 {
		namePrefixes = []string{"image-out"}
	}
	return fmt.Sprint(
		namePrefixes[0],
		"-",
		time.Now().Unix(),
		".jpg",
	)
}

func processAction(session webserver.Session) (interface{}, error) {
	var request = session.GetRequest()
	var parseErr = request.ParseMultipartForm(2097152)
	if parseErr != nil {
		return nil, parseErr
	}
	var leftImageBytes, leftImageErr = getImageBytes(request.MultipartForm, "left_image")
	if leftImageErr != nil {
		return nil, leftImageErr
	}
	var rightImageBytes, rightImageErr = getImageBytes(request.MultipartForm, "right_image")
	if rightImageErr != nil {
		return nil, rightImageErr
	}
	var quality = getImageQuality(request.MultipartForm)
	var outImageBytes, outImageErr = processImage(
		leftImageBytes,
		rightImageBytes,
		quality,
	)
	if outImageErr != nil {
		return nil, outImageErr
	}
	var outImageName = getImageName(request.MultipartForm)
	var responseWriter = session.GetResponseWriter()
	responseWriter.Header().Set("Content-Type", "application/octet-stream")
	responseWriter.Header().Set("Content-Disposition", fmt.Sprint("attachment;filename=", outImageName))
	responseWriter.WriteHeader(http.StatusOK)
	responseWriter.Write(outImageBytes)
	return webserver.SkipResponseHandling()
}
