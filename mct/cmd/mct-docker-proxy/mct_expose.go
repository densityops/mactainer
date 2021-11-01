package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
)

const (
	apiEndpoint     = "gateway.mct.mactainer"
	apiEndpointPort = "80"
)

type Expose struct {
	Local    string `json:"local"`
	Remote   string `json:"remote"`
	Protocol string `json:"protocol"`
}

type Unexpose struct {
	Local string `json:"local"`
}

func getAPIEndpoint() string {
	// read a envar this is required for testing
	endpoint := os.Getenv("PROXY_API_ADDR")
	if endpoint != "" {
		return endpoint
	}
	return apiEndpoint
}

func postRequest(ctx context.Context, url *url.URL, body interface{}) error {
	var buf io.ReadWriter
	client := &http.Client{}
	buf = new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(body); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url.String(), buf)
	if err != nil {
		return err
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return annotateResponseError(resp.Body)
	}
	return nil
}

func annotateResponseError(r io.Reader) error {
	b, err := ioutil.ReadAll(r)
	if err == nil && len(b) > 0 {
		return fmt.Errorf("gvisor-tap-vsock: %q", string(b))
	}
	return errors.New("gvisor-tap-vsock: could not read response")
}
