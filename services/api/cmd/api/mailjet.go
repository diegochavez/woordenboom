package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const mailjetSendURL = "https://api.mailjet.com/v3.1/send"

var mailjetHTTPClient = &http.Client{Timeout: 10 * time.Second}

type mailjetAddress struct {
	Email string `json:"Email"`
}

type mailjetMessage struct {
	From     mailjetAddress   `json:"From"`
	To       []mailjetAddress `json:"To"`
	Subject  string           `json:"Subject"`
	TextPart string           `json:"TextPart"`
}

func sendMailjetLoginCode(ctx context.Context, client *http.Client, endpoint, apiKey, secretKey, from, to, code string) error {
	payload := struct {
		Messages []mailjetMessage `json:"Messages"`
	}{Messages: []mailjetMessage{{
		From:     mailjetAddress{Email: from},
		To:       []mailjetAddress{{Email: to}},
		Subject:  "Your Woordenboom login code",
		TextPart: "Your Woordenboom code is " + code + ". It expires in 10 minutes.",
	}}}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode Mailjet request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create Mailjet request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(apiKey, secretKey)

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Mailjet request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("Mailjet rejected email with HTTP %d", resp.StatusCode)
	}
	var result struct {
		Messages []struct {
			Status string `json:"Status"`
		} `json:"Messages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode Mailjet response: %w", err)
	}
	if len(result.Messages) != 1 || result.Messages[0].Status != "success" {
		return fmt.Errorf("Mailjet did not accept the login email")
	}
	return nil
}
