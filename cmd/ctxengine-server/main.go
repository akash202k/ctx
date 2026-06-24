package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"ctxengine/internal/server"
)

func main() {
	port := flag.String("port", "8080", "port to listen on")
	flag.Parse()

	addr := ":" + *port
	fmt.Fprintf(os.Stdout, "Context Engine HTTP server listening on %s\n", addr)
	fmt.Fprintf(os.Stdout, "POST /select  — body: {task, repo_root, entry_point?, token_budget}\n")

	mux := server.NewServer()
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
