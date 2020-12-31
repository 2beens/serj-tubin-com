package blog

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	log "github.com/sirupsen/logrus"
)

// manual caching of blog posts not needed (at least for this use case):
// https://github.com/jackc/pgx/wiki/Automatic-Prepared-Statement-Caching

var _ BlogApi = (*PsqlApi)(nil)

type PsqlApi struct {
	db *pgxpool.Pool
}

func NewBlogPsqlApi() (*PsqlApi, error) {
	ctx := context.Background()

	// TODO: place in env variable
	// or even better - introduce yml config files, and store there
	const connString = "postgres://postgres@localhost:5432/serj_blogs"
	dbpool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v\n", err)
	}

	blogApi := &PsqlApi{
		db: dbpool,
	}

	return blogApi, nil
}

func (b *PsqlApi) CloseDB() {
	if b.db != nil {
		b.db.Close()
	}
}

func (b *PsqlApi) AddBlog(blog *Blog) error {
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

func (b *PsqlApi) UpdateBlog(blog *Blog) error {
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

func (b *PsqlApi) DeleteBlog(id int) (bool, error) {
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

func (b *PsqlApi) All() ([]*Blog, error) {
	rows, err := b.db.Query(
		context.Background(),
		`SELECT * FROM blog ORDER BY id DESC;`,
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

func (b *PsqlApi) BlogsCount() (int, error) {
	rows, err := b.db.Query(
		context.Background(),
		`SELECT COUNT(*) FROM blog;`,
	)
	if err != nil {
		return -1, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return -1, err
	}

	if rows.Next() {
		var count int
		if err := rows.Scan(&count); err == nil {
			return count, nil
		}
	}

	return -1, errors.New("unexpected error, failed to get blogs count")
}

func (b *PsqlApi) GetBlogsPage(page, size int) ([]*Blog, error) {
	limit := size
	offset := (page - 1) * size
	blogsCount, err := b.BlogsCount()
	if err != nil {
		return nil, err
	}

	if blogsCount <= limit {
		return b.All()
	}

	if blogsCount-offset < limit {
		offset = blogsCount - limit
	}

	log.Tracef("getting blogs, blogs count %d, limit %d, offset %d", blogsCount, limit, offset)

	rows, err := b.db.Query(
		context.Background(),
		`
			SELECT * FROM blog
			ORDER BY id DESC
			LIMIT $1
			OFFSET $2;
		`,
		limit,
		offset,
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
