package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"time"
)

func getZipName(namePrefix string) string {
	return fmt.Sprint(
		namePrefix,
		"-",
		time.Now().Unix(),
	)
}

func getImageName(index int, namePrefix string, saveAsPNG bool) string {
	var suffix = ".jpg"
	if saveAsPNG {
		suffix = ".png"
	}
	return fmt.Sprint(
		namePrefix,
		"-",
		index,
		suffix,
	)
}

func generateArchive(
	outImageBytes [][]byte,
	namePrefix string,
	saveAsPNG bool,
) ([]byte, string, error) {
	var buffer bytes.Buffer
	var zipName = getZipName(namePrefix)
	if len(outImageBytes) == 1 {
		return outImageBytes[0], getImageName(0, zipName, saveAsPNG), nil
	}
	var zipper = zip.NewWriter(&buffer)
	for i, bytes := range outImageBytes {
		var imageName = getImageName(i, zipName, saveAsPNG)
		var writer, err = zipper.Create(imageName)
		if err != nil {
			return nil, "", err
		}
		writer.Write(bytes)
	}
	var err = zipper.Close()
	if err != nil {
		return nil, "", err
	}
	return buffer.Bytes(),
		fmt.Sprint(zipName, ".zip"),
		nil
}
