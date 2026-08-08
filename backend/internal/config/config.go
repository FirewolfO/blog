package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Address                  string
	DatabaseDSN              string
	AllowedOrigins           []string
	PublicBaseURL            string
	GatewayInnerBaseURL      string
	GatewayAccessKey         string
	GatewaySecretKey         string
	GatewayUpstreamAccessKey string
	GatewayUpstreamSecretKey string
	PermissionAPIBaseURL     string
	PeopleAPIBaseURL         string
	PeopleAuthorizeURL       string
	PeopleClientID           string
	PeopleClientSecret       string
	OAuthRedirectURIs        []string
	StorageEndpoint          string
	StorageAccessKey         string
	StorageSecretKey         string
	StorageBucket            string
	StorageUseSSL            bool
	MaxUploadBytes           int64
}

func Load() Config {
	return Config{
		Address:                  env("BLOG_ADDR", ":8086"),
		DatabaseDSN:              env("BLOG_DB_DSN", "blog.db"),
		AllowedOrigins:           split(env("BLOG_ALLOWED_ORIGINS", "http://localhost:5178,http://127.0.0.1:5178,http://10.251.237.216:5178,http://localhost:5179,http://127.0.0.1:5179,http://10.251.237.216:5179")),
		PublicBaseURL:            strings.TrimRight(env("BLOG_PUBLIC_BASE_URL", ""), "/"),
		GatewayInnerBaseURL:      strings.TrimRight(env("BLOG_GATEWAY_INNER_BASE_URL", "http://127.0.0.1:8082/api/inner/blog"), "/"),
		GatewayAccessKey:         env("BLOG_GATEWAY_ACCESS_KEY", "gwak_blog_console_local"),
		GatewaySecretKey:         env("BLOG_GATEWAY_SECRET_KEY", "local-development-blog-gateway-secret-key"),
		GatewayUpstreamAccessKey: env("BLOG_GATEWAY_UPSTREAM_ACCESS_KEY", "gwak_blog_console_local"),
		GatewayUpstreamSecretKey: env("BLOG_GATEWAY_UPSTREAM_SECRET_KEY", "local-development-blog-gateway-secret-key"),
		PermissionAPIBaseURL:     strings.TrimRight(env("BLOG_PERMISSION_API_BASE_URL", "http://127.0.0.1:8081/api/v1"), "/"),
		PeopleAPIBaseURL:         strings.TrimRight(env("BLOG_PEOPLE_API_BASE_URL", "http://127.0.0.1:8082/api/open/people"), "/"),
		PeopleAuthorizeURL:       env("BLOG_PEOPLE_AUTHORIZE_URL", "http://localhost:5177/oauth/authorize"),
		PeopleClientID:           env("BLOG_PEOPLE_CLIENT_ID", "blog-ui"),
		PeopleClientSecret:       env("BLOG_PEOPLE_CLIENT_SECRET", "blog-local-client-secret-change-me"),
		OAuthRedirectURIs:        split(env("BLOG_OAUTH_REDIRECT_URIS", "http://localhost:5179/oauth/callback,http://127.0.0.1:5179/oauth/callback,http://10.251.237.216:5179/oauth/callback")),
		StorageEndpoint:          env("BLOG_STORAGE_ENDPOINT", "127.0.0.1:3900"),
		StorageAccessKey:         env("BLOG_STORAGE_ACCESS_KEY", "GK0123456789abcdef0123456789abcdef"),
		StorageSecretKey:         env("BLOG_STORAGE_SECRET_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"),
		StorageBucket:            env("BLOG_STORAGE_BUCKET", "blog-media"),
		StorageUseSSL:            strings.EqualFold(env("BLOG_STORAGE_USE_SSL", "false"), "true"),
		MaxUploadBytes:           envInt64("BLOG_MAX_UPLOAD_BYTES", 5*1024*1024),
	}
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt64(key string, fallback int64) int64 {
	value, err := strconv.ParseInt(strings.TrimSpace(os.Getenv(key)), 10, 64)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func split(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if value := strings.TrimSpace(part); value != "" {
			result = append(result, value)
		}
	}
	return result
}
