package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

var httpClient = &http.Client{Timeout: 30 * time.Second}

// WebhookPayload is the JSON body POSTed to the webhook URL.
type WebhookPayload struct {
	Source   string          `json:"source"`
	Item     json.RawMessage `json:"item"`
	PolledAt string          `json:"polled_at"`
}

// DeliverWebhook POSTs a webhook payload. Retries once on 5xx after 2s delay.
func DeliverWebhook(url, secret string, payload WebhookPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	// First attempt
	status, respBody, err := doPost(url, secret, body)
	if err != nil {
		return err
	}
	if status >= 200 && status < 300 {
		return nil
	}
	if status >= 400 && status < 500 {
		return fmt.Errorf("webhook returned %d: %s", status, respBody)
	}

	// 5xx: retry once after 2s
	log.Printf("Webhook returned %d, retrying in 2s...", status)
	time.Sleep(2 * time.Second)

	status, respBody, err = doPost(url, secret, body)
	if err != nil {
		return err
	}
	if status >= 200 && status < 300 {
		return nil
	}
	return fmt.Errorf("webhook returned %d after retry: %s", status, respBody)
}

func doPost(url, secret string, body []byte) (int, string, error) {
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return 0, "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if secret != "" {
		req.Header.Set("X-Webhook-Secret", secret)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	return resp.StatusCode, string(respBody), nil
}
