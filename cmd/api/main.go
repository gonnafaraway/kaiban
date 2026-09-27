package main

import (
	"log"

	"kaiban/internal/api/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
