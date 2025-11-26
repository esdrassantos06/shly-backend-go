package repositories

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/esdrassantos06/go-shortener/internal/core/domain"
	"github.com/esdrassantos06/go-shortener/internal/core/ports"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type postgresRepo struct {
	DB *sql.DB
}

func NewPostgresRepo(db *sql.DB) ports.LinkRepository {
	return &postgresRepo{DB: db}
}

func (r *postgresRepo) Save(ctx context.Context, link domain.Link) error {
	query := `
	INSERT INTO urls (id, "shortId", target_url, "userId", status, "createdAt", clicks)
	VALUES ($1, $2, $3, $4, $5, $6, $7)`

	_, err := r.DB.ExecContext(ctx, query, link.ID, link.ShortID, link.TargetURL, link.UserID, link.Status, time.Now(), 0)
	return err
}

func (r *postgresRepo) GetByShortID(ctx context.Context, shortID string) (domain.Link, error) {
	query := `SELECT id, "shortId", target_url, status FROM urls WHERE "shortId" = $1 LIMIT 1`
	var link domain.Link

	err := r.DB.QueryRowContext(ctx, query, shortID).Scan(&link.ID, &link.ShortID, &link.TargetURL, &link.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Link{}, errors.New("link not found")
		}
		return domain.Link{}, err
	}
	return link, nil
}

func (r *postgresRepo) IncrementClicks(ctx context.Context, shortID string) error {
	_, err := r.DB.ExecContext(ctx, `UPDATE urls SET clicks = clicks + 1 WHERE "shortId" = $1`, shortID)
	return err
}
