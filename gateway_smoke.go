//go:build ignore

package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	payload := []byte(`{
		"model": "gpt-4o",
		"messages": [
			{
				"role": "user",
				"content": "Write a clean markdown README layout template"
			}
		]
	}`)

	resp, err := http.Post(
		"http://localhost:8080/v1/chat/completions",
		"application/json",
		bytes.NewBuffer(payload),
	)
	if err != nil {
		log.Fatalf("[ERROR] Request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("[ERROR] Failed to read response body: %v", err)
	}

	fmt.Printf("Status Code : %d\n", resp.StatusCode)
	fmt.Printf("Body        :\n%s\n", string(body))
}
