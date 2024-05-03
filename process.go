package main

import (
	"bytes"
	"fmt"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	webserver "github.com/zhongjie-cai/web-server"
)

type item struct {
	sourceImageBytes []imageBytes
	targetImageBytes []imageBytes
	namePrefix       string
	reactorAPI       string
	quality          int
	batches          int
	session          webserver.SessionLogging
}

var queue = make(chan item, 64)

type progress struct {
	total   int
	current int
	file    string
	counter int
}

var statusList = map[int]*progress{}

var statusListLock = sync.RWMutex{}

func getImageBytes(multipartForm *multipart.Form, filename string) ([]imageBytes, error) {
	var files, found = multipartForm.File[filename]
	if !found || len(files) < 1 {
		return nil, nil
	}
	var allBytes = make([]imageBytes, 0, len(files))
	for _, file := range files {
		var imageFile, imageErr = file.Open()
		if imageErr != nil {
			return nil, imageErr
		}
		defer imageFile.Close()
		var buffer bytes.Buffer
		var _, bufferErr = buffer.ReadFrom(imageFile)
		if bufferErr != nil {
			return nil, bufferErr
		}
		allBytes = append(allBytes, imageBytes{
			bytes: buffer.Bytes(),
			name:  file.Filename,
		})
	}
	return allBytes, nil
}

func getNamePrefix(multipartForm *multipart.Form) string {
	var namePrefixes, found = multipartForm.Value["name_prefix"]
	if !found || len(namePrefixes) == 0 {
		namePrefixes = []string{"IMG"}
	}
	return namePrefixes[0]
}

func getReactorAPI(multipartForm *multipart.Form) string {
	var reactorAPI, found = multipartForm.Value["reactor_api"]
	if !found || len(reactorAPI) == 0 {
		reactorAPI = []string{"http://localhost:7860/reactor/image"}
	}
	return reactorAPI[0]
}

func getImageQuality(multipartForm *multipart.Form) int {
	var qualities, found = multipartForm.Value["quality"]
	if !found || len(qualities) == 0 {
		return 100
	}
	var quality, err = strconv.Atoi(qualities[0])
	if err != nil {
		return 100
	}
	return quality
}

func getSplitBatches(multipartForm *multipart.Form) int {
	var batches, found = multipartForm.Value["batches"]
	if !found || len(batches) == 0 {
		return 1
	}
	var batch, err = strconv.Atoi(batches[0])
	if err != nil {
		return 1
	}
	return batch
}

func processBatch(
	counter int,
	batchItem item,
	start int,
	end int,
	session webserver.SessionLogging,
) {
	if end > len(batchItem.targetImageBytes) {
		end = len(batchItem.targetImageBytes)
	}
	var progress = &progress{
		total:   end-start,
		current: 0,
		counter: counter,
	}
	statusListLock.Lock()
	statusList[counter] = progress
	statusListLock.Unlock()
	session.LogMethodLogic(
		webserver.LogLevelInfo,
		"process",
		"processBatch",
		"Start processing item no.%04d - batch from %d to %d: %s",
		counter,
		start,
		end,
		batchItem.namePrefix,
	)
	var outImageBytes = processImage(
		batchItem.sourceImageBytes[0],
		batchItem.targetImageBytes[start:end],
		batchItem.namePrefix,
		batchItem.reactorAPI,
		batchItem.quality,
		progress,
	)
	var archiveErr = writeArchive(
		outImageBytes,
		batchItem.namePrefix,
		progress,
	)
	if archiveErr != nil {
		writeErrorLog(
			batchItem.namePrefix,
			archiveErr,
			progress,
		)
	}
	session.LogMethodLogic(
		webserver.LogLevelInfo,
		"process",
		"processBatch",
		"Done processing item no.%04d - batch from %d to %d: %s",
		counter,
		start,
		end,
		batchItem.namePrefix,
	)
}

func initCounter() int {
	var allEntries, allEntriesError = os.ReadDir(".")
	if allEntriesError != nil {
		fmt.Print(
			"Unable to read working directory entries: ",
			allEntriesError.Error(),
		)
		return 0
	}
	var counter = 0
	for _, entry := range allEntries {
		var entryName = entry.Name()
		if strings.HasSuffix(entryName, ".error.log") ||
			strings.HasSuffix(entryName, ".cache.zip") {
			counter++
			statusListLock.Lock()
			statusList[counter] = &progress{
				file:    entryName,
				counter: counter,
			}
			statusListLock.Unlock()
		}
	}
	return counter
}

func doProcessing() {
	var counter = initCounter()
	for item := range queue {
		var count = float64(len(item.targetImageBytes))
		var size = int(math.Ceil(count / float64(item.batches)))
		for i := 0; i < item.batches; i++ {
			counter++
			go processBatch(
				counter,
				item,
				i * size,
				i * size + size,
				item.session,
			)
		}
	}
}

func processAction(session webserver.Session) (interface{}, error) {
	var request = session.GetRequest()
	var parseErr = request.ParseMultipartForm(2097152)
	if parseErr != nil {
		return nil, parseErr
	}
	var sourceImageBytes, sourceImageErr = getImageBytes(
		request.MultipartForm,
		"source_image",
	)
	if sourceImageErr != nil {
		return nil, sourceImageErr
	}
	if len(sourceImageBytes) == 0 {
		var fileBytes, fileErr = os.ReadFile("origin.jpg")
		if fileErr != nil {
			return nil, fileErr
		}
		sourceImageBytes = []imageBytes{
			{
				bytes: fileBytes,
				name: "origin.jpg",
			},
		}
	}
	var targetImageBytes, targetImageErr = getImageBytes(
		request.MultipartForm,
		"target_image",
	)
	if targetImageErr != nil {
		return nil, targetImageErr
	}
	var namePrefix = getNamePrefix(request.MultipartForm)
	var reactorAPI = getReactorAPI(request.MultipartForm)
	var quality = getImageQuality(request.MultipartForm)
	var batches = getSplitBatches(request.MultipartForm)
	if len(targetImageBytes) == 1 {
		var outImageBytes = processImage(
			sourceImageBytes[0],
			targetImageBytes,
			namePrefix,
			reactorAPI,
			quality,
			nil,
		)
		var responseWriter = session.GetResponseWriter()
		responseWriter.Header().Set(
			"Content-Type",
			"application/octet-stream",
		)
		responseWriter.Header().Set(
			"Content-Length",
			strconv.Itoa(len(outImageBytes[0].bytes)),
		)
		responseWriter.Header().Set(
			"Content-Disposition",
			fmt.Sprint("attachment;filename=", outImageBytes[0].name),
		)
		responseWriter.WriteHeader(http.StatusOK)
		responseWriter.Write(outImageBytes[0].bytes)
		return webserver.SkipResponseHandling()
	} else {
		queue <- item{
			sourceImageBytes,
			targetImageBytes,
			namePrefix,
			reactorAPI,
			quality,
			batches,
			session,
		}
		var responseWriter = session.GetResponseWriter()
		responseWriter.WriteHeader(http.StatusNoContent)
		return webserver.SkipResponseHandling()
	}
}
