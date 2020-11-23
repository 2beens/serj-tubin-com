package internal

import (
	"context"
	"errors"
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

	blogApi := &BlogApi{
		db: dbpool,
	}

	// add test blog
	//blogApi.AddBlog(&Blog{
	//	Title:     "aaa bbb",
	//	CreatedAt: time.Now(),
	//	Content:   "bla bla bla 1243",
	//})

	// get all blogs
	//allBlogs, err := blogApi.All()
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//log.Println("all blogs:")
	//for _, b := range allBlogs {
	//	log.Println(b)
	//}

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
