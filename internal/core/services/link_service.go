package services

import (
	"context"
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

	go func() {
		_ = s.Cache.Set(context.Background(), "url"+shortID, targetURL, 86400)
	}()

	return link, nil
}

func (s *DefaultLinkService) ResolveURL(ctx context.Context, shortID string) (string, error) {
	val, err := s.Cache.Get(ctx, "url"+shortID)
	if err == nil && val != "" {
		go s.trackClick(shortID)
		return val, nil
	}

	link, err := s.Repo.GetByShortID(ctx, shortID)
	if err != nil {
		return "", err
	}

	if link.Status == domain.StatusPaused {
		return "", errors.New("link is paused")
	}

	go func() {
		_ = s.Cache.Set(context.Background(), "url"+shortID, link.TargetURL, 86400)
		s.trackClick(shortID)
	}()

	return link.TargetURL, nil
}

func (s *DefaultLinkService) trackClick(shortID string) {
	ctx := context.Background()
	s.Cache.IncrementCounter(ctx, "stats:"+shortID)

	if err := s.Repo.IncrementClicks(ctx, shortID); err != nil {
		log.Printf("failed to increment clicks for shortID %s: %v", shortID, err)
	}
}
