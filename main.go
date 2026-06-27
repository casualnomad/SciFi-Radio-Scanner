package main

import (
	"Radio-Scanner/llm"
	"Radio-Scanner/radio"
	"Radio-Scanner/web"
	"fmt"
	"net/http"
)

func main() {

	client := llm.New("http://localhost:2276", "local-model")
	world := radio.NewWorld()

	server := web.New(client, world)
	server.Routes()

	fmt.Println("Scanner Running on Http://localhost:3000")
	http.ListenAndServe(":3000", nil)

}
