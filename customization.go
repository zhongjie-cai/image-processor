package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	webserver "github.com/zhongjie-cai/web-server"
)

type myCustomization struct {
    webserver.DefaultCustomization
}

func (customization *myCustomization) ServerCert() *tls.Certificate {
    var cert, err = tls.LoadX509KeyPair("/data/v2ray.crt", "/data/v2ray.key")
    if err != nil {
        return nil
    }
    return &cert
}

func (customization *myCustomization) Routes() []webserver.Route {
    return []webserver.Route{
        {
            Endpoint:   "Root",
            Method:     http.MethodGet,
            Path:       "/",
            ActionFunc: indexAction,
        },
        {
            Endpoint:   "Process",
            Method:     http.MethodPost,
            Path:       "/",
            ActionFunc: processAction,
        },
        {
            Endpoint:   "Download",
            Method:     http.MethodGet,
            Path:       "/{fileName}",
            ActionFunc: downloadAction,
            Parameters: map[string]webserver.ParameterType{
                "fileName": webserver.ParameterTypeAnything,
            },
        },
    }
}

const INDEX_PAGE_CONTENT string = `<html>
  <header>
    <title>Uploader v` + APP_VERSION + `</title>
  </header>
  <body>
    <form method="POST" enctype="multipart/form-data">
      <label>App Version = ` + APP_VERSION + `</label>
      <br />
      <div>
        %s
      </div>
      <br />
      <label>Source Image:&nbsp;</label>
      <input type="file" id="source_image" name="source_image" />
      <br />
      <label>Target image:&nbsp;</label>
      <input type="file" id="target_image" name="target_image"
        multiple="multiple" />
      <br />
      <label>Name prefix:&nbsp;</label>
      <input type="text" id="name_prefix"
        name="name_prefix" value="IMG" />
      <br />
      <label>Reactor API:&nbsp;</label>
      <input type="text" id="reactor_api"
        name="reactor_api" value="http://localhost:7860/reactor/image" />
      <br />
      <label>Quality:&nbsp;</label>
      <input type="text" id="quality"
        name="quality" value="100" />
      <br />
      <input type="submit" />
      <br />
      <br />
      <div>
        %s
      </div>
    </form>
  </body>
</html>`

func getServerIPsHtml(session webserver.Session) string {
    addresses, err := net.InterfaceAddrs()
    if err != nil {
        session.LogMethodLogic(
            webserver.LogLevelError,
            "Index",
            "getServerIPsHtml",
            "Failed to get server IP address: %v\n",
            err,
        )
        return "Failed to get server IP address."
    }
    var builder strings.Builder
    var counter = 0
    for _, addr := range addresses {
        if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            if ipnet.IP.To4() != nil && ipnet.IP.IsPrivate() {
                counter++
                builder.WriteString(
                    fmt.Sprintf(
                        "IP Address: %v<br />",
                        ipnet.IP,
                    ),
                )
            }
        }
    }
    if counter == 0 {
        return "No IP address found locally."
    }
    return builder.String()
}

func getListOfFilesHtml() string {
    var allEntries, allEntriesError = os.ReadDir(".")
    if allEntriesError != nil {
        return fmt.Sprint(
            "Unable to read working directory entries: ",
            allEntriesError.Error(),
        )
    }
    var builder strings.Builder
    var counter = 0
    for _, entry := range allEntries {
        var entryName = entry.Name()
        if strings.HasSuffix(entryName, ".error.log") ||
            strings.HasSuffix(entryName, ".cache.zip") {
            counter++
            builder.WriteString(
                fmt.Sprintf(
                    "<a href=\".\\%s\">%04d - %s</a><br />",
                    entryName,
                    counter,
                    entryName,
                ),
            )
        }
    }
    if counter == 0 {
        return "No .error.log or .cache.zip found locally."
    }
    return builder.String()
}

func indexAction(session webserver.Session) (interface{}, error) {
	var ipAddresses = getServerIPsHtml(session)
    var listOfFiles = getListOfFilesHtml()
    var pageContent = fmt.Sprintf(INDEX_PAGE_CONTENT, ipAddresses, listOfFiles)
    var request = session.GetRequest()
    var responseWriter = session.GetResponseWriter()
    http.ServeContent(
        responseWriter,
        request,
        "index.html",
        time.Now(),
        strings.NewReader(
            pageContent,
        ),
    )
    return webserver.SkipResponseHandling()
}

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

func doProcessing(
    sourceImageBytes []imageBytes,
    targetImageBytes []imageBytes,
    namePrefix string,
    reactorAPI string,
    quality int,
) {
    var outImageBytes, outImageErr = processImage(
        sourceImageBytes[0],
        targetImageBytes,
        namePrefix,
        reactorAPI,
        quality,
    )
    if outImageErr != nil {
        writeErrorLog(
            namePrefix,
            outImageErr,
        )
        return 
    }
    var archiveErr = writeArchive(
        outImageBytes,
        namePrefix,
    )
    if archiveErr != nil {
        writeErrorLog(
            namePrefix,
            outImageErr,
        )
        return 
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
	if len(targetImageBytes) == 1 {
		var outImageBytes, outImageErr = processImage(
			sourceImageBytes[0],
			targetImageBytes,
			namePrefix,
			reactorAPI,
			quality,
		)
		if outImageErr != nil {
			return nil, outImageErr
		}
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
		go doProcessing(
			sourceImageBytes,
			targetImageBytes,
			namePrefix,
			reactorAPI,
			quality,
		)
		var responseWriter = session.GetResponseWriter()
		responseWriter.WriteHeader(http.StatusNoContent)
		return webserver.SkipResponseHandling()
	}
}

func downloadAction(session webserver.Session) (interface{}, error) {
    var fileName string
    var fileNameError = session.GetRequestParameter(
        "fileName",
        &fileName,
    )
    if fileNameError != nil {
        return nil, fileNameError
    }
    var fileBytes, fileBytesError = os.ReadFile(fileName)
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
        fmt.Sprint("attachment;filename=", fileName),
    )
    responseWriter.WriteHeader(http.StatusOK)
    responseWriter.Write(fileBytes)
    return webserver.SkipResponseHandling()
}
