package main

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"

	"github.com/ekzyis/lntorch/db"
	"github.com/ekzyis/lntorch/server"
)

//go:embed banner.txt
var banner string

func main() {
	fmt.Println(banner)

	database, err := db.Open("lntorch.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	s := server.New(database)
	fmt.Println("Server running on http://localhost:8080")
	http.ListenAndServe(":8080", s)
}
