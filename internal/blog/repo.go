package blog

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	log "github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"

	"github.com/2beens/serjtubincom/internal/telemetry/tracing"
)

// manual caching of blog posts not needed (at least for this use case):
// https://github.com/jackc/pgx/wiki/Automatic-Prepared-Statement-Caching

var (
	ErrBlogNotFound            = errors.New("blog not found")
	ErrBlogTitleOrContentEmpty = errors.New("blog title or content empty")
)

type Blog struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	Content   string    `json:"content"`
	Claps     int       `json:"claps"` // basically blog likes
}

var _ blogRepo = (*Repo)(nil)

type Repo struct {
	db *pgxpool.Pool
}

func NewRepo(db *pgxpool.Pool) *Repo {
	return &Repo{
		db: db,
	}
}

func (r *Repo) AddBlog(ctx context.Context, blog *Blog) error {
	if blog.Content == "" || blog.Title == "" {
		return errors.New("blog title or content empty")
	}

	if blog.CreatedAt.IsZero() {
		blog.CreatedAt = time.Now()
	}

	rows, err := r.db.Query(
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
			blog.ID = id
			return nil
		}
	}

	return errors.New("unexpected error, failed to insert blog")
}

// UpdateBlog will update the content and title of the blog
// createdAt and claps are not updated
func (r *Repo) UpdateBlog(ctx context.Context, id int, title, content string) error {
	if content == "" || title == "" {
		return ErrBlogTitleOrContentEmpty
	}

	tag, err := r.db.Exec(
		ctx,
		`UPDATE blog SET title = $1, content = $2 WHERE id = $3`,
		title, content, id,
	)
	if err != nil {
		return err
	}

	if tag.RowsAffected() == 0 {
		log.Tracef("blog %d not updated", id)
	}

	return nil
}

func (r *Repo) BlogClapped(ctx context.Context, id int) error {
	tag, err := r.db.Exec(ctx, `UPDATE blog SET claps = claps + 1 WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrBlogNotFound
	}
	return nil
}

func (r *Repo) DeleteBlog(ctx context.Context, id int) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM blog WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrBlogNotFound
	}
	return nil
}

func (r *Repo) All(ctx context.Context) ([]*Blog, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "blogApi.All")
	defer span.End()

	rows, err := r.db.Query(
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

	return r.rows2blogs(rows)
}

func (r *Repo) BlogsCount(ctx context.Context) (int, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "blogApi.BlogsCount")
	defer span.End()

	rows, err := r.db.Query(ctx, `SELECT COUNT(*) FROM blog`)
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

func (r *Repo) GetBlogsPage(ctx context.Context, page, size int) ([]*Blog, error) {
	ctx, span := tracing.GlobalTracer.Start(ctx, "blogApi.GetBlogsPage")
	span.SetAttributes(attribute.Int("page", page))
	span.SetAttributes(attribute.Int("size", size))
	defer span.End()

	limit := size
	offset := (page - 1) * size
	blogsCount, err := r.BlogsCount(ctx)
	if err != nil {
		return nil, err
	}

	if blogsCount <= limit {
		return r.All(ctx)
	}

	if blogsCount-offset < limit {
		offset = blogsCount - limit
	}

	log.Tracef("getting blogs, blogs count %d, limit %d, offset %d", blogsCount, limit, offset)

	rows, err := r.db.Query(
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

	return r.rows2blogs(rows)
}

func (r *Repo) GetBlog(ctx context.Context, id int) (*Blog, error) {
	log.Tracef("getting blog %d", id)

	ctx, span := tracing.GlobalTracer.Start(ctx, "blogApi.GetBlog")
	span.SetAttributes(attribute.Int("id", id))
	defer span.End()

	rows, err := r.db.Query(
		ctx,
		`
			SELECT * FROM blog
			WHERE id = $1;
		`,
		id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if !rows.Next() {
		return nil, ErrBlogNotFound
	}

	var blogId int
	var title string
	var createdAt time.Time
	var content string
	var claps int
	if err := rows.Scan(&blogId, &title, &createdAt, &content, &claps); err != nil {
		return nil, err
	}
	return &Blog{
		ID:        blogId,
		Title:     title,
		CreatedAt: createdAt,
		Content:   content,
		Claps:     claps,
	}, nil
}

func (r *Repo) rows2blogs(rows pgx.Rows) ([]*Blog, error) {
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
			ID:        id,
			Title:     title,
			CreatedAt: createdAt,
			Content:   content,
			Claps:     claps,
		})
	}
	return blogs, nil
}
