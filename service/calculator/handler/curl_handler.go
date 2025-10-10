package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

type CurlHandler struct {
	BaseHandler
}

func NewCurlHandler() *CurlHandler {
	return &CurlHandler{}
}

func (h *CurlHandler) CanHandle(input string) bool {
	return strings.HasPrefix(input, "curl")
}

func (h *CurlHandler) Handle(input string) (string, error) {
	args := strings.Fields(input)[1:]
	if len(args) == 0 {
		return "", fmt.Errorf("URL required for curl")
	}

	url := args[0]
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %v", err)
	}

	for i := 1; i < len(args); i++ {
		if args[i] == "-H" && i+1 < len(args) {
			header := strings.Split(args[i+1], ":")
			if len(header) == 2 {
				req.Header.Add(strings.TrimSpace(header[0]), strings.TrimSpace(header[1]))
			}
			i++
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %v", err)
	}

	if len(body) > 500 {
		body = append(body[:500], []byte("... [truncated]")...)
	}

	return string(body), nil
}
