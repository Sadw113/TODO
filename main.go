package main

import (
	"log"
	"todo/internal/app"
)

func main() {
	s := app.New()

	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}
