package services

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/esdrassantos06/go-shortener/internal/core/domain"
	"github.com/esdrassantos06/go-shortener/internal/core/ports"
	"github.com/google/uuid"
)

type DefaultLinkService struct {
	Repo  ports.LinkRepository
	Cache ports.CacheRepository
}

func NewLinkService(repo ports.LinkRepository, cache ports.CacheRepository) ports.LinkService {
	return &DefaultLinkService{Repo: repo, Cache: cache}
}

func (s *DefaultLinkService) ShortenURL(ctx context.Context, targetURL string, customSlug string, userID *string) (domain.Link, error) {
	if userID == nil || *userID == "" {
		return domain.Link{}, errors.New("userID is required")
	}

	var linkID, shortID string
	if customSlug == "" {
		linkUUID := uuid.New().String()
		linkID = linkUUID
		shortID = linkUUID[:6]
	} else {
		shortID = customSlug
		linkID = uuid.New().String()
	}

	link := domain.Link{
		ID:        linkID,
		ShortID:   shortID,
		TargetURL: targetURL,
		UserID:    userID,
		Status:    domain.StatusActive,
	}

	link, err := s.Repo.Save(ctx, link)
	if err != nil {
		return domain.Link{}, err
	}

	go s.cacheLink(link)

	return link, nil
}

type cachedLink struct {
	TargetURL string            `json:"target_url"`
	Status    domain.LinkStatus `json:"status"`
}

func (s *DefaultLinkService) ResolveURL(ctx context.Context, shortID string) (domain.Link, error) {
	if shortID == "" {
		return domain.Link{}, errors.New("shortID is required")
	}

	cacheKey := "url" + shortID
	if val, err := s.Cache.Get(ctx, cacheKey); err == nil && val != "" {
		var cached cachedLink
		if json.Unmarshal([]byte(val), &cached) == nil {
			if cached.Status == domain.StatusPaused {
				return domain.Link{}, errors.New("link is paused")
			}
			go s.trackClick(shortID)
			return domain.Link{
				ShortID:   shortID,
				TargetURL: cached.TargetURL,
				Status:    cached.Status,
			}, nil
		}
	}

	link, err := s.Repo.GetByShortID(ctx, shortID)
	if err != nil {
		return domain.Link{}, err
	}

	if link.Status == domain.StatusPaused {
		return domain.Link{}, errors.New("link is paused")
	}

	go func(l domain.Link) {
		s.cacheLink(l)
		s.trackClick(shortID)
	}(link)

	return link, nil
}

func (s *DefaultLinkService) trackClick(shortID string) {
	ctx := context.Background()
	s.Cache.IncrementCounter(ctx, "stats:"+shortID)

	if err := s.Repo.IncrementClicks(ctx, shortID); err != nil {
		log.Printf("failed to increment clicks for shortID %s: %v", shortID, err)
	}
}

func (s *DefaultLinkService) cacheLink(link domain.Link) {
	cacheKey := "url" + link.ShortID
	payload, err := json.Marshal(cachedLink{
		TargetURL: link.TargetURL,
		Status:    link.Status,
	})
	if err != nil {
		return
	}
	_ = s.Cache.Set(context.Background(), cacheKey, string(payload), 86400)
}
