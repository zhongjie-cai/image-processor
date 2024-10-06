package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"net/http"
	"time"
)

const IMAGE_PREFIX string = "data:image/png;base64,"

type reactorRequest struct {
	TargetImage  string `json:"target_image"`
	FaceRestorer string `json:"face_restorer"`
	GenderSource int    `json:"gender_source"`
	GenderTarget int    `json:"gender_target"`
	Device       string `json:"device"`
	MaskFace     int    `json:"mask_face"`
	SelectSource int    `json:"select_source"`
	FaceModel    string `json:"face_model"`
	CodeFormerWeight float64 `json:"codeformer_weight"`
}

type reactorResponse struct {
	Image string `json:"image"`
}

type imageBytes struct {
	bytes []byte
	name  string
}

func flipImage(imageBytes []byte, quality int) ([]byte, error) {
	var reader = bytes.NewReader(imageBytes)
	var decodedImage, decodeErr = png.Decode(reader)
	if decodeErr != nil {
		return nil, decodeErr
	}
	var bounds = decodedImage.Bounds().Max
	var rect = image.Rect(0, 0, bounds.X, bounds.Y)
	var outputImage = image.NewRGBA(rect)
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
	var jpegErr = jpeg.Encode(
		writer,
		outputImage,
		&jpeg.Options{
			Quality: quality,
		},
	)
	if jpegErr != nil {
		return nil, jpegErr
	}
	return buffer.Bytes(), nil
}

func getImageName(namePrefix string) string {
	var now = time.Now()
	return fmt.Sprintf(
		"%v_%v_%09d.jpg",
		namePrefix,
		now.Format("20060102_150405"),
		now.Nanosecond(),
	)
}

func getErrorBytes(originalName string, errorData error) *imageBytes {
	return &imageBytes{
		name: fmt.Sprintf("%v.error.log", originalName),
		bytes: []byte(fmt.Sprintf("Failed processing file %v: %v", originalName, errorData.Error())),
	}
}

func processImage(
	targetImageBytes []imageBytes,
	namePrefix string,
	reactorAPI string,
	quality int,
	weight float64,
	progress *progress,
) []imageBytes {
	var count = len(targetImageBytes)
	var allBytes = make([]imageBytes, 0, count)
	for i := 0; i < count; i++ {
		if progress != nil {
			progress.current = i + 1
		}
		var originalName = targetImageBytes[i].name
		var tarImage = IMAGE_PREFIX + base64.StdEncoding.EncodeToString(
			targetImageBytes[i].bytes,
		)
		var content, contentError = json.Marshal(
			reactorRequest{
				TargetImage:  tarImage,
				FaceRestorer: "CodeFormer",
				Device:       "CUDA",
				MaskFace:     1,
				GenderSource: 1,
				GenderTarget: 1,
				CodeFormerWeight: weight,
				SelectSource: 1,
				FaceModel: "origin.safetensors",
			},
		)
		if contentError != nil {
			var errorBytes = getErrorBytes(originalName, contentError)
			allBytes = append(allBytes, *errorBytes)
			continue
		}
		var body = bytes.NewReader(content)
		var request, requestError = http.NewRequest(
			http.MethodPost,
			reactorAPI,
			body,
		)
		if requestError != nil {
			var errorBytes = getErrorBytes(originalName, requestError)
			allBytes = append(allBytes, *errorBytes)
			continue
		}
		var response, responseError = http.DefaultClient.Do(request)
		if responseError != nil {
			var errorBytes = getErrorBytes(originalName, responseError)
			allBytes = append(allBytes, *errorBytes)
			continue
		}
		if response.StatusCode != http.StatusOK {
			var errorBytes = getErrorBytes(originalName, fmt.Errorf("wrong response [%d]: {%s}", response.StatusCode, response.Status))
			allBytes = append(allBytes, *errorBytes)
			continue
		}
		var buffer = &bytes.Buffer{}
		buffer.ReadFrom(response.Body)
		var respImg reactorResponse
		var respImgError = json.Unmarshal(buffer.Bytes(), &respImg)
		if respImgError != nil {
			var errorBytes = getErrorBytes(originalName, respImgError)
			allBytes = append(allBytes, *errorBytes)
			continue
		}
		var resultImg, resultImgError = base64.StdEncoding.DecodeString(
			respImg.Image,
		)
		if resultImgError != nil {
			var errorBytes = getErrorBytes(originalName, respImgError)
			allBytes = append(allBytes, *errorBytes)
			continue
		}
		var flipImg, flipImgError = flipImage(resultImg, quality)
		if flipImgError != nil {
			var errorBytes = getErrorBytes(originalName, flipImgError)
			allBytes = append(allBytes, *errorBytes)
			continue
		}
		allBytes = append(allBytes, imageBytes{
			bytes: flipImg,
			name:  getImageName(namePrefix),
		})
	}
	return allBytes
}
