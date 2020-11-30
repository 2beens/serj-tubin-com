package internal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/sirupsen/logrus"
)

// TODO: add caching

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

	blogApi := &BlogApi{
		db: dbpool,
	}

	return blogApi, nil
}

func (b *BlogApi) CloseDB() {
	if b.db != nil {
		b.db.Close()
	}
}

func (b *BlogApi) AddBlog(blog *Blog) error {
	if blog.Content == "" || blog.Title == "" {
		return errors.New("blog title or content empty")
	}

	rows, err := b.db.Query(
		context.Background(),
		`INSERT INTO blog (title, created_at, content) VALUES ($1, $2, $3) RETURNING id;`,
		blog.Title, blog.CreatedAt, blog.Content,
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return err
	}

	if rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			blog.Id = id
			return nil
		}
	}

	return errors.New("unexpected error, failed to insert blog")
}

func (b *BlogApi) UpdateBlog(blog *Blog) error {
	if blog.Content == "" || blog.Title == "" {
		return errors.New("blog title or content empty")
	}

	tag, err := b.db.Exec(
		context.Background(),
		`UPDATE blog SET title = $1, content = $2 WHERE id = $3;`,
		blog.Title, blog.Content, blog.Id,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		log.Tracef("blog %d not updated", blog.Id)
	}

	return nil
}

func (b *BlogApi) DeleteBlog(id int) (bool, error) {
	tag, err := b.db.Exec(
		context.Background(),
		`DELETE FROM blog WHERE id = $1`,
		id,
	)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	return true, nil
}

func (b *BlogApi) All() ([]*Blog, error) {
	rows, err := b.db.Query(
		context.Background(),
		`SELECT * FROM blog;`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var blogs []*Blog
	for rows.Next() {
		var id int
		var title string
		var createdAt time.Time
		var content string
		if err := rows.Scan(&id, &title, &createdAt, &content); err != nil {
			return nil, err
		}
		blogs = append(blogs, &Blog{
			Id:        id,
			Title:     title,
			CreatedAt: createdAt,
			Content:   content,
		})
	}

	return blogs, nil
}

func (b *BlogApi) GetBlogsPage(page, size int) ([]*Blog, error) {
	log.Tracef("getting blogs page %d, size %d", page, size)

	// https://www.postgresql.org/docs/8.3/queries-limit.html

	return nil, nil
}
