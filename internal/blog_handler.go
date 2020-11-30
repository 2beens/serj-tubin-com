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
	session *LoginSession
}

func NewBlogHandler(
	blogRouter *mux.Router,
	blogApi *BlogApi,
	session *LoginSession,
) *BlogHandler {
	handler := &BlogHandler{
		blogApi: blogApi,
		session: session,
	}

	blogRouter.HandleFunc("/new", handler.handleNewBlog).Methods("POST", "OPTIONS").Name("new-blog")
	blogRouter.HandleFunc("/update", handler.handleUpdateBlog).Methods("POST", "OPTIONS").Name("update-blog")
	blogRouter.HandleFunc("/delete/{id}", handler.handleDeleteBlog).Methods("GET", "OPTIONS").Name("delete-blog")
	blogRouter.HandleFunc("/all", handler.handleAll).Methods("GET").Name("all-blogs")
	blogRouter.HandleFunc("/all/page/{page}/size/{size}", handler.handleGetPage).Methods("GET").Name("blogs-page")

	blogRouter.Use(handler.authMiddleware())

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

func (handler *BlogHandler) handleGetPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	pageStr := vars["page"]
	page, err := strconv.Atoi(pageStr)
	if err != nil {
		log.Errorf("handle get blogs page, from <page> param: %s", err)
		http.Error(w, "parse form error, parameter <page>", http.StatusBadRequest)
		return
	}
	sizeStr := vars["size"]
	size, err := strconv.Atoi(sizeStr)
	if err != nil {
		log.Errorf("handle get blogs page, from <size> param: %s", err)
		http.Error(w, "parse form error, parameter <size>", http.StatusInternalServerError)
		return
	}

	log.Tracef("get blogs - page %s size %s", pageStr, sizeStr)

	if page < 1 {
		http.Error(w, "invalid page size (has to be non-zero value)", http.StatusInternalServerError)
		return
	}
	if size < 1 {
		http.Error(w, "invalid size (has to be non-zero value)", http.StatusInternalServerError)
		return
	}

	blogPosts, err := handler.blogApi.GetBlogsPage(page, size)
	if err != nil {
		log.Errorf("get blogs error: %s", err)
		http.Error(w, "failed to get blog posts", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")

	if len(blogPosts) == 0 {
		WriteResponse(w, "application/json", "[]")
		return
	}

	blogPostsJson, err := json.Marshal(blogPosts)
	if err != nil {
		log.Errorf("marshal blogs error: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	WriteResponseBytes(w, "application/json", blogPostsJson)
}

func (handler *BlogHandler) authMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "OPTIONS" {
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.WriteHeader(http.StatusOK)
				return
			}

			// allow getting all blog posts, but not editing
			if r.URL.Path == "/blog/all" {
				next.ServeHTTP(w, r)
				return
			}

			authToken := r.Header.Get("X-SERJ-TOKEN")
			if authToken == "" || handler.session.Token == "" {
				http.Error(w, "no can do", http.StatusUnauthorized)
				return
			}

			if handler.session.Token != authToken {
				http.Error(w, "no can do", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
