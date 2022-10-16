package blog

import "context"

type Api interface {
	AddBlog(ctx context.Context, blog *Blog) error
	UpdateBlog(ctx context.Context, id int, title, content string) error
	BlogClapped(ctx context.Context, id int) error
	DeleteBlog(ctx context.Context, id int) (bool, error)
	All(ctx context.Context) ([]*Blog, error)
	BlogsCount(ctx context.Context) (int, error)
	GetBlogsPage(ctx context.Context, page, size int) ([]*Blog, error)
}
