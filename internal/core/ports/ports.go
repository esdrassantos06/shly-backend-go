package ports

import (
	"context"

	"github.com/esdrassantos06/go-shortener/internal/core/domain"
)

type LinkRepository interface {
	Save(ctx context.Context, link domain.Link) (domain.Link, error)
	GetByShortID(ctx context.Context, shortID string) (domain.Link, error)
	IncrementClicks(ctx context.Context, shortID string) error
}

type CacheRepository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttlSeconds int) error
	IncrementCounter(ctx context.Context, key string) error
}

type LinkService interface {
	ShortenURL(ctx context.Context, targetURL string, customSlug string, userID *string) (domain.Link, error)
	ResolveURL(ctx context.Context, shortID string) (string, error)
}
