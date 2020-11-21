package internal

import (
	"context"
	"fmt"

	"time"

	"github.com/jackc/pgx/v4/pgxpool"
)

type BlogApi struct {
	db *pgxpool.Pool
}

func NewBlogApi() (*BlogApi, error) {
	ctx := context.Background()

	// TODO: place in env variable
	const connString = "postgres://postgres@localhost:5432/serj_blogs"
	dbpool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v\n", err)
	}

	dbpool.Exec(ctx,
		`INSERT INTO blog(title, created_at, content) VALUES ($1, $2, $3)`,
		"test-title", time.Now(), "test content",
	)

	return &BlogApi{
		db: dbpool,
	}, nil
}

func (b *BlogApi) CloseDB() {
	if b.db != nil {
		b.db.Close()
	}
}

func (b *BlogApi) AddBlog(blog *Blog) error {
	return nil
}

func (b *BlogApi) All() ([]Blog, error) {
	return nil, nil
}
