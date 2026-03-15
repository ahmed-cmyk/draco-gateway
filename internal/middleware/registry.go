package middleware

import (
	"net/http"
)

type MiddlewareFunc func(http.Handler) http.Handler

var Registry = map[string]MiddlewareFunc{
	"logging": Logging,
}
