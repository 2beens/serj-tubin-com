package internal

import (
	"net/http"

	"github.com/gorilla/mux"
)

type BlogHandler struct {
	blogApi *BlogApi
}

func NewBlogHandler(blogRouter *mux.Router, blogApi *BlogApi) *BlogHandler {
	handler := &BlogHandler{
		blogApi: blogApi,
	}

	blogRouter.HandleFunc("/blogs/new", handler.handleNewBlog).Methods("POST", "OPTIONS").Name("new-blog")

	return handler
}

func (handler *BlogHandler) handleNewBlog(w http.ResponseWriter, r *http.Request) {
	// TODO:
}
