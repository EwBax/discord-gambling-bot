package main

import (
	"github.com/tkanos/gonfig"
	"log"
)

type Configuration struct {
	Token  string
	DbPath string
}

func GetConfig() Configuration {
	config := Configuration{}

	fileName := "config.json"

	if err := gonfig.GetConf(fileName, &config); err != nil {
		log.Fatal(err)
	}

	return config
}
