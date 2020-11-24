package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

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
	blogRouter.HandleFunc("/update", handler.handleUpdateBlog).Methods("POST").Name("update-blog")
	blogRouter.HandleFunc("/delete/{id}", handler.handleDeleteBlog).Methods("GET").Name("delete-blog")
	blogRouter.HandleFunc("/all", handler.handleAll).Methods("GET").Name("all-blogs")

	return handler
}

func (handler *BlogHandler) handleNewBlog(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Errorf("add new blog failed, parse form error: %s", err)
		http.Error(w, "parse form error", http.StatusInternalServerError)
		return
	}

	title := r.Form.Get("title")
	if title == "" {
		http.Error(w, "error, title empty", http.StatusBadRequest)
		return
	}

	content := r.Form.Get("content")
	if content == "" {
		http.Error(w, "error, content empty", http.StatusBadRequest)
		return
	}

	newBlog := &Blog{
		Title:     title,
		CreatedAt: time.Now(),
		Content:   content,
	}

	if err := handler.blogApi.AddBlog(newBlog); err != nil {
		log.Errorf("add new blog failed: %s", err)
		http.Error(w, "add new blog failed", http.StatusInternalServerError)
		return
	}

	log.Tracef("new blog %d: [%s] added", newBlog.Id, newBlog.Title)

	// TODO: refactor and unify responses
	WriteResponse(w, "", fmt.Sprintf("added:%d", newBlog.Id))
}

func (handler *BlogHandler) handleUpdateBlog(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.Errorf("update blog failed, parse form error: %s", err)
		http.Error(w, "parse form error", http.StatusInternalServerError)
		return
	}

	idStr := r.Form.Get("id")
	if idStr == "" {
		http.Error(w, "error, id empty", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "error, id NaN", http.StatusBadRequest)
		return
	}

	title := r.Form.Get("title")
	if title == "" {
		http.Error(w, "error, title empty", http.StatusBadRequest)
		return
	}

	content := r.Form.Get("content")
	if content == "" {
		http.Error(w, "error, content empty", http.StatusBadRequest)
		return
	}

	blog := &Blog{
		Id:        id,
		Title:     title,
		CreatedAt: time.Now(),
		Content:   content,
	}

	if err := handler.blogApi.UpdateBlog(blog); err != nil {
		log.Errorf("update blog failed: %s", err)
		http.Error(w, "update blog failed", http.StatusInternalServerError)
		return
	}

	WriteResponse(w, "", fmt.Sprintf("updated:%d", blog.Id))
}

func (handler *BlogHandler) handleDeleteBlog(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	idStr := vars["id"]
	if idStr == "" {
		http.Error(w, "error, id empty", http.StatusBadRequest)
		return
	}
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "error, id NaN", http.StatusBadRequest)
		return
	}

	deleted, err := handler.blogApi.DeleteBlog(id)
	if err != nil {
		log.Printf("failed to delete blog %d: %s", id, err)
		http.Error(w, "error, blog not deleted, internal server error", http.StatusInternalServerError)
		return
	}

	if deleted {
		WriteResponse(w, "", fmt.Sprintf("deleted:%d", id))
	} else {
		WriteResponse(w, "", fmt.Sprintf("not-deleted:%d", id))
	}
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
