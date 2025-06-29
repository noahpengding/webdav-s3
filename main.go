package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

var Cfg *Config

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	var err error
	Cfg, err = LoadConfig()
	if err != nil {
		Logoutput("Unable to load config", "error")
		return
	}

	Logoutput("Webdav server started", "info")
	Logoutput("Log level: "+Cfg.Loglevel, "info")
	webdav := NewWebDAVClient()
	Logoutput("Starting server on port "+Cfg.Port, "info")
	Logoutput("Base URL: "+Cfg.BaseURL, "info")
	http.Handle("/", webdav)
	http.ListenAndServe(":"+Cfg.Port, nil)
}
