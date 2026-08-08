package identity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"blog/internal/store"
)

func TestPeopleOAuthAndPermissionFallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/people/oauth/token":
			_ = json.NewEncoder(writer).Encode(map[string]any{"access_token": "people-access"})
		case "/people/oauth/userinfo":
			_ = json.NewEncoder(writer).Encode(map[string]any{"ID": "pep_writer", "Username": "writer", "DisplayName": "作者", "Status": "enabled"})
		case "/permission/auth/me":
			if request.Header.Get("Authorization") != "Bearer permission-token" {
				writer.WriteHeader(http.StatusUnauthorized)
				return
			}
			_ = json.NewEncoder(writer).Encode(map[string]any{"data": map[string]any{"user": map[string]any{"id": "pep_admin", "username": "admin", "displayName": "管理员"}, "permissions": []string{"svc.inner.blog:view"}}})
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()
	database, err := store.Open("file:identity?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	redirectURI := "http://localhost:5179/oauth/callback"
	client := New(database, server.URL+"/permission", "gateway-admin", "permission-secret-at-least-32-bytes", server.URL+"/people", "http://people.local/oauth/authorize", "blog-ui", "secret", []string{redirectURI})
	authorize, err := client.AuthorizationURL(redirectURI)
	if err != nil {
		t.Fatal(err)
	}
	parsed, _ := url.Parse(authorize)
	result, err := client.Exchange(context.Background(), "code", parsed.Query().Get("state"), redirectURI)
	if err != nil || result.User.ID != "pep_writer" {
		t.Fatalf("Exchange() = %#v, %v", result, err)
	}
	local, err := client.Authenticate(context.Background(), result.AccessToken)
	if err != nil || local.Username != "writer" {
		t.Fatalf("Authenticate(local) = %#v, %v", local, err)
	}
	permission, err := client.Authenticate(context.Background(), "permission-token")
	if err != nil || permission.ID != "pep_admin" || !permission.Can("svc.inner.blog:view") || permission.Can("svc.inner.blog:manage") {
		t.Fatalf("Authenticate(permission) = %#v, %v", permission, err)
	}
}
