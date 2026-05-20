package main

import (
	_ "embed"
	"fmt"
	"net/http"

	"github.com/ekzyis/lntorch/server"
)

//go:embed banner.txt
var banner string

func main() {
	fmt.Println(banner)
	s := server.New()
	fmt.Println("Server running on http://localhost:8080")
	http.ListenAndServe(":8080", s)
}
