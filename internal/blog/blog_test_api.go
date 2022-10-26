package blog

import (
	"context"
	"errors"
	"sort"
	"sync"
)

var _ Api = (*TestApi)(nil)

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

func (api *TestApi) AddBlog(_ context.Context, blog *Blog) error {
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

func (api *TestApi) UpdateBlog(_ context.Context, id int, title, content string) error {
	api.mutex.Lock()
	defer api.mutex.Unlock()
	api.Posts[id].Title = title
	api.Posts[id].Content = content
	return nil
}

func (api *TestApi) BlogClapped(_ context.Context, id int) error {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	if b, found := api.Posts[id]; !found {
		return ErrBlogNotFound
	} else {
		b.Claps++
	}

	return nil
}

func (api *TestApi) DeleteBlog(_ context.Context, id int) error {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	_, ok := api.Posts[id]
	if !ok {
		return ErrBlogNotFound
	}

	delete(api.Posts, id)

	return nil
}

func (api *TestApi) All(_ context.Context) ([]*Blog, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()
	var blogs []*Blog
	for id := range api.Posts {
		blogs = append(blogs, api.Posts[id])
	}
	return blogs, nil
}

func (api *TestApi) BlogsCount(_ context.Context) (int, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()
	return len(api.Posts), nil
}

func (api *TestApi) GetBlogsPage(ctx context.Context, page, size int) ([]*Blog, error) {
	api.mutex.Lock()
	defer api.mutex.Unlock()

	if len(api.Posts) <= size {
		return api.All(ctx)
	}

	var allPosts []*Blog
	for id := range api.Posts {
		allPosts = append(allPosts, api.Posts[id])
	}

	sort.Slice(allPosts, func(i, j int) bool {
		return allPosts[i].CreatedAt.Before(allPosts[j].CreatedAt)
	})

	startIndex := (page - 1) * size
	endIndex := startIndex + size

	// overflow
	if startIndex >= len(allPosts) {
		return []*Blog{}, nil
	}

	return allPosts[startIndex:endIndex], nil
}
