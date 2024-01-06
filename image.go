package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"time"
)

const IMAGE_PREFIX string = "data:image/png;base64,"

type reactorRequest struct {
	SourceImage  string `json:"source_image"`
	TargetImage  string `json:"target_image"`
	FaceRestorer string `json:"face_restorer"`
	GenderSource int    `json:"gender_source"`
	GenderTarget int    `json:"gender_target"`
	Device       string `json:"device"`
	MaskFace     int    `json:"mask_face"`
}

type reactorResponse struct {
	Image string `json:"image"`
}

type imageBytes struct {
	bytes []byte
	name  string
}

func flipImage(imageBytes []byte) ([]byte, error) {
	var reader = bytes.NewReader(imageBytes)
	var decodedImage, _, decodeErr = image.Decode(reader)
	if decodeErr != nil {
		return nil, decodeErr
	}
	var bounds = decodedImage.Bounds().Max
	var rect = image.Rect(0, 0, bounds.X, bounds.Y)
	var outputImage = image.NewNRGBA(rect)
	for y := 0; y < bounds.Y; y++ {
		for x := 0; x < bounds.X; x++ {
			outputImage.Set(
				x,
				y,
				decodedImage.At(
					bounds.X-x,
					y,
				),
			)
		}
	}
	var buffer bytes.Buffer
	var writer = bufio.NewWriter(&buffer)
	var pngErr = png.Encode(writer, outputImage)
	if pngErr != nil {
		return nil, pngErr
	}
	return buffer.Bytes(), nil
}

func getImageName(namePrefix string) string {
	var now = time.Now()
	return fmt.Sprintf(
		"%v_%v_%09d.png",
		namePrefix,
		now.Format("20060102_150405"),
		now.Nanosecond(),
	)
}

func processImage(
	sourceImageByte imageBytes,
	targetImageBytes []imageBytes,
	namePrefix string,
) ([]imageBytes, error) {
	var count = len(targetImageBytes)
	var allBytes = make([]imageBytes, 0, count)
	var srcImage = IMAGE_PREFIX + base64.StdEncoding.EncodeToString(
		sourceImageByte.bytes,
	)
	for i := 0; i < count; i++ {
		var tarImage = IMAGE_PREFIX + base64.StdEncoding.EncodeToString(
			targetImageBytes[i].bytes,
		)
		var content, contentError = json.Marshal(
			reactorRequest{
				SourceImage:  srcImage,
				TargetImage:  tarImage,
				FaceRestorer: "CodeFormer",
				GenderSource: 1,
				GenderTarget: 1,
				Device:       "CUDA",
				MaskFace:     1,
			},
		)
		if contentError != nil {
			return nil, contentError
		}
		var body = bytes.NewReader(content)
		var request, requestError = http.NewRequest(
			http.MethodPost,
			"http://localhost:7860/reactor/image",
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
		var buffer = &bytes.Buffer{}
		buffer.ReadFrom(response.Body)
		var respImg reactorResponse
		var respImgError = json.Unmarshal(buffer.Bytes(), &respImg)
		if respImgError != nil {
			return nil, respImgError
		}
		var resultImg, resultImgError = base64.StdEncoding.DecodeString(
			respImg.Image,
		)
		if resultImgError != nil {
			return nil, resultImgError
		}
		var flipImg, flipImgError = flipImage(resultImg)
		if flipImgError != nil {
			return nil, flipImgError
		}
		allBytes = append(allBytes, imageBytes{
			bytes: flipImg,
			name:  getImageName(namePrefix),
		})
	}
	return allBytes, nil
}
