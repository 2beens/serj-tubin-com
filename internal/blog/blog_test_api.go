package blog

import (
	"errors"
	"sync"
)

type TestApi struct {
	Posts map[int]*Blog
	mutex sync.Mutex
}

func NewBlogTestApi() *TestApi {
	return &TestApi{
		Posts: make(map[int]*Blog),
	}
}

func (api *TestApi) PostsCount() int {
	return len(api.Posts)
}

func (api *TestApi) CloseDB() {
	// NOP
}

func (api *TestApi) AddBlog(blog *Blog) error {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	if blog.Id == 0 {
		blog.Id = len(api.Posts)
	}

	if _, ok := api.Posts[blog.Id]; ok {
		return errors.New("blog exists already")
	}

	api.Posts[blog.Id] = blog
	return nil
}

func (api *TestApi) UpdateBlog(blog *Blog) error {
	api.mutex.Lock()
	defer api.mutex.Unlock()
	api.Posts[blog.Id] = blog
	return nil
}

func (api *TestApi) DeleteBlog(id int) (bool, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	_, ok := api.Posts[id]
	if !ok {
		return false, nil
	}

	delete(api.Posts, id)

	return true, nil
}

func (api *TestApi) All() ([]*Blog, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()
	var blogs []*Blog
	for id := range api.Posts {
		blogs = append(blogs, api.Posts[id])
	}
	return blogs, nil
}

func (api *TestApi) BlogsCount() (int, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()
	return len(api.Posts), nil
}

func (api *TestApi) GetBlogsPage(page, size int) ([]*Blog, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	if len(api.Posts) <= size {
		return api.All()
	}

	panic("not implemented yet")
}
