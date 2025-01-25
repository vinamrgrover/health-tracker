package main

import (
	h "health/api"
	"log"
)

func main() {
	r := h.GetRouter()
	log.Fatal(r.Run(":8080"))
}
