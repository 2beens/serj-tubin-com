package blog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/2beens/serjtubincom/internal/auth"
	"github.com/2beens/serjtubincom/pkg"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type clapBlogRequest struct {
	ID int `json:"id"`
}

type newBlogRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
}

type updateBlogRequest struct {
	ID      int    `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type Handler struct {
	blogApi      Api
	loginChecker auth.Checker
}

func NewBlogHandler(
	blogApi Api,
	loginChecker auth.Checker,
) *Handler {
	return &Handler{
		blogApi:      blogApi,
		loginChecker: loginChecker,
	}
}

func (handler *Handler) SetupRoutes(blogRouter *mux.Router) {
	blogRouter.HandleFunc("/new", handler.handleNewBlog).Methods("POST", "OPTIONS").Name("new-blog")
	blogRouter.HandleFunc("/update", handler.handleUpdateBlog).Methods("POST", "OPTIONS").Name("update-blog")
	blogRouter.HandleFunc("/clap", handler.handleBlogClapped).Methods("PATCH", "OPTIONS").Name("blog-clapped")
	blogRouter.HandleFunc("/delete/{id}", handler.handleDeleteBlog).Methods("DELETE", "OPTIONS").Name("delete-blog")
	blogRouter.HandleFunc("/all", handler.handleAll).Methods("GET").Name("all-blogs")
	blogRouter.HandleFunc("/page/{page}/size/{size}", handler.handleGetPage).Methods("GET").Name("blogs-page")

	blogRouter.Use(handler.authMiddleware())
}

func (handler *Handler) handleNewBlog(w http.ResponseWriter, r *http.Request) {
	var newBlogReq newBlogRequest
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&newBlogReq); err != nil {
			log.Errorf("new blog, unmarshal json params: %s", err)
			http.Error(w, "add blog failed", http.StatusBadRequest)
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
			log.Errorf("add new blog failed, parse form error: %s", err)
			http.Error(w, "parse form error", http.StatusInternalServerError)
			return
		}
		newBlogReq = newBlogRequest{
			Title:   r.Form.Get("title"),
			Content: r.Form.Get("content"),
		}
	}

	if newBlogReq.Title == "" {
		http.Error(w, "error, title empty", http.StatusBadRequest)
		return
	}
	if newBlogReq.Content == "" {
		http.Error(w, "error, content empty", http.StatusBadRequest)
		return
	}

	newBlog := &Blog{
		Title:     newBlogReq.Title,
		Content:   newBlogReq.Content,
		CreatedAt: time.Now(),
	}

	if err := handler.blogApi.AddBlog(r.Context(), newBlog); err != nil {
		log.Errorf("add new blog failed: %s", err)
		http.Error(w, "add new blog failed", http.StatusInternalServerError)
		return
	}

	log.Tracef("new blog %d: [%s] added", newBlog.Id, newBlog.Title)

	// TODO: refactor and unify responses
	pkg.WriteResponse(w, "", fmt.Sprintf("added:%d", newBlog.Id))
}

func (handler *Handler) handleUpdateBlog(w http.ResponseWriter, r *http.Request) {
	var updateBlogReq updateBlogRequest
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&updateBlogReq); err != nil {
			log.Errorf("update blog, unmarshal json params: %s", err)
			http.Error(w, "update blog failed", http.StatusBadRequest)
			return
		}
	} else {
		if err := r.ParseForm(); err != nil {
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
		updateBlogReq = updateBlogRequest{
			ID:      id,
			Title:   r.Form.Get("title"),
			Content: r.Form.Get("content"),
		}
	}

	if updateBlogReq.Title == "" {
		http.Error(w, "error, title empty", http.StatusBadRequest)
		return
	}
	if updateBlogReq.Content == "" {
		http.Error(w, "error, content empty", http.StatusBadRequest)
		return
	}

	if err := handler.blogApi.UpdateBlog(r.Context(), updateBlogReq.ID, updateBlogReq.Title, updateBlogReq.Content); err != nil {
		log.Errorf("update blog failed: %s", err)
		http.Error(w, "update blog failed", http.StatusInternalServerError)
		return
	}

	pkg.WriteResponse(w, "", fmt.Sprintf("updated:%d", updateBlogReq.ID))
}

func (handler *Handler) handleBlogClapped(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Errorf("update blog failed, parse form error: %s", err)
		http.Error(w, "parse form error", http.StatusInternalServerError)
		return
	}

	var clapBlogReq clapBlogRequest
	if r.Header.Get("Content-Type") == "application/json" {
		if err := json.NewDecoder(r.Body).Decode(&clapBlogReq); err != nil {
			log.Errorf("blog clap, unmarshal json params: %s", err)
			http.Error(w, "update blog failed", http.StatusBadRequest)
			return
		}
	} else {
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
		clapBlogReq = clapBlogRequest{
			ID: id,
		}
	}

	if err := handler.blogApi.BlogClapped(r.Context(), clapBlogReq.ID); err != nil {
		log.Errorf("update blog failed: %s", err)
		http.Error(w, "update blog failed", http.StatusInternalServerError)
		return
	}

	pkg.WriteResponse(w, "", fmt.Sprintf("updated:%d", clapBlogReq.ID))
}

func (handler *Handler) handleDeleteBlog(w http.ResponseWriter, r *http.Request) {
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

	if err := handler.blogApi.DeleteBlog(r.Context(), id); err != nil {
		log.Printf("failed to delete blog %d: %s", id, err)
		http.Error(w, "error, blog not deleted, internal server error", http.StatusInternalServerError)
		return
	}

	pkg.WriteResponse(w, "", fmt.Sprintf("deleted:%d", id))
}

func (handler *Handler) handleAll(w http.ResponseWriter, r *http.Request) {
	allBlogs, err := handler.blogApi.All(r.Context())

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

	pkg.WriteResponseBytes(w, "application/json", allBlogsJson)
}

func (handler *Handler) handleGetPage(w http.ResponseWriter, r *http.Request) {
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

	blogPosts, err := handler.blogApi.GetBlogsPage(r.Context(), page, size)
	if err != nil {
		log.Errorf("get blogs error: %s", err)
		http.Error(w, "failed to get blog posts", http.StatusInternalServerError)
		return
	}

	w.Header().Add("Content-Type", "application/json")

	if len(blogPosts) == 0 {
		blogPosts = []*Blog{}
	}

	blogPostsJson, err := json.Marshal(blogPosts)
	if err != nil {
		log.Errorf("marshal blogs error: %s", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	totalBlogsCount, err := handler.blogApi.BlogsCount(r.Context())
	if err != nil {
		log.Errorf("get blogs error: %s", err)
		http.Error(w, "failed to get blog posts", http.StatusInternalServerError)
		return
	}

	resJson := fmt.Sprintf(`{"posts": %s, "total": %d}`, blogPostsJson, totalBlogsCount)

	pkg.WriteResponseBytes(w, "application/json", []byte(resJson))
}

func (handler *Handler) authMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions {
				w.Header().Set("Access-Control-Allow-Headers", "*")
				w.WriteHeader(http.StatusOK)
				return
			}

			// allow getting blog posts, but not editing
			// TODO: find a better way to mark routes auth-free
			switch {
			case strings.HasPrefix(r.URL.Path, "/blog/page/"),
				r.URL.Path == "/blog/all",
				r.URL.Path == "/blog/clap":
				next.ServeHTTP(w, r)
				return
			}

			authToken := r.Header.Get("X-SERJ-TOKEN")
			if authToken == "" {
				log.Tracef("[missing token] unauthorized => %s", r.URL.Path)
				http.Error(w, "no can do", http.StatusUnauthorized)
				return
			}

			isLogged, err := handler.loginChecker.IsLogged(r.Context(), authToken)
			if err != nil {
				log.Tracef("[failed login check] => %s: %s", r.URL.Path, err)
				http.Error(w, "no can do", http.StatusUnauthorized)
				return
			}
			if !isLogged {
				log.Tracef("[invalid token] unauthorized => %s", r.URL.Path)
				http.Error(w, "no can do", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
