package main

import (
	"log"
	"strings"
)

func Logoutput(message string, level string) {
	switch level {
		case "info_force":
			log.Println("INFO: " + message)
		case "error_force":
			log.Fatalln("ERROR: " + message)
		case "debug":
			if strings.ToLower(Cfg.Loglevel) == "debug" {
				log.Println("DEBUG: " + message)
			}else{
				return
			}
		case "info":
			if strings.ToLower(Cfg.Loglevel) == "debug" || strings.ToLower(Cfg.Loglevel) == "info" {
				log.Println("INFO: " + message)
			}else{
				return
			}
		case "error":
			log.Fatalln("ERROR: " + message)
		default:
			return
	}
}
