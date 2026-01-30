package main

import (
    "net/http"
    "github.com/G1D0/Api-Gateway/internal/proxy"
    "github.com/G1D0/Api-Gateway/internal/lb"
	"log"
)

func main() {
	backends := []string{"http://localhost:8080", "http://localhost:8081", "http://localhost:8082"}
	balancer := lb.NewRoundRobin(backends)
	p := proxy.NewProxy(balancer)
    // 2. Start server: http.ListenAndServe(":9000", p)
	log.Println("Proxy listening on :9000")
    err := http.ListenAndServe(":9000", p)
    if err != nil {
    log.Fatal(err)
    }
	

}
