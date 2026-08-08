package gateway

import (
	"bytes"
	"net/http"
	"testing"
	"time"
)

func TestSignAndVerifyRequest(t *testing.T) {
	body := []byte(`{"title":"Inner Blog"}`)
	request, err := http.NewRequest(http.MethodPost, "http://gateway.local/api/inner/blog/posts?tag=go+lang&tag=api", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	secret := "0123456789abcdef0123456789abcdef"
	if err := SignRequest(request, body, "gwak_blog_test", secret); err != nil {
		t.Fatal(err)
	}
	if err := VerifyRequest(request, body, "gwak_blog_test", secret, 5*time.Minute); err != nil {
		t.Fatalf("VerifyRequest() error = %v", err)
	}
	verifier := NewVerifier("gwak_blog_test", secret, 5*time.Minute)
	if err := verifier.Verify(request, body); err != nil {
		t.Fatalf("Verifier.Verify() error = %v", err)
	}
	if err := verifier.Verify(request, body); err == nil {
		t.Fatal("Verifier.Verify() accepted replay")
	}
	request.Header.Set(PayloadHeader, PayloadHash([]byte("changed")))
	if err := VerifyRequest(request, body, "gwak_blog_test", secret, 5*time.Minute); err == nil {
		t.Fatal("VerifyRequest() accepted changed payload")
	}
}
