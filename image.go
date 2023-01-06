package main

import (
	"bufio"
	"bytes"
	"image"
	"image/draw"
	"image/jpeg"
)

func readImage(imageBytes []byte) (image.Image, error) {
	var reader = bytes.NewReader(imageBytes)
	var decodedImage, _, decodeErr = image.Decode(reader)
	if decodeErr != nil {
		return nil, decodeErr
	}
	return decodedImage, nil
}

func cropImage(input image.Image, left bool) image.Image {
	var bounds = input.Bounds().Max
	var x0=0
	var y0=0
	var x1=bounds.X/ 2
	var y1 = bounds.Y
	if !left {
		x0 = bounds.X/2
		x1 = bounds.X
	}
	var sp = image.Pt(x0, 0)
	var crops = image.Rect(0, y0, x1-x0, y1)
	var output = image.NewNRGBA64(crops)
	draw.Draw(output, crops, input, sp, draw.Src)
	return output
}

func mergeImage(left image.Image, right image.Image) image.Image {
	var boundsLeft = left.Bounds().Max
	var boundsRight = right.Bounds().Max
	var width = boundsLeft.X + boundsRight.X
	var height = boundsLeft.Y
	if height < boundsRight.Y {
		height = boundsRight.Y
	}
	var merge = image.Rect(0, 0, width, height)
	var sp = image.Point{}
	var leftRect = image.Rect(0, 0, boundsLeft.X, boundsLeft.Y)
	var rightRect = image.Rect(boundsLeft.X, 0, boundsLeft.X+boundsRight.X, boundsRight.Y)
	var output = image.NewNRGBA64(merge)
	draw.Draw(output, leftRect, left, sp, draw.Src)
	draw.Draw(output, rightRect, right, sp, draw.Src)
	return output
}

func writeImage(output image.Image, quality int) ([]byte, error) {
	var buffer bytes.Buffer
	var writer = bufio.NewWriter(&buffer)
	var jpegErr = jpeg.Encode(writer, output, &jpeg.Options{Quality: quality})
	if jpegErr != nil {
		return nil, jpegErr
	}
	return buffer.Bytes(), nil
}

func processImage(leftImageBytes []byte, rightImageBytes []byte, quality int) ([]byte, error) {
	var leftImage, leftImageErr = readImage(leftImageBytes)
	if leftImageErr != nil {
		return nil, leftImageErr
	}
	var rightImage, rightImageErr = readImage(rightImageBytes)
	if rightImageErr != nil {
		return nil, rightImageErr
	}
	var imageOut = mergeImage(
		cropImage(leftImage, true),
		cropImage(rightImage, false),
	)
	var imageOutBytes, imageOutErr = writeImage(imageOut, quality)
	if imageOutErr != nil {
		return nil, imageOutErr
	}
	return imageOutBytes, nil
}