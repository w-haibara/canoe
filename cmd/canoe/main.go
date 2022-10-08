package main

import (
	"log"
	"github.com/w-haibara/canoe"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if err := canoe.Deploy(); err != nil {
		panic(err.Error())
	}
}
