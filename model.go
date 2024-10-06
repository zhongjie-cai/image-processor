package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"

	webserver "github.com/zhongjie-cai/web-server"
)

type faceModelRequest struct {
	SourceImages  []string `json:"source_images"`
	Name          string   `json:"name"`
	ComputeMethod int      `json:"compute_method"`
}

func generateFaceModel(
	faceImageBytes []imageBytes,
	reactorAPI string,
) (*imageBytes, error) {
	var faceImages = make([]string, 0)
	for _, faceImageItem := range faceImageBytes {
		var faceImage = IMAGE_PREFIX + base64.StdEncoding.EncodeToString(
			faceImageItem.bytes,
		)
		faceImages = append(faceImages, faceImage)
	}
	var content, contentError = json.Marshal(
		faceModelRequest{
			SourceImages:  faceImages,
			Name:          "origin",
			ComputeMethod: 0,
		},
	)
	if contentError != nil {
		return nil, contentError
	}
	var body = bytes.NewReader(content)
	var request, requestError = http.NewRequest(
		http.MethodPost,
		reactorAPI,
		body,
	)
	if requestError != nil {
		return nil, requestError
	}
	var response, responseError = http.DefaultClient.Do(request)
	if responseError != nil {
		return nil, responseError
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("wrong response [%d]: {%s}", response.StatusCode, response.Status)
	}
	return nil, nil
}

func modelAction(session webserver.Session) (interface{}, error) {
	var request = session.GetRequest()
	var parseErr = request.ParseMultipartForm(2097152)
	if parseErr != nil {
		return nil, parseErr
	}
	var faceImageBytes, faceImageErr = getImageBytes(
		request.MultipartForm,
		"face_image",
	)
	if faceImageErr != nil {
		return nil, faceImageErr
	}
	if len(faceImageBytes) == 0 {
		var fileBytes, fileErr = os.ReadFile("origin.jpg")
		if fileErr != nil {
			return nil, fileErr
		}
		faceImageBytes = []imageBytes{
			{
				bytes: fileBytes,
				name: "origin.jpg",
			},
		}
	}
	var reactorAPI = getReactorAPI(request.MultipartForm)
	var faceModelBytes, faceModelErr = generateFaceModel(faceImageBytes, reactorAPI)
	if faceModelErr != nil {
		return nil, faceModelErr
	}
	var responseWriter = session.GetResponseWriter()
	responseWriter.Header().Set(
		"Content-Type",
		"application/octet-stream",
	)
	responseWriter.Header().Set(
		"Content-Length",
		strconv.Itoa(len(faceModelBytes.bytes)),
	)
	responseWriter.Header().Set(
		"Content-Disposition",
		fmt.Sprint("attachment;filename=", faceModelBytes.name),
	)
	responseWriter.WriteHeader(http.StatusOK)
	responseWriter.Write(faceModelBytes.bytes)
	return webserver.SkipResponseHandling()
}
