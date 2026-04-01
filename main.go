package main

import (
	"log"

	"search-api/internal/api"
)

func main() {
	if err := api.Run(); err != nil {
		log.Fatal(err)
	}
}
