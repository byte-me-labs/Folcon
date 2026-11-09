package main

import (
	"log"

	"github.com/egoist/mygo"

	"github.com/byte-me-labs/folcon/internal/app"
)

func main() {
	a := app.New()
	mygo.App.WhenReady(func() {
		a.Start()
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
