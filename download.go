package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"

	webserver "github.com/zhongjie-cai/web-server"
)

func downloadAction(session webserver.Session) (interface{}, error) {
	var counter int
	var counterError = session.GetRequestParameter(
		"counter",
		&counter,
	)
	if counterError != nil {
		return nil, counterError
	}
	statusListLock.RLock()
	var progress, found = statusList[counter]
	statusListLock.RUnlock()
	if !found {
		return nil, fmt.Errorf("target not found for counter %d", counter)
	}
	var filename = progress.file
	var fileBytes, fileBytesError = os.ReadFile(filename)
	if fileBytesError != nil {
		return nil, fileBytesError
	}
	var responseWriter = session.GetResponseWriter()
	responseWriter.Header().Set(
		"Content-Type",
		"application/octet-stream",
	)
	responseWriter.Header().Set(
		"Content-Length",
		strconv.Itoa(len(fileBytes)),
	)
	responseWriter.Header().Set(
		"Content-Disposition",
		fmt.Sprint("attachment;filename=", filename),
	)
	responseWriter.WriteHeader(http.StatusOK)
	responseWriter.Write(fileBytes)
	return webserver.SkipResponseHandling()
}

func downloadAndDeleteAction(session webserver.Session) (interface{}, error) {
	var counter int
	var counterError = session.GetRequestParameter(
		"counter",
		&counter,
	)
	if counterError != nil {
		return nil, counterError
	}
	statusListLock.RLock()
	var progress, found = statusList[counter]
	statusListLock.RUnlock()
	if !found {
		return nil, fmt.Errorf("target not found for counter %d", counter)
	}
	var filename = progress.file
	var fileBytes, fileBytesError = os.ReadFile(filename)
	if fileBytesError != nil {
		return nil, fileBytesError
	}
	var deleteError = os.Remove(filename)
	if deleteError != nil {
		return nil, deleteError
	}
	statusListLock.Lock()
	delete(statusList, counter)
	statusListLock.Unlock()
	var responseWriter = session.GetResponseWriter()
	responseWriter.Header().Set(
		"Content-Type",
		"application/octet-stream",
	)
	responseWriter.Header().Set(
		"Content-Length",
		strconv.Itoa(len(fileBytes)),
	)
	responseWriter.Header().Set(
		"Content-Disposition",
		fmt.Sprint("attachment;filename=", filename),
	)
	responseWriter.WriteHeader(http.StatusOK)
	responseWriter.Write(fileBytes)
	return webserver.SkipResponseHandling()
}
