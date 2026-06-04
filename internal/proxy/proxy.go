package proxy

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
)

type GatewayProxy struct {
	ReverseProxy *httputil.ReverseProxy
	TargetURL    *url.URL
}

func NewGatewayProxy(upstreamTarget string) (*GatewayProxy, error) {
	target, err := url.Parse(upstreamTarget)
	if err != nil {
		return nil, err
	}

	rp := httputil.NewSingleHostReverseProxy(target)

	// Ensure underlying SSL configurations do not choke on custom local dev certificates
	rp.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: false},
	}

	gp := &GatewayProxy{
		ReverseProxy: rp,
		TargetURL:    target,
	}

	// Attach director rule chains by wrapping the default director
	originalDirector := rp.Director
	rp.Director = func(req *http.Request) {
		originalDirector(req)
		gp.modifyOutgoingRequest(req)
	}

	// Wire up the response interceptor hook
	rp.ModifyResponse = gp.interceptDownstreamResponse

	return gp, nil
}

func (gp *GatewayProxy) modifyOutgoingRequest(req *http.Request) {
	req.Header.Set("X-Forwarded-Host", req.Header.Get("Host"))
	req.Host = gp.TargetURL.Host

	// Grab your private secure cloud credentials (BYOK)
	// In the next phase, these will come dynamically from your SQLite store
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
}

func (gp *GatewayProxy) interceptDownstreamResponse(resp *http.Response) error {
	// Skip parsing if the upstream server fails
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	// Clone the response stream using an io.TeeReader to process data asynchronously
	respBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("[ERROR] Failed reading downstream response buffer: %v", err)
		return nil
	}

	// Restore the response body so the calling IDE receives its text data normally
	resp.Body = io.NopCloser(bytes.NewBuffer(respBodyBytes))

	// Track chunk streaming counts inside a separate concurrent runtime thread
	go analyzeResponseStream(respBodyBytes)

	return nil
}

func analyzeResponseStream(body []byte) {
	scanner := bufio.NewScanner(bytes.NewReader(body))
	tokenCount := 0

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			dataContent := strings.TrimPrefix(line, "data: ")
			if dataContent == "[DONE]" {
				break
			}
			
			// Increment mock token counts based on whitespace segments
			tokenCount++
		}
	}

	if tokenCount > 0 {
		log.Printf("[METRICS] Model completed execution stream. Generated ~%d response tokens.", tokenCount)
	}
}
