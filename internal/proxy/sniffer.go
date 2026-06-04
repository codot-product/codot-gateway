package proxy

import (
	"bytes"
	"io"
	"log"
	"net/http"
)

// InterceptAndSniff clones the payload bytes safely
func InterceptAndSniff(req *http.Request, analyzerFunc func([]byte)) {
	if req.Body == nil {
		return
	}

	// Use an io.TeeReader structure clone to avoid disrupting the request thread data
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		log.Printf("[ERROR] Failed reading incoming stream payload: %v", err)
		return
	}

	// Restore original buffer states for the proxy pipeline
	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Fire an un-blocking go-routine thread out to process metrics
	go analyzerFunc(bodyBytes)
}
