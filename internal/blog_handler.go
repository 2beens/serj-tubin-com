package internal

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type BlogHandler struct {
	blogApi *BlogApi
}

func NewBlogHandler(blogRouter *mux.Router, blogApi *BlogApi) *BlogHandler {
	handler := &BlogHandler{
		blogApi: blogApi,
	}

	blogRouter.HandleFunc("/new", handler.handleNewBlog).Methods("POST").Name("new-blog")
	blogRouter.HandleFunc("/update/{id}", handler.handleUpdateBlog).Methods("POST").Name("update-blog")
	blogRouter.HandleFunc("/delete/{id}", handler.handleDeleteBlog).Methods("GET").Name("delete-blog")
	blogRouter.HandleFunc("/all", handler.handleAll).Methods("GET").Name("all-blogs")

	return handler
}

func (handler *BlogHandler) handleNewBlog(w http.ResponseWriter, r *http.Request) {
	// TODO:
}

func (handler *BlogHandler) handleUpdateBlog(w http.ResponseWriter, r *http.Request) {
	// TODO:
}

func (handler *BlogHandler) handleDeleteBlog(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	blogIdStr := vars["id"]
	blogId, err := strconv.Atoi(blogIdStr)
	if err != nil {
		log.Errorf("handle delete blog error: %s", err)
		http.Error(w, "parse form error, parameter <blogId>", http.StatusBadRequest)
		return
	}

	// TODO:
	_ = blogId
}

func (handler *BlogHandler) handleAll(w http.ResponseWriter, r *http.Request) {
	allBlogs, err := handler.blogApi.All()
	if err != nil {
		log.Errorf("get all blogs error: %s", err)
		http.Error(w, "get all blogs error", http.StatusInternalServerError)
		return
	}

	allBlogsJson, err := json.Marshal(allBlogs)
	if err != nil {
		log.Errorf("marshal all blogs error: %s", err)
		http.Error(w, "marshal all blogs error", http.StatusInternalServerError)
		return
	}

	WriteResponseBytes(w, "application/json", allBlogsJson)
}
