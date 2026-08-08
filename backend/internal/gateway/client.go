package gateway

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	CredentialHeader = "X-Gateway-Credential"
	SignatureHeader  = "X-Gateway-Signature"
	TimestampHeader  = "X-Gateway-Timestamp"
	NonceHeader      = "X-Gateway-Nonce"
	PayloadHeader    = "X-Gateway-Content-SHA256"
)

type Client struct {
	baseURL, accessKey, secretKey string
	httpClient                    *http.Client
}

type Verifier struct {
	accessKey, secretKey string
	skew                 time.Duration
	mu                   sync.Mutex
	nonces               map[string]time.Time
}

func NewVerifier(accessKey, secretKey string, skew time.Duration) *Verifier {
	return &Verifier{accessKey: accessKey, secretKey: secretKey, skew: skew, nonces: map[string]time.Time{}}
}

func (v *Verifier) Verify(request *http.Request, body []byte) error {
	if err := VerifyRequest(request, body, v.accessKey, v.secretKey, v.skew); err != nil {
		return err
	}
	nonce := request.Header.Get(NonceHeader)
	if len(nonce) < 16 || len(nonce) > 128 {
		return fmt.Errorf("invalid Gateway nonce")
	}
	now := time.Now().UTC()
	v.mu.Lock()
	defer v.mu.Unlock()
	for value, expiresAt := range v.nonces {
		if !expiresAt.After(now) {
			delete(v.nonces, value)
		}
	}
	if _, exists := v.nonces[nonce]; exists {
		return fmt.Errorf("Gateway nonce was already used")
	}
	v.nonces[nonce] = now.Add(v.skew)
	return nil
}

func New(baseURL, accessKey, secretKey string) *Client {
	return &Client{baseURL: strings.TrimRight(baseURL, "/"), accessKey: accessKey, secretKey: secretKey, httpClient: &http.Client{Timeout: 20 * time.Second}}
}

func (c *Client) Do(ctx context.Context, method, path, rawQuery, contentType string, body []byte, headers map[string]string) (*http.Response, error) {
	target := c.baseURL + "/" + strings.TrimLeft(path, "/")
	if rawQuery != "" {
		target += "?" + rawQuery
	}
	request, err := http.NewRequestWithContext(ctx, method, target, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	if err := SignRequest(request, body, c.accessKey, c.secretKey); err != nil {
		return nil, err
	}
	return c.httpClient.Do(request)
}

func SignRequest(request *http.Request, body []byte, accessKey, secretKey string) error {
	if accessKey == "" || len(secretKey) < 32 {
		return fmt.Errorf("Gateway credential is invalid")
	}
	timestamp := strconv.FormatInt(time.Now().UTC().Unix(), 10)
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return err
	}
	nonce := hex.EncodeToString(nonceBytes)
	payloadHash := PayloadHash(body)
	canonical, err := CanonicalRequest(request.Method, request.URL.EscapedPath(), request.URL.RawQuery, timestamp, nonce, payloadHash)
	if err != nil {
		return err
	}
	request.Header.Set(CredentialHeader, accessKey)
	request.Header.Set(TimestampHeader, timestamp)
	request.Header.Set(NonceHeader, nonce)
	request.Header.Set(PayloadHeader, payloadHash)
	request.Header.Set(SignatureHeader, Sign(secretKey, canonical))
	return nil
}

func VerifyRequest(request *http.Request, body []byte, accessKey, secretKey string, skew time.Duration) error {
	if request.Header.Get(CredentialHeader) != accessKey || len(secretKey) < 32 {
		return fmt.Errorf("invalid Gateway credential")
	}
	seconds, err := strconv.ParseInt(request.Header.Get(TimestampHeader), 10, 64)
	if err != nil || time.Since(time.Unix(seconds, 0)) > skew || time.Until(time.Unix(seconds, 0)) > skew {
		return fmt.Errorf("expired Gateway signature")
	}
	payloadHash := PayloadHash(body)
	if !hmac.Equal([]byte(strings.ToLower(request.Header.Get(PayloadHeader))), []byte(payloadHash)) {
		return fmt.Errorf("invalid Gateway payload")
	}
	canonical, err := CanonicalRequest(request.Method, request.URL.EscapedPath(), request.URL.RawQuery, request.Header.Get(TimestampHeader), request.Header.Get(NonceHeader), payloadHash)
	if err != nil {
		return err
	}
	expected := Sign(secretKey, canonical)
	if !hmac.Equal([]byte(strings.ToLower(request.Header.Get(SignatureHeader))), []byte(expected)) {
		return fmt.Errorf("invalid Gateway signature")
	}
	return nil
}

func CanonicalRequest(method, path, rawQuery, timestamp, nonce, payloadHash string) (string, error) {
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return "", err
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0)
	for _, key := range keys {
		items := append([]string(nil), values[key]...)
		sort.Strings(items)
		if len(items) == 0 {
			items = []string{""}
		}
		for _, item := range items {
			parts = append(parts, encode(key)+"="+encode(item))
		}
	}
	if path == "" {
		path = "/"
	}
	return strings.Join([]string{strings.ToUpper(method), path, strings.Join(parts, "&"), timestamp, nonce, strings.ToLower(payloadHash)}, "\n"), nil
}

func PayloadHash(body []byte) string { sum := sha256.Sum256(body); return hex.EncodeToString(sum[:]) }
func Sign(secret, value string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = io.WriteString(mac, value)
	return hex.EncodeToString(mac.Sum(nil))
}
func encode(value string) string { return strings.ReplaceAll(url.QueryEscape(value), "+", "%20") }
