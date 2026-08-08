package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"blog/internal/blog"
	"blog/internal/gateway"
	"blog/internal/identity"
	"blog/internal/model"
	"blog/internal/storage"
)

type Server struct {
	address, publicBase string
	allowedOrigins      map[string]bool
	maxUploadBytes      int64
	blog                *blog.Service
	identity            *identity.Client
	gateway             *gateway.Client
	verifier            *gateway.Verifier
	objects             storage.Store
}

type contextKey string

const actorKey contextKey = "blog-actor"

type envelope struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"requestId"`
}

func New(address, publicBase string, allowedOrigins []string, maxUploadBytes int64, service *blog.Service, identities *identity.Client, gatewayClient *gateway.Client, verifier *gateway.Verifier, objects storage.Store) *Server {
	origins := make(map[string]bool, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		origins[origin] = true
	}
	return &Server{address: address, publicBase: strings.TrimRight(publicBase, "/"), allowedOrigins: origins, maxUploadBytes: maxUploadBytes, blog: service, identity: identities, gateway: gatewayClient, verifier: verifier, objects: objects}
}

func (s *Server) ListenAndServe() error {
	server := &http.Server{Addr: s.address, Handler: s.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 90 * time.Second}
	return server.ListenAndServe()
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /media/{id}", s.media)
	mux.HandleFunc("GET /api/v1/auth/oauth/url", s.oauthURL)
	mux.HandleFunc("POST /api/v1/auth/oauth/callback", s.oauthCallback)
	mux.HandleFunc("GET /api/v1/auth/me", s.me)
	mux.HandleFunc("POST /api/v1/auth/logout", s.logout)

	mux.HandleFunc("GET /api/v1/inner/dashboard", s.internal(s.dashboard))
	mux.HandleFunc("GET /api/v1/inner/posts", s.internal(s.listPosts))
	mux.HandleFunc("POST /api/v1/inner/posts", s.internal(s.createPost))
	mux.HandleFunc("GET /api/v1/inner/posts/{id}", s.internal(s.getPost))
	mux.HandleFunc("PUT /api/v1/inner/posts/{id}", s.internal(s.updatePost))
	mux.HandleFunc("DELETE /api/v1/inner/posts/{id}", s.internal(s.deletePost))
	mux.HandleFunc("POST /api/v1/inner/posts/{id}/publish", s.internal(s.publishPost))
	mux.HandleFunc("GET /api/v1/inner/posts/{id}/comments", s.internal(s.listComments))
	mux.HandleFunc("POST /api/v1/inner/posts/{id}/comments", s.internal(s.createComment))
	mux.HandleFunc("DELETE /api/v1/inner/comments/{id}", s.internal(s.deleteComment))
	mux.HandleFunc("GET /api/v1/inner/categories", s.internal(s.listCategories))
	mux.HandleFunc("POST /api/v1/inner/categories", s.internal(s.createCategory))
	mux.HandleFunc("PUT /api/v1/inner/categories/{id}", s.internal(s.updateCategory))
	mux.HandleFunc("DELETE /api/v1/inner/categories/{id}", s.internal(s.deleteCategory))
	mux.HandleFunc("POST /api/v1/inner/media", s.internal(s.uploadMedia))
	mux.HandleFunc("/api/v1/", s.proxy)
	return s.cors(s.requestID(mux))
}

func (s *Server) requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requestID := request.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = blog.NewID("req")
		}
		writer.Header().Set("X-Request-ID", requestID)
		request.Header.Set("X-Request-ID", requestID)
		next.ServeHTTP(writer, request)
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if origin != "" && s.allowedOrigins[origin] {
			writer.Header().Set("Access-Control-Allow-Origin", origin)
			writer.Header().Set("Access-Control-Allow-Credentials", "true")
			writer.Header().Set("Vary", "Origin")
			writer.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		}
		if request.Method == http.MethodOptions {
			writer.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(writer, request)
	})
}

func (s *Server) internal(next http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		body, err := io.ReadAll(http.MaxBytesReader(writer, request.Body, s.maxUploadBytes+1024*1024))
		if err != nil {
			fail(writer, request, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "请求内容超过大小限制")
			return
		}
		request.Body = io.NopCloser(bytes.NewReader(body))
		if err := s.verifier.Verify(request, body); err != nil {
			fail(writer, request, http.StatusUnauthorized, "INVALID_GATEWAY_SIGNATURE", "Gateway 上游签名无效")
			return
		}
		actor, err := decodeActor(request.Header.Get("X-Blog-Identity"))
		if err != nil {
			fail(writer, request, http.StatusUnauthorized, "INVALID_IDENTITY", "调用身份无效")
			return
		}
		next(writer, request.WithContext(context.WithValue(request.Context(), actorKey, actor)))
	}
}

func (s *Server) proxy(writer http.ResponseWriter, request *http.Request) {
	if strings.HasPrefix(request.URL.Path, "/api/v1/inner/") {
		http.NotFound(writer, request)
		return
	}
	actor, token, err := s.authenticate(request)
	if err != nil {
		fail(writer, request, http.StatusUnauthorized, "UNAUTHORIZED", "登录状态无效或已过期")
		return
	}
	if permission := requiredPermission(request); !actor.Can(permission) {
		fail(writer, request, http.StatusForbidden, "FORBIDDEN", "没有 Blog 操作权限")
		return
	}
	body, err := io.ReadAll(http.MaxBytesReader(writer, request.Body, s.maxUploadBytes+1024*1024))
	if err != nil {
		fail(writer, request, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "请求内容超过大小限制")
		return
	}
	encoded, _ := json.Marshal(actor)
	headers := map[string]string{"X-Blog-Identity": base64.RawURLEncoding.EncodeToString(encoded), "X-Request-ID": request.Header.Get("X-Request-ID")}
	response, err := s.gateway.Do(request.Context(), request.Method, strings.TrimPrefix(request.URL.Path, "/api/v1/"), request.URL.RawQuery, request.Header.Get("Content-Type"), body, headers)
	_ = token
	if err != nil {
		log.Printf("Blog Gateway request failed: %v", err)
		fail(writer, request, http.StatusBadGateway, "GATEWAY_UNAVAILABLE", "Gateway 暂时不可用")
		return
	}
	defer response.Body.Close()
	for _, key := range []string{"Content-Type", "Cache-Control", "ETag"} {
		if value := response.Header.Get(key); value != "" {
			writer.Header().Set(key, value)
		}
	}
	writer.WriteHeader(response.StatusCode)
	_, _ = io.Copy(writer, response.Body)
}

func (s *Server) health(writer http.ResponseWriter, request *http.Request) {
	success(writer, request, map[string]string{"status": "ok"})
}

func (s *Server) oauthURL(writer http.ResponseWriter, request *http.Request) {
	result, err := s.identity.AuthorizationURL(request.URL.Query().Get("redirectUri"))
	if err != nil {
		fail(writer, request, http.StatusBadRequest, "INVALID_REDIRECT_URI", "OAuth 回调地址无效")
		return
	}
	success(writer, request, map[string]string{"authorizationUrl": result})
}

func (s *Server) oauthCallback(writer http.ResponseWriter, request *http.Request) {
	var input struct{ Code, State, RedirectURI string }
	if !decodeJSON(writer, request, &input) {
		return
	}
	result, err := s.identity.Exchange(request.Context(), input.Code, input.State, input.RedirectURI)
	if err != nil {
		fail(writer, request, http.StatusUnauthorized, "OAUTH_FAILED", "People OAuth 登录失败")
		return
	}
	success(writer, request, result)
}

func (s *Server) me(writer http.ResponseWriter, request *http.Request) {
	actor, _, err := s.authenticate(request)
	if err != nil {
		fail(writer, request, http.StatusUnauthorized, "UNAUTHORIZED", "登录状态无效或已过期")
		return
	}
	success(writer, request, map[string]any{"user": actor})
}

func (s *Server) logout(writer http.ResponseWriter, request *http.Request) {
	_, token, err := s.authenticate(request)
	if err != nil {
		fail(writer, request, http.StatusUnauthorized, "UNAUTHORIZED", "登录状态无效或已过期")
		return
	}
	_ = s.identity.Logout(token)
	success(writer, request, map[string]bool{"loggedOut": true})
}

func (s *Server) dashboard(writer http.ResponseWriter, request *http.Request) {
	result, err := s.blog.Dashboard()
	respond(writer, request, result, err)
}

func (s *Server) listPosts(writer http.ResponseWriter, request *http.Request) {
	page, _ := strconv.Atoi(request.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(request.URL.Query().Get("pageSize"))
	result, err := s.blog.ListPosts(request.URL.Query().Get("q"), request.URL.Query().Get("status"), request.URL.Query().Get("categoryId"), page, pageSize)
	respond(writer, request, result, err)
}

func (s *Server) createPost(writer http.ResponseWriter, request *http.Request) {
	var input blog.PostInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	result, err := s.blog.CreatePost(actor(request), input)
	if err != nil {
		handleError(writer, request, err)
		return
	}
	write(writer, request, http.StatusCreated, "CREATED", "创建成功", result)
}

func (s *Server) getPost(writer http.ResponseWriter, request *http.Request) {
	result, err := s.blog.GetPost(request.PathValue("id"))
	respond(writer, request, result, err)
}
func (s *Server) updatePost(writer http.ResponseWriter, request *http.Request) {
	var input blog.PostInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	result, err := s.blog.UpdatePost(request.PathValue("id"), actor(request), input)
	respond(writer, request, result, err)
}
func (s *Server) publishPost(writer http.ResponseWriter, request *http.Request) {
	result, err := s.blog.PublishPost(request.PathValue("id"))
	respond(writer, request, result, err)
}
func (s *Server) deletePost(writer http.ResponseWriter, request *http.Request) {
	err := s.blog.DeletePost(request.PathValue("id"))
	if err != nil {
		handleError(writer, request, err)
		return
	}
	success(writer, request, map[string]bool{"deleted": true})
}

func (s *Server) listCategories(writer http.ResponseWriter, request *http.Request) {
	result, err := s.blog.ListCategories()
	respond(writer, request, result, err)
}
func (s *Server) createCategory(writer http.ResponseWriter, request *http.Request) {
	var input blog.CategoryInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	result, err := s.blog.CreateCategory(input)
	if err != nil {
		handleError(writer, request, err)
		return
	}
	write(writer, request, http.StatusCreated, "CREATED", "创建成功", result)
}
func (s *Server) updateCategory(writer http.ResponseWriter, request *http.Request) {
	var input blog.CategoryInput
	if !decodeJSON(writer, request, &input) {
		return
	}
	result, err := s.blog.UpdateCategory(request.PathValue("id"), input)
	respond(writer, request, result, err)
}
func (s *Server) deleteCategory(writer http.ResponseWriter, request *http.Request) {
	err := s.blog.DeleteCategory(request.PathValue("id"))
	if err != nil {
		handleError(writer, request, err)
		return
	}
	success(writer, request, map[string]bool{"deleted": true})
}

func (s *Server) listComments(writer http.ResponseWriter, request *http.Request) {
	result, err := s.blog.ListComments(request.PathValue("id"))
	respond(writer, request, result, err)
}
func (s *Server) createComment(writer http.ResponseWriter, request *http.Request) {
	var input struct {
		Content string `json:"content"`
	}
	if !decodeJSON(writer, request, &input) {
		return
	}
	result, err := s.blog.CreateComment(request.PathValue("id"), actor(request), input.Content)
	if err != nil {
		handleError(writer, request, err)
		return
	}
	write(writer, request, http.StatusCreated, "CREATED", "评论成功", result)
}
func (s *Server) deleteComment(writer http.ResponseWriter, request *http.Request) {
	err := s.blog.DeleteComment(request.PathValue("id"))
	if err != nil {
		handleError(writer, request, err)
		return
	}
	success(writer, request, map[string]bool{"deleted": true})
}

func (s *Server) uploadMedia(writer http.ResponseWriter, request *http.Request) {
	file, header, err := request.FormFile("file")
	if err != nil {
		fail(writer, request, http.StatusBadRequest, "INVALID_FILE", "请选择要上传的图片")
		return
	}
	defer file.Close()
	contentType := header.Header.Get("Content-Type")
	if !allowedImageType(contentType) {
		fail(writer, request, http.StatusBadRequest, "INVALID_FILE_TYPE", "仅支持 JPEG、PNG、GIF 和 WebP 图片")
		return
	}
	data, err := io.ReadAll(io.LimitReader(file, s.maxUploadBytes+1))
	if err != nil || int64(len(data)) > s.maxUploadBytes {
		fail(writer, request, http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "图片超过大小限制")
		return
	}
	id := blog.NewID("media")
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if len(ext) > 8 {
		ext = ""
	}
	key := time.Now().UTC().Format("2006/01/") + id + ext
	if err := s.objects.Put(request.Context(), key, bytes.NewReader(data), int64(len(data)), contentType); err != nil {
		log.Printf("store Blog media: %v", err)
		fail(writer, request, http.StatusServiceUnavailable, "STORAGE_UNAVAILABLE", "文件存储暂时不可用")
		return
	}
	media := &model.Media{ID: id, ObjectKey: key, Filename: filepath.Base(header.Filename), ContentType: contentType, Size: int64(len(data)), URL: s.publicBase + "/media/" + id, UploaderID: actor(request).ID}
	if err := s.blog.SaveMedia(media); err != nil {
		fail(writer, request, http.StatusInternalServerError, "INTERNAL_ERROR", "保存媒体记录失败")
		return
	}
	write(writer, request, http.StatusCreated, "CREATED", "上传成功", media)
}

func (s *Server) media(writer http.ResponseWriter, request *http.Request) {
	media, err := s.blog.GetMedia(request.PathValue("id"))
	if err != nil {
		handleError(writer, request, err)
		return
	}
	object, err := s.objects.Get(request.Context(), media.ObjectKey)
	if err != nil {
		fail(writer, request, http.StatusNotFound, "MEDIA_NOT_FOUND", "图片不存在")
		return
	}
	defer object.Body.Close()
	writer.Header().Set("Content-Type", media.ContentType)
	writer.Header().Set("Content-Length", strconv.FormatInt(object.Size, 10))
	writer.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = io.Copy(writer, object.Body)
}

func (s *Server) authenticate(request *http.Request) (*blog.Identity, string, error) {
	value := strings.TrimSpace(request.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(value), "bearer ") {
		return nil, "", identity.ErrUnauthorized
	}
	token := strings.TrimSpace(value[7:])
	actor, err := s.identity.Authenticate(request.Context(), token)
	return actor, token, err
}

func decodeActor(value string) (blog.Identity, error) {
	raw, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return blog.Identity{}, err
	}
	var actor blog.Identity
	if err := json.Unmarshal(raw, &actor); err != nil || actor.ID == "" {
		return blog.Identity{}, fmt.Errorf("invalid identity")
	}
	return actor, nil
}

func actor(request *http.Request) blog.Identity {
	value, _ := request.Context().Value(actorKey).(blog.Identity)
	return value
}
func allowedImageType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0])) {
	case "image/jpeg", "image/png", "image/gif", "image/webp":
		return true
	}
	return false
}

func requiredPermission(request *http.Request) string {
	if request.Method == http.MethodGet || (request.Method == http.MethodPost && strings.HasSuffix(request.URL.Path, "/comments")) {
		return "svc.inner.blog:view"
	}
	return "svc.inner.blog:manage"
}

func decodeJSON(writer http.ResponseWriter, request *http.Request, target any) bool {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		fail(writer, request, http.StatusBadRequest, "INVALID_REQUEST", "请求参数无效")
		return false
	}
	return true
}

func respond(writer http.ResponseWriter, request *http.Request, result any, err error) {
	if err != nil {
		handleError(writer, request, err)
		return
	}
	success(writer, request, result)
}
func success(writer http.ResponseWriter, request *http.Request, data any) {
	write(writer, request, http.StatusOK, "OK", "success", data)
}
func fail(writer http.ResponseWriter, request *http.Request, status int, code, message string) {
	write(writer, request, status, code, message, nil)
}
func write(writer http.ResponseWriter, request *http.Request, status int, code, message string, data any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(envelope{Code: code, Message: message, Data: data, RequestID: request.Header.Get("X-Request-ID")})
}
func handleError(writer http.ResponseWriter, request *http.Request, err error) {
	switch {
	case errors.Is(err, blog.ErrInvalid):
		fail(writer, request, http.StatusBadRequest, "INVALID_REQUEST", err.Error())
	case errors.Is(err, blog.ErrNotFound):
		fail(writer, request, http.StatusNotFound, "NOT_FOUND", "资源不存在")
	case errors.Is(err, blog.ErrConflict):
		fail(writer, request, http.StatusConflict, "CONFLICT", err.Error())
	default:
		log.Printf("Blog request %s failed: %v", request.Header.Get("X-Request-ID"), err)
		fail(writer, request, http.StatusInternalServerError, "INTERNAL_ERROR", "服务内部错误")
	}
}
