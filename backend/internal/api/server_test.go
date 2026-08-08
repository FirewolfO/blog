package api

import (
	"net/http"
	"net/url"
	"testing"

	"blog/internal/blog"
)

func TestRequiredPermission(t *testing.T) {
	tests := []struct{ method, path, want string }{
		{http.MethodGet, "/api/v1/posts", "svc.inner.blog:view"},
		{http.MethodPost, "/api/v1/posts/post-1/comments", "svc.inner.blog:view"},
		{http.MethodPost, "/api/v1/posts", "svc.inner.blog:manage"},
		{http.MethodDelete, "/api/v1/comments/comment-1", "svc.inner.blog:manage"},
	}
	for _, test := range tests {
		request := &http.Request{Method: test.method, URL: &url.URL{Path: test.path}}
		if got := requiredPermission(request); got != test.want {
			t.Errorf("requiredPermission(%s %s) = %s, want %s", test.method, test.path, got, test.want)
		}
	}
}

func TestPermissionIdentityAuthorization(t *testing.T) {
	viewer := blog.Identity{Source: "permission", Permissions: []string{"svc.inner.blog:view"}}
	if !viewer.Can("svc.inner.blog:view") || viewer.Can("svc.inner.blog:manage") {
		t.Fatalf("viewer permissions are incorrect")
	}
	if !(blog.Identity{Source: "people"}).Can("svc.inner.blog:manage") {
		t.Fatalf("People OAuth identity should be an internal author")
	}
}
