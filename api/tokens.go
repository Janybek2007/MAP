package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"
)

const (
	tokenTTL                  = 2 * time.Minute
	tokenPurposeResource      = "resource"
	tokenPurposeTokenEndpoint = "token_endpoint"
	httpMethodPost            = "POST"
)

type tokenClaims struct {
	URL       string `json:"url"`
	Method    string `json:"method"`
	Purpose   string `json:"purpose"`
	ExpiresAt int64  `json:"expires_at"`
	Nonce     string `json:"nonce"`
}

type TokenManager struct {
	apiKey string
	mu     sync.Mutex
	used   map[string]time.Time
}

func NewTokenManager(apiKey string) *TokenManager {
	return &TokenManager{
		apiKey: apiKey,
		used:   make(map[string]time.Time),
	}
}

func (manager *TokenManager) Issue(method string, url string) (tokenResponse, error) {
	return manager.issue(method, url, tokenPurposeResource)
}

func (manager *TokenManager) IssueTokenEndpoint() (tokenResponse, error) {
	return manager.issue(httpMethodPost, "/api/tokens", tokenPurposeTokenEndpoint)
}

func (manager *TokenManager) issue(method string, url string, purpose string) (tokenResponse, error) {
	cleanMethod := strings.ToUpper(strings.TrimSpace(method))
	cleanURL := normalizeTokenURL(url)
	if cleanMethod == "" || cleanURL == "" {
		return tokenResponse{}, errors.New("адрес и метод обязательны")
	}

	nonce, err := randomHex(16)
	if err != nil {
		return tokenResponse{}, err
	}

	claims := tokenClaims{
		URL:       cleanURL,
		Method:    cleanMethod,
		Purpose:   purpose,
		ExpiresAt: time.Now().Add(tokenTTL).Unix(),
		Nonce:     nonce,
	}

	token, err := manager.sign(claims)
	if err != nil {
		return tokenResponse{}, err
	}

	return tokenResponse{
		Token:     token,
		ExpiresAt: claims.ExpiresAt,
	}, nil
}

func (manager *TokenManager) Validate(token string, method string, url string) error {
	return manager.validate(token, method, url, tokenPurposeResource)
}

func (manager *TokenManager) ValidateTokenEndpoint(token string) error {
	return manager.validate(token, httpMethodPost, "/api/tokens", tokenPurposeTokenEndpoint)
}

func (manager *TokenManager) validate(token string, method string, url string, purpose string) error {
	claims, signature, rawPayload, err := manager.decode(token)
	if err != nil {
		return err
	}

	expectedSignature := manager.signature(rawPayload)
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return errors.New("неверная подпись токена")
	}

	if time.Now().Unix() > claims.ExpiresAt {
		return errors.New("срок действия токена истёк")
	}

	if claims.Purpose != purpose {
		return errors.New("токен не подходит для этого действия")
	}

	if claims.Method != strings.ToUpper(strings.TrimSpace(method)) {
		return errors.New("метод токена не совпадает")
	}

	if claims.URL != normalizeTokenURL(url) {
		return errors.New("адрес токена не совпадает")
	}

	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.cleanupLocked()

	if _, exists := manager.used[token]; exists {
		return errors.New("токен уже был использован")
	}

	manager.used[token] = time.Unix(claims.ExpiresAt, 0)
	return nil
}

func (manager *TokenManager) sign(claims tokenClaims) (string, error) {
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	encodedPayload := base64.RawURLEncoding.EncodeToString(payload)
	signature := manager.signature(encodedPayload)
	return encodedPayload + "." + base64.RawURLEncoding.EncodeToString([]byte(signature)), nil
}

func (manager *TokenManager) decode(token string) (tokenClaims, string, string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return tokenClaims{}, "", "", errors.New("неверный формат токена")
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return tokenClaims{}, "", "", errors.New("неверное содержимое токена")
	}

	signatureBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return tokenClaims{}, "", "", errors.New("неверная подпись токена")
	}

	var claims tokenClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return tokenClaims{}, "", "", errors.New("неверные данные токена")
	}

	return claims, string(signatureBytes), parts[0], nil
}

func (manager *TokenManager) signature(payload string) string {
	mac := hmac.New(sha256.New, []byte(manager.apiKey))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

func (manager *TokenManager) cleanupLocked() {
	now := time.Now()
	for token, expiresAt := range manager.used {
		if now.After(expiresAt) {
			delete(manager.used, token)
		}
	}
}

func randomHex(size int) (string, error) {
	payload := make([]byte, size)
	if _, err := rand.Read(payload); err != nil {
		return "", err
	}
	return hex.EncodeToString(payload), nil
}

func normalizeTokenURL(url string) string {
	trimmed := strings.TrimSpace(url)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "http://") || strings.HasPrefix(trimmed, "https://") {
		if index := strings.Index(trimmed, "://"); index >= 0 {
			pathIndex := strings.Index(trimmed[index+3:], "/")
			if pathIndex >= 0 {
				return trimmed[index+3+pathIndex:]
			}
			return "/"
		}
	}
	if !strings.HasPrefix(trimmed, "/") {
		return "/" + trimmed
	}
	return trimmed
}
