package main

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"os"

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

	port := "4444"
	if len(os.Args) > 1 {
		port = os.Args[1]
	}

	s := server.New(database)
	fmt.Printf("lntorch running on port %s\n", port)
	http.ListenAndServe(":"+port, s)
}
