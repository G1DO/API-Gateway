package main

import (
    "net/http"
    "github.com/G1D0/Api-Gateway/internal/proxy"
	"log"
)

func main() {
    // 1. Create proxy: p := proxy.NewProxy("http://localhost:8080")
	p := proxy.NewProxy("http://localhost:8080")
    // 2. Start server: http.ListenAndServe(":9000", p)
	log.Println("Proxy listening on :9000")
    err := http.ListenAndServe(":9000", p)
    if err != nil {
    log.Fatal(err)
    }
	

}
