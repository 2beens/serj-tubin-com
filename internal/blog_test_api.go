package internal

import "sync"

type BlogTestApi struct {
	posts map[int]*Blog
	mutex sync.Mutex
}

func NewBlogTestApi() *BlogTestApi {
	return &BlogTestApi{
		posts: make(map[int]*Blog),
	}
}

func (api *BlogTestApi) CloseDB() {
	// NOP
}

func (api *BlogTestApi) AddBlog(blog *Blog) error {
	api.mutex.Lock()
	defer api.mutex.Unlock()
	api.posts[blog.Id] = blog
	return nil
}

func (api *BlogTestApi) UpdateBlog(blog *Blog) error {
	api.mutex.Lock()
	defer api.mutex.Unlock()
	api.posts[blog.Id] = blog
	return nil
}

func (api *BlogTestApi) DeleteBlog(id int) (bool, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	_, ok := api.posts[id]
	if !ok {
		return false, nil
	}

	delete(api.posts, id)

	return true, nil
}

func (api *BlogTestApi) All() ([]*Blog, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()
	var blogs []*Blog
	for id, _ := range api.posts {
		blogs = append(blogs, api.posts[id])
	}
	return blogs, nil
}

func (api *BlogTestApi) BlogsCount() (int, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()
	return len(api.posts), nil
}

func (api *BlogTestApi) GetBlogsPage(page, size int) ([]*Blog, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	if len(api.posts) <= size {
		return api.All()
	}

	panic("not implemented yet")
}
