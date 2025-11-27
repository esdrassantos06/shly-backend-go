package auth

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/esdrassantos06/go-shortener/internal/core/ports"
)

type SessionValidator struct {
	DB           *sql.DB
	Cache        ports.CacheRepository
	validateStmt *sql.Stmt
	initOnce     sync.Once
}

func NewSessionValidator(db *sql.DB, cache ports.CacheRepository) *SessionValidator {
	sv := &SessionValidator{
		DB:    db,
		Cache: cache,
	}
	sv.initOnce.Do(sv.initStatements)
	return sv
}

func (sv *SessionValidator) initStatements() {
	var err error
	sv.validateStmt, err = sv.DB.Prepare(`SELECT "userId" FROM session WHERE token = $1 AND "expiresAt" > CURRENT_TIMESTAMP LIMIT 1`)
	if err != nil {
		panic("failed to prepare validate session statement: " + err.Error())
	}
}

func GetSessionFromCookie(cookieHeader string) (string, error) {
	if cookieHeader == "" {
		return "", errors.New("no cookie header")
	}

	securePrefix := "__Secure-better-auth.session_token="
	if idx := strings.Index(cookieHeader, securePrefix); idx != -1 {
		start := idx + len(securePrefix)
		end := strings.IndexByte(cookieHeader[start:], ';')
		if end == -1 {
			end = len(cookieHeader)
		} else {
			end = start + end
		}
		token := cookieHeader[start:end]
		if decoded, err := url.QueryUnescape(token); err == nil {
			return decoded, nil
		}
		return token, nil
	}

	normalPrefix := "better-auth.session_token="
	if idx := strings.Index(cookieHeader, normalPrefix); idx != -1 {
		start := idx + len(normalPrefix)
		end := strings.IndexByte(cookieHeader[start:], ';')
		if end == -1 {
			end = len(cookieHeader)
		} else {
			end = start + end
		}
		token := cookieHeader[start:end]
		if decoded, err := url.QueryUnescape(token); err == nil {
			return decoded, nil
		}
		return token, nil
	}

	return "", errors.New("session cookie not found")
}

// ValidateSession validates the token by querying Redis first (Better Auth format), then PostgreSQL
// Optimized to reduce Redis round trips by checking the optimized cache key first
func (sv *SessionValidator) ValidateSession(ctx context.Context, sessionToken string) (string, error) {
	if sessionToken == "" {
		return "", errors.New("session token is required")
	}

	dotIdx := strings.IndexByte(sessionToken, '.')
	var sessionID string
	if dotIdx == -1 {
		sessionID = sessionToken
	} else {
		sessionID = sessionToken[:dotIdx]
	}
	cacheKey := "session:" + sessionID

	if cachedUserID, err := sv.Cache.Get(ctx, cacheKey); err == nil && cachedUserID != "" {
		return cachedUserID, nil
	}

	cachedData, err := sv.Cache.Get(ctx, sessionID)
	if err == nil && cachedData != "" {
		var sessionData struct {
			Session struct {
				UserID    string    `json:"userId"`
				ExpiresAt time.Time `json:"expiresAt"`
			} `json:"session"`
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		}

		if json.Unmarshal([]byte(cachedData), &sessionData) == nil {
			if sessionData.Session.ExpiresAt.After(time.Now()) {
				userID := sessionData.Session.UserID
				if userID == "" {
					userID = sessionData.User.ID
				}
				if userID != "" {
					go func() {
						_ = sv.Cache.Set(context.Background(), cacheKey, userID, 300)
					}()
					return userID, nil
				}
			}
		}
	}

	var userID string
	if err := sv.validateStmt.QueryRowContext(ctx, sessionID).Scan(&userID); err == nil {
		go func() {
			_ = sv.Cache.Set(context.Background(), cacheKey, userID, 300)
		}()
		return userID, nil
	}

	return "", errors.New("invalid or expired session")
}
