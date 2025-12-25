package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

const url = "https://gamma-api.polymarket.com/public-search"

func main() {
	client := &http.Client{Timeout: 20 * time.Second}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		log.Fatalf("build request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		log.Fatalf("polymarket response %s: %s", resp.Status, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("read body: %v", err)
	}

	fmt.Println(string(body))
}
