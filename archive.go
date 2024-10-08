package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"time"
)

func getZipName(namePrefix string, counter int) string {
	var timeNow = time.Now()
	return fmt.Sprint(
		namePrefix,
		"_",
		timeNow.Format("2006_01_02"),
		"_",
		fmt.Sprintf("%04d", counter),
		"_",
		timeNow.Format("15_04_05"),
		"_",
		timeNow.Nanosecond(),
	)
}

func writeErrorLog(
	namePrefix string,
	errData error,
	progress *progress,
) {
	var filename = fmt.Sprint(namePrefix, ".error.log")
	progress.file = filename
	os.WriteFile(
		filename,
		[]byte(errData.Error()),
		0,
	)
}

func writeArchive(
	outImageBytes []imageBytes,
	namePrefix string,
	progress *progress,
) error {
	var buffer bytes.Buffer
	var zipName = getZipName(namePrefix, progress.counter)
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
	var filename = fmt.Sprint(zipName, ".cache.zip")
	progress.file = filename
	return os.WriteFile(
		filename,
		buffer.Bytes(),
		0,
	)
}
