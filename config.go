package main

import (
	"os"
)

type Config struct {
	Loglevel string

	AccessKey  string
	SecretKey  string
	BucketName string
	Region     string
	Endpoint   string

	Port    string
	BaseURL string
}

func LoadConfig() *Config {
	config := Config{
		Loglevel:   getEnv("loglevel", "INFO"),
		AccessKey:  os.Getenv("access_key"),
		SecretKey:  os.Getenv("secret_key"),
		BucketName: os.Getenv("bucket_name"),
		Region:     os.Getenv("region"),
		Endpoint:   os.Getenv("endpoint"),
		Port:       os.Getenv("port"),
		BaseURL:    os.Getenv("baseurl"),
	}

	Logoutput("Using Environment Variables", "info_force")
	return &config
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
