package api

import (
	"net/http"
	"net/url"
	"testing"

	"blog/internal/blog"
)

func TestRequiredPermission(t *testing.T) {
	tests := []struct{ method, path, want string }{
		{http.MethodGet, "/api/v1/posts", blog.PermissionView},
		{http.MethodPost, "/api/v1/posts/post-1/comments", blog.PermissionView},
		{http.MethodPost, "/api/v1/posts", blog.PermissionView},
		{http.MethodDelete, "/api/v1/comments/comment-1", blog.PermissionView},
		{http.MethodPost, "/api/v1/categories", blog.PermissionManage},
		{http.MethodGet, "/api/v1/reviews/mine", blog.PermissionView},
		{http.MethodGet, "/api/v1/reviews/notifications", blog.PermissionView},
		{http.MethodPost, "/api/v1/reviews/notifications/read", blog.PermissionView},
		{http.MethodPost, "/api/v1/reviews/post-1/approve", blog.PermissionReview},
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
	if !(blog.Identity{Source: "people"}).Can(blog.PermissionView) {
		t.Fatalf("People OAuth identity should be an internal author")
	}
	if (blog.Identity{Source: "people"}).IsAdmin() {
		t.Fatalf("People user must not implicitly become an administrator")
	}
}
