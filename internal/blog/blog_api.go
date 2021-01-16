package blog

type BlogApi interface {
	CloseDB()
	AddBlog(blog *Blog) error
	UpdateBlog(blog *Blog) error
	DeleteBlog(id int) (bool, error)
	All() ([]*Blog, error)
	BlogsCount() (int, error)
	GetBlogsPage(page, size int) ([]*Blog, error)
}
