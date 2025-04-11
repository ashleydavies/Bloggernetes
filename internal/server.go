package internal

import (
	"context"
	"embed"
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/log"
)

// Templates contains the embedded HTML templates
//
//go:embed templates/*
var Templates embed.FS

// RSS feed structures
type RSSItem struct {
	XMLName     xml.Name `xml:"item"`
	Title       string   `xml:"title"`
	Link        string   `xml:"link"`
	Description string   `xml:"description"`
	PubDate     string   `xml:"pubDate"`
	GUID        string   `xml:"guid"`
	Author      string   `xml:"author,omitempty"`
}

type RSSChannel struct {
	XMLName       xml.Name  `xml:"channel"`
	Title         string    `xml:"title"`
	Link          string    `xml:"link"`
	Description   string    `xml:"description"`
	Language      string    `xml:"language"`
	LastBuildDate string    `xml:"lastBuildDate"`
	Generator     string    `xml:"generator"`
	Items         []RSSItem `xml:"item"`
}

type RSS struct {
	XMLName xml.Name   `xml:"rss"`
	Version string     `xml:"version,attr"`
	Channel RSSChannel `xml:"channel"`
}

// Server is the HTTP server for the blog
type Server struct {
	store      *Store
	templates  map[string]*template.Template
	Addr       string
	blogName   string
	httpServer *http.Server
}

// templateData holds common data for templates
type templateData map[string]interface{}

// baseData returns the common data for all templates
func (s *Server) baseData() templateData {
	return templateData{
		"BlogName": s.blogName,
		"Tags":     s.store.GetAllTags(),
		"Authors":  s.store.GetAllAuthors(),
		"Pages":    s.store.GetAllPages(),
	}
}

// render executes the template with the given name and data
func (s *Server) render(w http.ResponseWriter, name string, data templateData) {
	if err := s.templates[name].Execute(w, data); err != nil {
		log.Error("Failed to render template", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// NewServer creates a new HTTP server for the blog
func NewServer(store *Store, addr string, blogName string) (*Server, error) {
	// Initialize a map to store templates for each page
	templates := make(map[string]*template.Template)

	// Define the page templates to create
	pageTemplates := []struct {
		name     string
		filename string
	}{
		{"home", "templates/home.html"},
		{"tag", "templates/tag.html"},
		{"author", "templates/author.html"},
		{"post", "templates/post.html"},
		{"page", "templates/page.html"},
	}

	// Read the layout template content once
	layoutContent, err := Templates.ReadFile("templates/layout.html")
	if err != nil {
		return nil, fmt.Errorf("failed to read layout template: %w", err)
	}

	// Create a template for each page
	for _, page := range pageTemplates {
		// Create a template with the layout content
		tmpl, err := parseTemplateWithLayout(page.name, page.filename, layoutContent)
		if err != nil {
			return nil, err
		}

		// Add the template to the map
		templates[page.name] = tmpl
	}

	return &Server{
		store:     store,
		templates: templates,
		Addr:      addr,
		blogName:  blogName,
	}, nil
}

// parseTemplateWithLayout parses a template with the layout content
func parseTemplateWithLayout(name, filename string, layoutContent []byte) (*template.Template, error) {
	// Create a new template with the layout content
	tmpl := template.New("layout.html")
	tmpl, err := tmpl.Parse(string(layoutContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse layout template for %s: %w", name, err)
	}

	// Read the page template content
	pageContent, err := Templates.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s template: %w", name, err)
	}

	// Parse the page template
	tmpl, err = tmpl.Parse(string(pageContent))
	if err != nil {
		return nil, fmt.Errorf("failed to parse %s template: %w", name, err)
	}

	return tmpl, nil
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	mux := s.setupRoutes()

	s.httpServer = &http.Server{
		Addr:    s.Addr,
		Handler: mux,
	}

	// Channel to signal when the server has shut down and to communicate errors
	serverError := make(chan error, 1)
	serverShutdown := make(chan struct{})

	// Start server in a goroutine
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Server error", "error", err)
			serverError <- err
		}
		close(serverShutdown)
	}()

	// Wait for context cancellation in a goroutine
	go func() {
		<-ctx.Done()
		log.Info("Shutting down server...")

		// Create a timeout context for shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			log.Error("Server shutdown error", "error", err)
		}
	}()

	// Check if there was an immediate error (like port already in use)
	select {
	case err := <-serverError:
		return fmt.Errorf("failed to start server: %w", err)
	case <-time.After(100 * time.Millisecond):
		// If no error after a short delay, assume server started successfully
		return nil
	}
}

// setupRoutes configures and returns the HTTP routes
func (s *Server) setupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// Serve static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(Templates))))

	// Home page - all posts
	mux.HandleFunc("/", s.handleHome)

	// Posts by tag
	mux.HandleFunc("/tag/", s.handleTag)

	// Posts by author
	mux.HandleFunc("/author/", s.handleAuthor)

	// Individual post
	mux.HandleFunc("/post/", s.handlePost)

	// Individual page
	mux.HandleFunc("/page/", s.handlePage)

	// RSS feed
	mux.HandleFunc("/rss.xml", s.handleRSS)

	return mux
}

// handleHome handles requests to the home page
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data := s.baseData()
	data["Title"] = "Home"
	data["Posts"] = s.store.GetAllPosts()

	s.render(w, "home", data)
}

// handleTag handles requests to filter posts by tag
func (s *Server) handleTag(w http.ResponseWriter, r *http.Request) {
	tag := strings.TrimPrefix(r.URL.Path, "/tag/")
	if tag == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := s.baseData()
	data["Title"] = fmt.Sprintf("Posts tagged with %s", tag)
	data["Tag"] = tag
	data["Posts"] = s.store.GetPostsByTag(tag)
	data["FilterBy"] = "tag"

	s.render(w, "tag", data)
}

// handleAuthor handles requests to filter posts by author
func (s *Server) handleAuthor(w http.ResponseWriter, r *http.Request) {
	author := strings.TrimPrefix(r.URL.Path, "/author/")
	if author == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	data := s.baseData()
	data["Title"] = fmt.Sprintf("Posts by %s", author)
	data["Author"] = author
	data["Posts"] = s.store.GetPostsByAuthor(author)
	data["FilterBy"] = "author"

	s.render(w, "author", data)
}

// handlePost handles requests to view a single post
func (s *Server) handlePost(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/post/")
	if id == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	post, exists := s.store.GetPost(id)
	if !exists {
		http.NotFound(w, r)
		return
	}

	data := s.baseData()
	data["Title"] = post.Title
	data["Post"] = post

	s.render(w, "post", data)
}

// handlePage handles requests to view a single page
func (s *Server) handlePage(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/page/")
	if id == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	page, exists := s.store.GetPage(id)
	if !exists {
		http.NotFound(w, r)
		return
	}

	data := s.baseData()
	data["Title"] = page.Title
	data["Page"] = page
	data["PageID"] = page.ID

	s.render(w, "page", data)
}

// handleRSS handles requests for the RSS feed
func (s *Server) handleRSS(w http.ResponseWriter, r *http.Request) {
	posts := s.store.GetAllPosts()

	// Set content type
	w.Header().Set("Content-Type", "application/rss+xml")

	// Create RSS feed
	rss := RSS{
		Version: "2.0",
		Channel: RSSChannel{
			Title:         s.blogName,
			Link:          fmt.Sprintf("http://%s", r.Host),
			Description:   fmt.Sprintf("%s - A Kubernetes-native blog", s.blogName),
			Language:      "en-us",
			LastBuildDate: time.Now().Format(time.RFC1123Z),
			Generator:     "Bloggernetes",
		},
	}

	// Add items to the feed
	for _, post := range posts {
		description := getPostDescription(post)

		item := RSSItem{
			Title:       post.Title,
			Link:        fmt.Sprintf("http://%s/post/%s", r.Host, post.ID),
			Description: description,
			PubDate:     post.AuthoredDate.Format(time.RFC1123Z),
			GUID:        fmt.Sprintf("http://%s/post/%s", r.Host, post.ID),
			Author:      post.Author,
		}

		rss.Channel.Items = append(rss.Channel.Items, item)
	}

	// Marshal to XML
	output, err := xml.MarshalIndent(rss, "", "  ")
	if err != nil {
		log.Error("Failed to marshal RSS feed", "error", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Add XML header
	w.Write([]byte(xml.Header))
	w.Write(output)
}

// getPostDescription returns the description for a blog post
func getPostDescription(post *BlogPost) string {
	if post.MetaDescription != "" {
		return post.MetaDescription
	}

	// Use the first 200 characters of the body as description
	if len(post.Body) > 200 {
		return post.Body[:200] + "..."
	}
	return post.Body
}
