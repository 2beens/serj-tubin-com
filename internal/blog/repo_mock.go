package blog

import (
	"context"
	"errors"
	"sort"
	"sync"
)

var _ blogRepo = (*repoMock)(nil)

type repoMock struct {
	Posts map[int]*Blog
	mutex sync.Mutex
}

func newRepoMock() *repoMock {
	return &repoMock{
		Posts: make(map[int]*Blog),
	}
}

func (r *repoMock) PostsCount() int {
	return len(r.Posts)
}

func (r *repoMock) AddBlog(_ context.Context, blog *Blog) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if blog.Id == 0 {
		blog.Id = len(r.Posts)
	}

	if _, ok := r.Posts[blog.Id]; ok {
		return errors.New("blog exists already")
	}

	r.Posts[blog.Id] = blog
	return nil
}

func (r *repoMock) UpdateBlog(_ context.Context, id int, title, content string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.Posts[id].Title = title
	r.Posts[id].Content = content
	return nil
}

func (r *repoMock) BlogClapped(_ context.Context, id int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if b, found := r.Posts[id]; !found {
		return ErrBlogNotFound
	} else {
		b.Claps++
	}

	return nil
}

func (r *repoMock) DeleteBlog(_ context.Context, id int) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	_, ok := r.Posts[id]
	if !ok {
		return ErrBlogNotFound
	}

	delete(r.Posts, id)

	return nil
}

func (r *repoMock) All(_ context.Context) ([]*Blog, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	var blogs []*Blog
	for id := range r.Posts {
		blogs = append(blogs, r.Posts[id])
	}
	return blogs, nil
}

func (r *repoMock) BlogsCount(_ context.Context) (int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	return len(r.Posts), nil
}

func (r *repoMock) GetBlogsPage(ctx context.Context, page, size int) ([]*Blog, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if len(r.Posts) <= size {
		return r.All(ctx)
	}

	var allPosts []*Blog
	for id := range r.Posts {
		allPosts = append(allPosts, r.Posts[id])
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
