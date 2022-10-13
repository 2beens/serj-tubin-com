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

var _ Api = (*PsqlApi)(nil)

type PsqlApi struct {
	db *pgxpool.Pool
}

func NewBlogPsqlApi(ctx context.Context, dbHost, dbPort, dbName string) (*PsqlApi, error) {
	connString := fmt.Sprintf("postgres://postgres@%s:%s/%s", dbHost, dbPort, dbName)
	dbPool, err := pgxpool.Connect(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to database: %v\n", err)
	}

	log.Debugf("blog api connected to: %s", connString)

	blogApi := &PsqlApi{
		db: dbPool,
	}

	return blogApi, nil
}

func (api *PsqlApi) CloseDB() {
	if api.db != nil {
		api.db.Close()
	}
}

func (api *PsqlApi) AddBlog(ctx context.Context, blog *Blog) error {
	if blog.Content == "" || blog.Title == "" {
		return errors.New("blog title or content empty")
	}

	rows, err := api.db.Query(
		ctx,
		`INSERT INTO blog (title, created_at, content, claps) VALUES ($1, $2, $3, $4) RETURNING id;`,
		blog.Title, blog.CreatedAt, blog.Content, blog.Claps,
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

func (api *PsqlApi) UpdateBlog(ctx context.Context, blog *Blog) error {
	if blog.Content == "" || blog.Title == "" {
		return ErrBlogTitleOrContentEmpty
	}

	tag, err := api.db.Exec(
		ctx,
		`UPDATE blog SET title = $1, content = $2, claps = $3, WHERE id = $4`,
		blog.Title, blog.Content, blog.Claps, blog.Id,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		log.Tracef("blog %d not updated", blog.Id)
	}

	return nil
}

func (api *PsqlApi) BlogClapped(ctx context.Context, id int) error {
	tag, err := api.db.Exec(ctx, `UPDATE blog SET claps = claps + 1 WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		log.Tracef("blog %d not updated", id)
	}
	return nil
}

func (api *PsqlApi) DeleteBlog(ctx context.Context, id int) (bool, error) {
	tag, err := api.db.Exec(ctx, `DELETE FROM blog WHERE id = $1`, id)
	if err != nil {
		return false, err
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	return true, nil
}

func (api *PsqlApi) All(ctx context.Context) ([]*Blog, error) {
	rows, err := api.db.Query(
		ctx,
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
		var claps int
		if err := rows.Scan(&id, &title, &createdAt, &content, &claps); err != nil {
			return nil, err
		}
		blogs = append(blogs, &Blog{
			Id:        id,
			Title:     title,
			CreatedAt: createdAt,
			Content:   content,
			Claps:     claps,
		})
	}

	return blogs, nil
}

func (api *PsqlApi) BlogsCount(ctx context.Context) (int, error) {
	rows, err := api.db.Query(ctx, `SELECT COUNT(*) FROM blog`)
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

func (api *PsqlApi) GetBlogsPage(ctx context.Context, page, size int) ([]*Blog, error) {
	limit := size
	offset := (page - 1) * size
	blogsCount, err := api.BlogsCount(ctx)
	if err != nil {
		return nil, err
	}

	if blogsCount <= limit {
		return api.All(ctx)
	}

	if blogsCount-offset < limit {
		offset = blogsCount - limit
	}

	log.Tracef("getting blogs, blogs count %d, limit %d, offset %d", blogsCount, limit, offset)

	rows, err := api.db.Query(
		ctx,
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
		var claps int
		if err := rows.Scan(&id, &title, &createdAt, &content, &claps); err != nil {
			return nil, err
		}
		blogs = append(blogs, &Blog{
			Id:        id,
			Title:     title,
			CreatedAt: createdAt,
			Content:   content,
			Claps:     claps,
		})
	}

	return blogs, nil
}
