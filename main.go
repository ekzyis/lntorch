package main

import (
	_ "embed"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/ekzyis/lntorch/db"
	"github.com/ekzyis/lntorch/lightning"
	"github.com/ekzyis/lntorch/server"
)

//go:embed banner.txt
var banner string

func main() {
	if len(os.Args) > 1 && os.Args[1] == "pay" {
		runPay(os.Args[2:])
		return
	}

	autopay := flag.Bool("debug.autopay", false, "automatically pay every invoice (mock backend, dev only)")
	flag.Parse()

	fmt.Println(banner)

	database, err := db.Open("lntorch.db")
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()

	port := "4444"
	if flag.NArg() > 0 {
		port = flag.Arg(0)
	}

	s := server.New(database, lightning.NewMock(*autopay))
	fmt.Printf("lntorch running on port %s\n", port)
	http.ListenAndServe(":"+port, s)
}

// runPay is the `lntorch pay [port]` dev command: it asks a locally running
// server to settle its outstanding mock invoices. Port defaults to 4444.
func runPay(args []string) {
	port := "4444"
	if len(args) > 0 {
		port = args[0]
	}

	resp, err := http.Post("http://localhost:"+port+"/pay", "", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("pay failed: %s: %s", resp.Status, body)
	}
	io.Copy(os.Stdout, resp.Body)
	fmt.Println()
}
