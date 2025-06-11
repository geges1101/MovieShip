package internal

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
)

type OIDCAuth struct {
	Verifier *oidc.IDTokenVerifier
	Provider *oidc.Provider
}

func NewOIDCMiddleware() (*OIDCAuth, error) {
	issuerURL := os.Getenv("OIDC_ISSUER")
	if issuerURL == "" {
		return nil, fmt.Errorf("OIDC_ISSUER environment variable is not set")
	}

	ctx := context.Background()
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %v", err)
	}

	// Создаем конфигурацию для верификации токенов
	config := &oidc.Config{
		ClientID: os.Getenv("OIDC_CLIENT_ID"),
		// Включаем проверку client_id
		SkipClientIDCheck: false,
	}

	verifier := provider.Verifier(config)
	return &OIDCAuth{
		Verifier: verifier,
		Provider: provider,
	}, nil
}

func (oa *OIDCAuth) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			log.Printf("Missing Authorization header")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing Authorization header"})
			return
		}

		if !strings.HasPrefix(header, "Bearer ") {
			log.Printf("Invalid Authorization header format")
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid Authorization header format"})
			return
		}

		token := strings.TrimPrefix(header, "Bearer ")

		// Верифицируем токен
		idToken, err := oa.Verifier.Verify(c.Request.Context(), token)
		if err != nil {
			log.Printf("Token verification error: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token", "details": err.Error()})
			return
		}

		// Получаем claims
		var claims struct {
			Email             string `json:"email"`
			Sub               string `json:"sub"`
			PreferredUsername string `json:"preferred_username"`
			RealmAccess       struct {
				Roles []string `json:"roles"`
			} `json:"realm_access"`
			Azp string `json:"azp"` // Authorized party
		}

		if err := idToken.Claims(&claims); err != nil {
			log.Printf("Claims parsing error: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
			return
		}

		// Проверяем azp только если он присутствует
		if claims.Azp != "" {
			expectedClientID := os.Getenv("OIDC_CLIENT_ID")
			if claims.Azp != expectedClientID {
				log.Printf("Invalid azp: got %s, want %s", claims.Azp, expectedClientID)
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid client"})
				return
			}
		}

		// Устанавливаем данные пользователя в контекст
		c.Set("user_email", claims.Email)
		c.Set("user_id", claims.Sub)
		c.Set("user_roles", claims.RealmAccess.Roles)

		// Добавляем отладочную информацию в заголовки ответа в режиме разработки
		if gin.Mode() == gin.DebugMode {
			c.Header("X-Debug-Token-Valid", "true")
			c.Header("X-Debug-User-Email", claims.Email)
			c.Header("X-Debug-User-Roles", strings.Join(claims.RealmAccess.Roles, ","))
		}

		c.Next()
	}
}
