package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"time"
)

func getZipName(namePrefix string) string {
	return fmt.Sprint(
		namePrefix,
		"_",
		time.Now().Unix(),
	)
}

func writeErrorLog(
	namePrefix string,
	errData error,
) {
	os.WriteFile(
		fmt.Sprint(namePrefix, ".error.log"),
		[]byte(errData.Error()),
		0,
	)
}

func writeArchive(
	outImageBytes []imageBytes,
	namePrefix string,
) error {
	var buffer bytes.Buffer
	var zipName = getZipName(namePrefix)
	if len(outImageBytes) == 1 {
		return os.WriteFile(
			outImageBytes[0].name,
			outImageBytes[0].bytes, 
			0,
		)
	}
	var zipper = zip.NewWriter(&buffer)
	for _, imageBytes := range outImageBytes {
		var writer, err = zipper.Create(imageBytes.name)
		if err != nil {
			return err
		}
		writer.Write(imageBytes.bytes)
	}
	var err = zipper.Close()
	if err != nil {
		return err
	}
	return os.WriteFile(
		fmt.Sprint(zipName, ".cache.zip"),
		buffer.Bytes(),
		0,
	)
}
