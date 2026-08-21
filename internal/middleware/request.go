package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const RequestIDKey = "request_id"

type requestIdentity struct {
	ID     string
	Parent string
	Depth  int
}

func normalizeRequestIdentity(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" || len(value) > 128 {
		return newRequestID()
	}
	return value
}

func newIngressIdentity(raw string) requestIdentity {
	return requestIdentity{ID: normalizeRequestIdentity(raw)}
}

func (identity requestIdentity) child(suffix string) requestIdentity {
	suffix = strings.TrimSpace(suffix)
	if suffix == "" {
		suffix = "operation"
	}
	return requestIdentity{
		ID:     identity.ID + "-" + suffix,
		Parent: identity.ID,
		Depth:  identity.Depth + 1,
	}
}

type requestIdentityContextKey struct{}

func bindRequestIdentity(ctx context.Context, identity requestIdentity) context.Context {
	return context.WithValue(ctx, requestIdentityContextKey{}, identity)
}

func IdentityFromContext(ctx context.Context) (string, bool) {
	identity, ok := ctx.Value(requestIdentityContextKey{}).(requestIdentity)
	if !ok || strings.TrimSpace(identity.ID) == "" {
		return "", false
	}
	return identity.ID, true
}

func DeriveRequestIdentity(ctx context.Context) (context.Context, string) {
	parent, ok := ctx.Value(requestIdentityContextKey{}).(requestIdentity)
	if !ok {
		parent = newIngressIdentity("")
	}
	derived := parent.child("timeout")
	return bindRequestIdentity(ctx, derived), derived.ID
}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		identity := newIngressIdentity(c.GetHeader("X-Request-ID"))
		requestID := identity.ID
		c.Set(RequestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		c.Request = c.Request.WithContext(bindRequestIdentity(c.Request.Context(), identity))
		c.Next()
	}
}

func newRequestID() string {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err == nil {
		return "req-" + hex.EncodeToString(buffer)
	}
	return "req-" + time.Now().UTC().Format("20060102T150405.000000000")
}

func CurrentRequestID(c *gin.Context) string {
	if requestID, ok := IdentityFromContext(c.Request.Context()); ok {
		return requestID
	}
	value, exists := c.Get(RequestIDKey)
	if !exists {
		return "unknown"
	}
	requestID, ok := value.(string)
	if !ok || requestID == "" {
		return "unknown"
	}
	return requestID
}
