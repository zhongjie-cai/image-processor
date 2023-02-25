package main

import (
	"bufio"
	"bytes"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
)

func readImage(imageBytes []byte, needInvert bool) (image.Image, error) {
	var reader = bytes.NewReader(imageBytes)
	var decodedImage, _, decodeErr = image.Decode(reader)
	if decodeErr != nil {
		return nil, decodeErr
	}
	if !needInvert {
		return decodedImage, nil
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
					bounds.X - x,
					y,
				),
			)
		}
	}
	return outputImage, nil
}

func cropImage(input image.Image) image.Image {
	var bounds = input.Bounds().Max
	var x1=bounds.X/ 2
	var y1 = bounds.Y
	var sp = image.Pt(0, 0)
	var crops = image.Rect(0, 0, x1, y1)
	var output = image.NewNRGBA(crops)
	draw.Draw(output, crops, input, sp, draw.Src)
	return output
}

func mergeImage(left image.Image, right image.Image) image.Image {
	var boundsLeft = left.Bounds().Max
	var boundsRight = right.Bounds().Max
	var width = boundsLeft.X + boundsRight.X
	var height = boundsLeft.Y
	if height > boundsRight.Y {
		height = boundsRight.Y
	}
	var merge = image.Rect(0, 0, width, height)
	var output = image.NewNRGBA(merge)
	var sp = image.Pt(0, 0)
	var leftRect = image.Rect(0, 0, boundsLeft.X, boundsLeft.Y)
	draw.Draw(output, leftRect, left, sp, draw.Src)
	for y := 0; y < height; y++ {
		for x := boundsLeft.X; x < boundsLeft.X+boundsRight.X; x++ {
			output.Set(
				x,
				y,
				right.At(
					boundsRight.X - 1 - x + boundsLeft.X,
					y,
				),
			)
		}
	}
	return output
}

func writeImage(output image.Image, quality int, saveAsPNG bool) ([]byte, error) {
	var buffer bytes.Buffer
	var writer = bufio.NewWriter(&buffer)
	if saveAsPNG {
		var pngErr = png.Encode(writer, output)
		if pngErr != nil {
			return nil, pngErr
		}
		return buffer.Bytes(), nil
	}
	var jpegErr = jpeg.Encode(writer, output, &jpeg.Options{Quality: quality})
	if jpegErr != nil {
		return nil, jpegErr
	}
	return buffer.Bytes(), nil
}

func processImage(
	leftImageBytes [][]byte,
	leftWatermarkOnRight bool,
	rightImageBytes [][]byte,
	rightWatermarkOnRight bool,
	quality int,
	saveAsPNG bool,
) ([][]byte, error) {
	var count = len(leftImageBytes)
	if count > len(rightImageBytes) {
		count = len(rightImageBytes)
	}
	var allBytes = make([][]byte, 0, count)
	for i := 0; i < count; i++ {
		var leftImage, leftImageErr = readImage(
			leftImageBytes[i],
			!leftWatermarkOnRight,
		)
		if leftImageErr != nil {
			return nil, leftImageErr
		}
		var rightImage, rightImageErr = readImage(
			rightImageBytes[i],
			!rightWatermarkOnRight,
		)
		if rightImageErr != nil {
			return nil, rightImageErr
		}
		var imageOut = mergeImage(
			cropImage(leftImage),
			cropImage(rightImage),
		)
		var imageOutBytes, imageOutErr = writeImage(
			imageOut,
			quality,
			saveAsPNG,
		)
		if imageOutErr != nil {
			return nil, imageOutErr
		}
		allBytes = append(allBytes, imageOutBytes)
	}
	return allBytes, nil
}
