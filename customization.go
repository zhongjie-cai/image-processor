package main

import (
	"crypto/tls"
	"net/http"

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

func (customization *myCustomization) PostBootstrap() error {
    go doProcessing()
	return nil
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
            Path:       "/process",
            ActionFunc: processAction,
        },
        {
            Endpoint:   "Model",
            Method:     http.MethodPost,
            Path:       "/model",
            ActionFunc: modelAction,
        },
        {
            Endpoint:   "Download",
            Method:     http.MethodGet,
            Path:       "/dl/{counter}",
            ActionFunc: downloadAction,
            Parameters: map[string]webserver.ParameterType{
                "counter": webserver.ParameterTypeInteger,
            },
        },
        {
            Endpoint:   "Delete",
            Method:     http.MethodGet,
            Path:       "/dnd/{counter}",
            ActionFunc: downloadAndDeleteAction,
            Parameters: map[string]webserver.ParameterType{
                "counter": webserver.ParameterTypeInteger,
            },
        },
    }
}
