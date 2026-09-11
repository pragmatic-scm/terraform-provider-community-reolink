// SPDX-License-Identifier: MPL-2.0

package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Config struct {
	URL      string
	Username string
	Password string
	Insecure bool
}

type Client struct {
	baseURL  string
	username string
	password string
	http     *http.Client
	mu       sync.Mutex
	token    string
	expires  time.Time
	apiPath  string
}

type Envelope struct {
	Cmd     string          `json:"cmd"`
	Code    int             `json:"code"`
	Error   *APIError       `json:"error,omitempty"`
	Initial json.RawMessage `json:"initial,omitempty"`
	Range   json.RawMessage `json:"range,omitempty"`
	Value   json.RawMessage `json:"value,omitempty"`
}

type APIError struct {
	RspCode int    `json:"rspCode"`
	Detail  string `json:"detail"`
}

type loginValue struct {
	Token struct {
		Name  string `json:"name"`
		Lease int    `json:"leaseTime"`
	} `json:"Token"`
}

var querySecretPattern = regexp.MustCompile(`(?i)([?&](?:token|password|passwd|pass)=)[^&\s"'<>]+`)

func redact(message string) string {
	return querySecretPattern.ReplaceAllString(message, `${1}[REDACTED]`)
}

func New(config Config) (*Client, error) {
	u, err := url.Parse(config.URL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("invalid camera URL %q", config.URL)
	}
	if u.User != nil {
		return nil, fmt.Errorf("camera URL must not contain credentials")
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12, InsecureSkipVerify: config.Insecure} // #nosec G402 -- explicit provider option
	return &Client{
		baseURL: strings.TrimRight(config.URL, "/"), username: config.Username, password: config.Password,
		http: &http.Client{Timeout: 30 * time.Second, Transport: transport},
	}, nil
}

func (c *Client) call(ctx context.Context, apiPath, command string, body any, token string) ([]Envelope, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	u := c.baseURL + apiPath + "?cmd=" + url.QueryEscape(command) + "&token=" + url.QueryEscape(token)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Reolink %s request failed: %s", command, redact(err.Error()))
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Reolink %s returned HTTP %d", command, resp.StatusCode)
	}
	var envelopes []Envelope
	if err := json.Unmarshal(data, &envelopes); err != nil {
		return nil, fmt.Errorf("decode Reolink %s response: %w", command, err)
	}
	if len(envelopes) == 0 {
		return nil, fmt.Errorf("Reolink %s returned an empty response", command)
	}
	return envelopes, nil
}

func (c *Client) login(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.expires.Add(-30*time.Second)) {
		return nil
	}
	body := []any{map[string]any{"cmd": "Login", "action": 0, "param": map[string]any{"User": map[string]string{"userName": c.username, "password": c.password}}}}
	var lastErr error
	for _, path := range []string{"/cgi-bin/api.cgi", "/api.cgi"} {
		responses, err := c.call(ctx, path, "Login", body, "null")
		if err != nil {
			lastErr = err
			continue
		}
		if responses[0].Code != 0 {
			lastErr = apiResponseError("Login", responses[0])
			continue
		}
		var value loginValue
		if err := json.Unmarshal(responses[0].Value, &value); err != nil || value.Token.Name == "" {
			lastErr = fmt.Errorf("Reolink Login omitted token")
			continue
		}
		c.token = value.Token.Name
		lease := time.Duration(value.Token.Lease) * time.Second
		if lease <= 0 {
			lease = time.Hour
		}
		c.expires = time.Now().Add(lease)
		c.apiPath = path
		return nil
	}
	return lastErr
}

func (c *Client) Command(ctx context.Context, command string, action int, param any) (*Envelope, error) {
	if err := c.login(ctx); err != nil {
		return nil, err
	}
	c.mu.Lock()
	token, apiPath := c.token, c.apiPath
	c.mu.Unlock()
	body := []any{map[string]any{"cmd": command, "action": action, "param": param}}
	responses, err := c.call(ctx, apiPath, command, body, token)
	if err != nil {
		return nil, err
	}
	if responses[0].Code != 0 {
		return nil, apiResponseError(command, responses[0])
	}
	return &responses[0], nil
}

func apiResponseError(command string, response Envelope) error {
	if response.Error != nil {
		return fmt.Errorf("Reolink %s failed (code %d, rspCode %d): %s", command, response.Code, response.Error.RspCode, response.Error.Detail)
	}
	return fmt.Errorf("Reolink %s failed with code %d", command, response.Code)
}

type Registry map[string]*Client
