package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/esdrassantos06/go-shortener/internal/core/ports"
)

type SessionValidator struct {
	DB    *sql.DB
	Cache ports.CacheRepository
}

func NewSessionValidator(db *sql.DB, cache ports.CacheRepository) *SessionValidator {
	return &SessionValidator{
		DB:    db,
		Cache: cache,
	}
}

func GetSessionFromCookie(cookieHeader string) (string, error) {
	if cookieHeader == "" {
		return "", errors.New("no cookie header")
	}

	cookies := strings.Split(cookieHeader, ";")
	for _, cookie := range cookies {
		cookie = strings.TrimSpace(cookie)

		if strings.HasPrefix(cookie, "__Secure-better-auth.session_token=") {
			token := strings.TrimPrefix(cookie, "__Secure-better-auth.session_token=")
			decoded, err := url.QueryUnescape(token)
			if err != nil {
				return token, nil
			}
			return decoded, nil
		}
	}

	return "", errors.New("session cookie not found")
}

// ValidateSession validates the token by querying the Better Auth database
// Uses Redis cache to avoid repeated database queries for the same session
// The session table has: id, token, userId, expiresAt, etc.
// The cookie comes in the format: sessionId.encryptedData
// The 'token' field in the table stores only the sessionId (first part before the dot)
func (sv *SessionValidator) ValidateSession(ctx context.Context, sessionToken string) (string, error) {
	if sessionToken == "" {
		return "", errors.New("session token is required")
	}

	parts := strings.Split(sessionToken, ".")
	sessionID := parts[0]

	cacheKey := "session:" + sessionID
	cachedUserID, err := sv.Cache.Get(ctx, cacheKey)
	if err == nil && cachedUserID != "" {
		return cachedUserID, nil
	}

	var userID string
	var expiresAt time.Time

	query := `
		SELECT "userId", "expiresAt" 
		FROM session 
		WHERE token = $1 
		AND "expiresAt" > NOW()
		LIMIT 1
	`

	err = sv.DB.QueryRowContext(ctx, query, sessionID).Scan(&userID, &expiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errors.New("invalid or expired session")
		}
		return "", err
	}

	go func() {
		_ = sv.Cache.Set(context.Background(), cacheKey, userID, 60)
	}()

	return userID, nil
}
