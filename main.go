package main

import webserver "github.com/zhongjie-cai/web-server"

const APP_VERSION string = `1.0.5`

func main() {
	var application = webserver.NewApplication(
		"ImageProcessor",
		18605,
		APP_VERSION,
		&myCustomization{},
	)
	defer application.Stop()
	application.Start()
}
