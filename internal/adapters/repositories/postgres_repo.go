package repositories

import (
	"context"
	"database/sql"
	"errors"
	"sync"

	"github.com/esdrassantos06/go-shortener/internal/core/domain"
	"github.com/esdrassantos06/go-shortener/internal/core/ports"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type postgresRepo struct {
	DB                  *sql.DB
	saveStmt            *sql.Stmt
	getByShortIDStmt    *sql.Stmt
	incrementClicksStmt *sql.Stmt
	initOnce            sync.Once
}

func NewPostgresRepo(db *sql.DB) ports.LinkRepository {
	repo := &postgresRepo{DB: db}
	repo.initOnce.Do(repo.initStatements)
	return repo
}

func (r *postgresRepo) initStatements() {
	var err error

	r.saveStmt, err = r.DB.Prepare(`
		INSERT INTO urls (id, "shortId", target_url, "userId", status, "createdAt", clicks)
		VALUES ($1, $2, $3, $4, $5, NOW(), 0)
		RETURNING "createdAt", clicks`)
	if err != nil {
		panic("failed to prepare save statement: " + err.Error())
	}

	r.getByShortIDStmt, err = r.DB.Prepare(`
		SELECT id, "shortId", target_url, status, "createdAt", clicks, "userId" 
		FROM urls 
		WHERE "shortId" = $1 
		LIMIT 1`)
	if err != nil {
		panic("failed to prepare getByShortID statement: " + err.Error())
	}

	r.incrementClicksStmt, err = r.DB.Prepare(`
		UPDATE urls 
		SET clicks = clicks + 1 
		WHERE "shortId" = $1`)
	if err != nil {
		panic("failed to prepare incrementClicks statement: " + err.Error())
	}
}

func (r *postgresRepo) Save(ctx context.Context, link domain.Link) (domain.Link, error) {
	err := r.saveStmt.QueryRowContext(ctx, link.ID, link.ShortID, link.TargetURL, link.UserID, link.Status).Scan(&link.CreatedAt, &link.Clicks)
	return link, err
}

func (r *postgresRepo) GetByShortID(ctx context.Context, shortID string) (domain.Link, error) {
	var link domain.Link

	err := r.getByShortIDStmt.QueryRowContext(ctx, shortID).Scan(&link.ID, &link.ShortID, &link.TargetURL, &link.Status, &link.CreatedAt, &link.Clicks, &link.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Link{}, errors.New("link not found")
		}
		return domain.Link{}, err
	}
	return link, nil
}

func (r *postgresRepo) IncrementClicks(ctx context.Context, shortID string) error {
	_, err := r.incrementClicksStmt.ExecContext(ctx, shortID)
	return err
}
