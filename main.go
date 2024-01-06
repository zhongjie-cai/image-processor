package main

import (
	webserver "github.com/zhongjie-cai/web-server"
)

const APP_VERSION string = `1.0.8`

func main() {
	var application = webserver.NewApplication(
		"ImageProcessor",
		":8080",
		APP_VERSION,
		&myCustomization{},
	)
	defer application.Stop()
	application.Start()
}
