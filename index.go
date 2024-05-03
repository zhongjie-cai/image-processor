package main

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	webserver "github.com/zhongjie-cai/web-server"
)

const INDEX_PAGE_CONTENT string = `<html>
  <header>
    <title>Uploader v` + APP_VERSION + `</title>
	<style>
      html * {
        font-size: 8px;
      }
    </style>
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
      <label>Batches:&nbsp;</label>
      <input type="text" id="batches"
        name="batches" value="1" />
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

func getListOfProgressesHtml() string {
	var progresses = []*progress{}
	statusListLock.RLock()
	for _, progress := range statusList {
		progresses = append(progresses, progress)
	}
	statusListLock.RUnlock()
	sort.SliceStable(
		progresses,
		func(i, j int) bool {
			return progresses[i].counter < progresses[j].counter
		},
	)
	var builder strings.Builder
	for _, entry := range progresses {
		if entry.file != "" {
			if _, err := os.Stat(entry.file); err != nil {
				statusListLock.Lock()
				delete(statusList, entry.counter)
				statusListLock.Unlock()
			} else {
				builder.WriteString(
					fmt.Sprintf(
						"<p>%04d&nbsp;-&nbsp;%s<br /><a href=\".\\dl\\%d\">Download Only</a>&nbsp;&nbsp;-&nbsp;&nbsp;<a href=\".\\dnd\\%d\">Download & Delete</a></p>",
						entry.counter,
						entry.file,
						entry.counter,
						entry.counter,
					),
				)
			}
		} else {
			builder.WriteString(
				fmt.Sprintf(
					"<p>%04d - In progress ( %d / %d )</p>",
					entry.counter,
					entry.current,
					entry.total,
				),
			)
		}
	}
	if builder.Len() == 0 {
		return "No .error.log or .cache.zip found locally."
	}
	return builder.String()
}

func indexAction(session webserver.Session) (interface{}, error) {
	var ipAddresses = getServerIPsHtml(session)
	var listOfFiles = getListOfProgressesHtml()
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
