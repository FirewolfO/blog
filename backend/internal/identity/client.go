package identity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"blog/internal/blog"
	"blog/internal/model"
	"blog/internal/store"
	"gorm.io/gorm"
)

var ErrUnauthorized = errors.New("unauthorized")

type Client struct {
	database                   *store.Store
	permissionBase, peopleBase string
	peopleAuthorize, clientID  string
	clientSecret               string
	redirectURIs               map[string]bool
	httpClient                 *http.Client
}

type SessionResult struct {
	AccessToken string        `json:"accessToken"`
	ExpiresAt   time.Time     `json:"expiresAt"`
	User        blog.Identity `json:"user"`
}

func New(database *store.Store, permissionBase, peopleBase, peopleAuthorize, clientID, clientSecret string, redirectURIs []string) *Client {
	allowed := make(map[string]bool, len(redirectURIs))
	for _, item := range redirectURIs {
		allowed[item] = true
	}
	return &Client{database: database, permissionBase: strings.TrimRight(permissionBase, "/"), peopleBase: strings.TrimRight(peopleBase, "/"), peopleAuthorize: peopleAuthorize, clientID: clientID, clientSecret: clientSecret, redirectURIs: allowed, httpClient: &http.Client{Timeout: 10 * time.Second}}
}

func (c *Client) AuthorizationURL(redirectURI string) (string, error) {
	if !c.redirectURIs[redirectURI] {
		return "", fmt.Errorf("redirect URI is not allowed")
	}
	state, err := randomToken(24)
	if err != nil {
		return "", err
	}
	if err := c.database.DB.Create(&model.OAuthState{StateHash: hash(state), RedirectURI: redirectURI, ExpiresAt: time.Now().UTC().Add(10 * time.Minute)}).Error; err != nil {
		return "", err
	}
	target, err := url.Parse(c.peopleAuthorize)
	if err != nil || !target.IsAbs() {
		return "", fmt.Errorf("invalid People authorize URL")
	}
	target.RawQuery = url.Values{"client_id": {c.clientID}, "redirect_uri": {redirectURI}, "response_type": {"code"}, "scope": {"openid profile"}, "state": {state}}.Encode()
	return target.String(), nil
}

func (c *Client) Exchange(ctx context.Context, code, state, redirectURI string) (*SessionResult, error) {
	var saved model.OAuthState
	err := c.database.DB.Where("state_hash = ? AND redirect_uri = ? AND expires_at > ?", hash(state), redirectURI, time.Now().UTC()).First(&saved).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: OAuth state is invalid or expired", ErrUnauthorized)
	}
	if err != nil {
		return nil, err
	}
	if err := c.database.DB.Delete(&saved).Error; err != nil {
		return nil, err
	}
	employee, err := c.exchangePeople(ctx, code, redirectURI)
	if err != nil {
		return nil, err
	}
	token, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	expiresAt := time.Now().UTC().Add(12 * time.Hour)
	session := &model.Session{TokenHash: hash(token), UserID: employee.ID, Username: employee.Username, DisplayName: employee.DisplayName, ExpiresAt: expiresAt}
	if err := c.database.DB.Create(session).Error; err != nil {
		return nil, err
	}
	return &SessionResult{AccessToken: token, ExpiresAt: expiresAt, User: *employee}, nil
}

func (c *Client) Authenticate(ctx context.Context, token string) (*blog.Identity, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, ErrUnauthorized
	}
	var session model.Session
	if err := c.database.DB.Where("token_hash = ? AND expires_at > ?", hash(token), time.Now().UTC()).First(&session).Error; err == nil {
		return &blog.Identity{ID: session.UserID, Username: session.Username, DisplayName: session.DisplayName, Source: "people"}, nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	return c.permissionIdentity(ctx, token)
}

func (c *Client) Logout(token string) error {
	return c.database.DB.Where("token_hash = ?", hash(strings.TrimSpace(token))).Delete(&model.Session{}).Error
}

func (c *Client) exchangePeople(ctx context.Context, code, redirectURI string) (*blog.Identity, error) {
	values := url.Values{"grant_type": {"authorization_code"}, "code": {strings.TrimSpace(code)}, "redirect_uri": {redirectURI}}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.peopleBase+"/oauth/token", strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.SetBasicAuth(c.clientID, c.clientSecret)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("People token request: %w", err)
	}
	defer response.Body.Close()
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(response.Body).Decode(&token); err != nil || response.StatusCode != http.StatusOK || token.AccessToken == "" {
		return nil, fmt.Errorf("%w: People token exchange failed", ErrUnauthorized)
	}
	request, err = http.NewRequestWithContext(ctx, http.MethodGet, c.peopleBase+"/oauth/userinfo", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token.AccessToken)
	response, err = c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("People userinfo request: %w", err)
	}
	defer response.Body.Close()
	var employee struct{ ID, Username, DisplayName, Status string }
	if err := json.NewDecoder(response.Body).Decode(&employee); err != nil || response.StatusCode != http.StatusOK || employee.ID == "" || employee.Status != "enabled" {
		return nil, fmt.Errorf("%w: People identity is invalid", ErrUnauthorized)
	}
	return &blog.Identity{ID: employee.ID, Username: employee.Username, DisplayName: employee.DisplayName, Source: "people"}, nil
}

func (c *Client) permissionIdentity(ctx context.Context, token string) (*blog.Identity, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.permissionBase+"/auth/me", nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("Permission identity request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, ErrUnauthorized
	}
	var payload struct {
		Data struct {
			User        struct{ ID, Username, DisplayName string } `json:"user"`
			Permissions []string                                   `json:"permissions"`
		} `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil || payload.Data.User.ID == "" {
		return nil, ErrUnauthorized
	}
	return &blog.Identity{ID: payload.Data.User.ID, Username: payload.Data.User.Username, DisplayName: payload.Data.User.DisplayName, Source: "permission", Permissions: payload.Data.Permissions}, nil
}

func randomToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return hex.EncodeToString(value), nil
}
func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
