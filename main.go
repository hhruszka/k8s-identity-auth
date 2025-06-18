package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	host = "192.168.8.110"
	path = "/v1/auth/kubernetes/login"
	port = "8222"
)

type AuthRequest struct {
	Role                string `json:"role"`
	ServiceAccountToken string `json:"jwt"`
}

func newAuthRequest(role string, token string) *AuthRequest {
	return &AuthRequest{Role: role, ServiceAccountToken: token}
}

const ServiceTokePath = "/var/run/secrets/kubernetes.io/serviceaccount/token"

func readToken() string {
	token, err := os.ReadFile(ServiceTokePath)
	if err != nil {
		log.Fatal(err)
	}
	return string(token)
}

func createHttpClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
			MaxIdleConns:    10,
			IdleConnTimeout: 30 * time.Second,
		},
	}
}

func createRequest(ctx context.Context, serviceToken string) (*http.Request, error) {
	authRequest := newAuthRequest("testapp", string(serviceToken))
	log.Println(authRequest)

	buf, err := json.Marshal(authRequest)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("https://%s:%s%s", host, port, path), bytes.NewBuffer(buf))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")

	return req, nil
}

func authenticate(ctx context.Context, serviceToken string) error {

	httpClient := createHttpClient()
	req, err := createRequest(ctx, serviceToken)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	log.Println(resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var data interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return err
	}

	formattedJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// Print the formatted JSON to the console
	log.Println(string(formattedJSON))
	return nil
}

func main() {
	var sigChan = make(chan os.Signal, 1)
	defer close(sigChan)
	defer log.Println("App stopped")

	signal.Notify(sigChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)

	log.Println("App started")

	serviceToken := readToken()
	log.Println(serviceToken)

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := authenticate(context.Background(), serviceToken); err != nil {
				log.Println(err)
				return
			}
		case <-sigChan:
			return
		}
	}
}
