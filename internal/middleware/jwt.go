package middleware

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/golang-jwt/jwt/v5"
)

// You should store your decoded key in a package-level variable or pass it to the middleware
var jwtSecret []byte

func InitKey(privateString string) {
	decoded, err := base64.StdEncoding.DecodeString(privateString)
	if err != nil {
		log.Warn("Key is not base64, using raw bytes")
		jwtSecret = []byte(privateString)
		return
	}
	jwtSecret = decoded
}

func JWTValidation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header is missing", http.StatusUnauthorized)
			return
		}

		tokenString, found := strings.CutPrefix(authHeader, "Bearer ")
		if !found {
			http.Error(w, "Invalid format (use 'Bearer <token>')", http.StatusUnauthorized)
			return
		}

		// 1. Parse and Validate the Token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// 2. Validate the algorithm (Don't skip this! Important for security)
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})

		// 3. Check if the token is valid
		if err != nil || !token.Valid {
			log.Errorf("Token validation failed: %v", err)
			http.Error(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// (Optional) 4. Extract claims and inject into context
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			log.Infof("Authenticated user: %v", claims["sub"])
		}

		next.ServeHTTP(w, r)
	})
}
