package main

import webserver "github.com/zhongjie-cai/web-server"

func main() {
	var application = webserver.NewApplication(
		"ImageProcessor",
		18605,
		"1.0.0",
		&myCustomization{},
	)
	defer application.Stop()
	application.Start()
}
