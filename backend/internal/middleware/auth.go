package middleware

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type KeycloakCerts struct {
	Keys []struct {
		Kid string `json:"kid"` // Идентификатор ключа (Key ID), уникальный идентификатор для каждого ключа.
		Kty string `json:"kty"` // Тип ключа (Key Type), например, RSA.
		Alg string `json:"alg"` // Алгоритм, используемый для подписи, например, RS256.
		Use string `json:"use"` // Назначение ключа, например, `sig` для подписи.
		N   string `json:"n"`   // Модуль RSA ключа, закодированный в base64.
		E   string `json:"e"`   // Показатель RSA ключа, закодированный в base64.
	} `json:"keys"`
}

// jwkToRsaPublicKey преобразует параметры JWK (JSON Web Key) в объект rsa.PublicKey.
// Она принимает два параметра: `n` и `e`, которые представляют собой модуль и показатель RSA ключа, закодированные в base64.
// Функция декодирует эти значения, преобразует их в большие целые числа и создает объект rsa.PublicKey, который затем возвращает.
func jwkToRsaPublicKey(n, e string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil {
		return nil, fmt.Errorf("failed to decode n: %v", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil {
		return nil, fmt.Errorf("failed to decode e: %v", err)
	}

	nBigInt := new(big.Int).SetBytes(nBytes)
	eBigInt := new(big.Int).SetBytes(eBytes)

	return &rsa.PublicKey{
		N: nBigInt,
		E: int(eBigInt.Int64()),
	}, nil
}

func AuthMiddleware(keycloakURL, realm, requiredRole string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "no authorization header", http.StatusUnauthorized)
				return
			}

			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			// Получаем публичные ключи от Keycloak
			certsURL := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/certs", keycloakURL, realm)
			resp, err := http.Get(certsURL)
			if err != nil {
				log.Printf("Failed to get certs: %v", err)
				http.Error(w, "failed to get public keys", http.StatusInternalServerError)
				return
			}
			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					log.Printf("Failed to close response body: %v", err)
				}
			}(resp.Body)

			var certs KeycloakCerts
			if err := json.NewDecoder(resp.Body).Decode(&certs); err != nil {
				log.Printf("Failed to decode certs: %v", err)
				http.Error(w, "failed to parse public keys", http.StatusInternalServerError)
				return
			}

			// Парсим заголовок токена для получения kid
			parts := strings.Split(tokenString, ".")
			if len(parts) != 3 {
				log.Printf("Invalid token format")
				http.Error(w, "invalid token format", http.StatusUnauthorized)
				return
			}

			headerJSON, err := base64.RawURLEncoding.DecodeString(parts[0])
			if err != nil {
				log.Printf("Failed to decode token header: %v", err)
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			var header struct {
				Kid string `json:"kid"`
				Alg string `json:"alg"`
			}
			if err := json.Unmarshal(headerJSON, &header); err != nil {
				log.Printf("Failed to parse token header: %v", err)
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			// Ищем соответствующий ключ
			var publicKey *rsa.PublicKey
			for _, key := range certs.Keys {
				if key.Kid == header.Kid {
					var err error
					publicKey, err = jwkToRsaPublicKey(key.N, key.E)
					if err != nil {
						log.Printf("Failed to convert JWK to RSA key: %v", err)
						http.Error(w, "invalid token", http.StatusUnauthorized)
						return
					}
					break
				}
			}

			if publicKey == nil {
				log.Printf("No matching key found for kid: %s", header.Kid)
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			// Парсим и проверяем токен
			token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
				// Проверяем, что используется правильный алгоритм
				if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return publicKey, nil
			}, jwt.WithLeeway(5*time.Second)) // Добавляем небольшой запас времени

			if err != nil {
				log.Printf("Token validation failed: %v", err)

				// Проверяем время истечения токена
				if claims, ok := token.Claims.(jwt.MapClaims); ok {
					if exp, ok := claims["exp"].(float64); ok {
						expTime := time.Unix(int64(exp), 0)
						log.Printf("Token expiration time: %v, Current time: %v",
							expTime.UTC(), time.Now().UTC())
					}
				}

				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			if !token.Valid {
				log.Printf("Token is invalid")
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				log.Printf("Failed to get token claims")
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			// Проверяем роли
			if realmAccess, ok := claims["realm_access"].(map[string]interface{}); ok {
				if roles, ok := realmAccess["roles"].([]interface{}); ok {
					hasRole := false
					for _, role := range roles {
						if roleStr, ok := role.(string); ok && roleStr == requiredRole {
							hasRole = true
							break
						}
					}
					if !hasRole {
						log.Printf("User does not have required role: %s", requiredRole)
						http.Error(w, "insufficient permissions", http.StatusForbidden)
						return
					}
				}
			} else {
				log.Printf("No realm_access.roles found in token")
				http.Error(w, "invalid token structure", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
