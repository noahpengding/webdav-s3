package main

import (
	"log"
	"net/http"
	_ "net/http/pprof"
)

var Cfg *Config

func main() {
	Cfg = LoadConfig()
	if Cfg.PprofAddr != "" {
		go startPprof(Cfg.PprofAddr)
	}

	Logoutput("Webdav server started", "info")
	Logoutput("Log level: "+Cfg.Loglevel, "info")
	webdav := NewWebDAVClient()
	Logoutput("Starting server on port "+Cfg.Port, "info")
	Logoutput("Base URL: "+Cfg.BaseURL, "info")

	mux := http.NewServeMux()
	mux.Handle("/", webdav)
	server := &http.Server{
		Addr:    ":" + Cfg.Port,
		Handler: mux,
	}
	if err := server.ListenAndServe(); err != nil {
		Logoutput("Server stopped: "+err.Error(), "error")
	}
}

func startPprof(addr string) {
	Logoutput("Starting pprof server on "+addr, "info")
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Println("ERROR: pprof server stopped: " + err.Error())
	}
}
