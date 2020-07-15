package main

import (
	"net/http"

	"github.com/nebril/image-pusher/p"
)

func main() {
	http.HandleFunc("/move", p.MoveImage)
	http.ListenAndServe(":8080", nil)
}
