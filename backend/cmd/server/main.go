package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"blog/internal/api"
	"blog/internal/blog"
	"blog/internal/config"
	"blog/internal/gateway"
	"blog/internal/identity"
	"blog/internal/storage"
	"blog/internal/store"
)

func main() {
	cfg := config.Load()
	if len(cfg.GatewaySecretKey) < 32 || len(cfg.GatewayUpstreamSecretKey) < 32 || len(cfg.PeopleClientSecret) < 32 {
		log.Fatal("Blog Gateway 与 OAuth 密钥至少需要 32 个字符")
	}
	database, err := store.Open(cfg.DatabaseDSN)
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	objects, err := storage.NewS3(cfg.StorageEndpoint, cfg.StorageAccessKey, cfg.StorageSecretKey, cfg.StorageBucket, cfg.StorageUseSSL)
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := objects.EnsureBucket(ctx); err != nil {
		log.Fatalf("初始化 Blog 文件存储失败: %v", err)
	}
	service := blog.New(database)
	identities := identity.New(database, cfg.PermissionAPIBaseURL, cfg.PeopleAPIBaseURL, cfg.PeopleAuthorizeURL, cfg.PeopleClientID, cfg.PeopleClientSecret, cfg.OAuthRedirectURIs)
	gatewayClient := gateway.New(cfg.GatewayInnerBaseURL, cfg.GatewayAccessKey, cfg.GatewaySecretKey)
	verifier := gateway.NewVerifier(cfg.GatewayUpstreamAccessKey, cfg.GatewayUpstreamSecretKey, 5*time.Minute)
	server := api.New(cfg.Address, cfg.PublicBaseURL, cfg.AllowedOrigins, cfg.MaxUploadBytes, service, identities, gatewayClient, verifier, objects)
	log.Printf("Blog 服务监听于 %s", cfg.Address)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
