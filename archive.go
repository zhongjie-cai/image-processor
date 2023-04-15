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
		"_",
		time.Now().Unix(),
	)
}

func generateArchive(
	outImageBytes []imageBytes,
	namePrefix string,
) ([]byte, string, error) {
	var buffer bytes.Buffer
	var zipName = getZipName(namePrefix)
	if len(outImageBytes) == 1 {
		return outImageBytes[0].bytes, outImageBytes[0].name, nil
	}
	var zipper = zip.NewWriter(&buffer)
	for _, imageBytes := range outImageBytes {
		var writer, err = zipper.Create(imageBytes.name)
		if err != nil {
			return nil, "", err
		}
		writer.Write(imageBytes.bytes)
	}
	var err = zipper.Close()
	if err != nil {
		return nil, "", err
	}
	return buffer.Bytes(),
		fmt.Sprint(zipName, ".zip"),
		nil
}
