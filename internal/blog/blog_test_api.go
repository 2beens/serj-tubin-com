package blog

import "sync"

type TestApi struct {
	posts map[int]*Blog
	mutex sync.Mutex
}

func NewBlogTestApi() *TestApi {
	return &TestApi{
		posts: make(map[int]*Blog),
	}
}

func (api *TestApi) PostsCount() int {
	return len(api.posts)
}

func (api *TestApi) CloseDB() {
	// NOP
}

func (api *TestApi) AddBlog(blog *Blog) error {
	api.mutex.Lock()
	defer api.mutex.Unlock()
	api.posts[blog.Id] = blog
	return nil
}

func (api *TestApi) UpdateBlog(blog *Blog) error {
	api.mutex.Lock()
	defer api.mutex.Unlock()
	api.posts[blog.Id] = blog
	return nil
}

func (api *TestApi) DeleteBlog(id int) (bool, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	_, ok := api.posts[id]
	if !ok {
		return false, nil
	}

	delete(api.posts, id)

	return true, nil
}

func (api *TestApi) All() ([]*Blog, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()
	var blogs []*Blog
	for id := range api.posts {
		blogs = append(blogs, api.posts[id])
	}
	return blogs, nil
}

func (api *TestApi) BlogsCount() (int, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()
	return len(api.posts), nil
}

func (api *TestApi) GetBlogsPage(page, size int) ([]*Blog, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	if len(api.posts) <= size {
		return api.All()
	}

	panic("not implemented yet")
}
