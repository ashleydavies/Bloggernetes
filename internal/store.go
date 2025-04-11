package internal

import (
	"sync"
)

// Store is an in-memory store for blog posts and pages
type Store struct {
	mu    sync.RWMutex
	posts map[string]*BlogPost // Indexed by ID
	pages map[string]*BlogPage // Indexed by ID
}

// NewStore creates a new in-memory store for blog posts and pages
func NewStore() *Store {
	return &Store{
		posts: make(map[string]*BlogPost),
		pages: make(map[string]*BlogPage),
	}
}

// AddOrUpdatePost adds or updates a blog post in the store
func (s *Store) AddOrUpdatePost(post *BlogPost) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.posts[post.ID] = post
}

// DeletePost deletes a blog post from the store
func (s *Store) DeletePost(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.posts, id)
}

// GetPost retrieves a blog post by ID
func (s *Store) GetPost(id string) (*BlogPost, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	post, exists := s.posts[id]
	return post, exists
}

// GetAllPosts returns all blog posts sorted by authored date (newest first)
func (s *Store) GetAllPosts() []*BlogPost {
	s.mu.RLock()
	defer s.mu.RUnlock()

	posts := make([]*BlogPost, 0, len(s.posts))
	for _, post := range s.posts {
		posts = append(posts, post)
	}

	SortByAuthoredDate(posts)
	return posts
}

// GetPostsByTag returns all blog posts with the specified tag, sorted by authored date
func (s *Store) GetPostsByTag(tag string) []*BlogPost {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []*BlogPost
	for _, post := range s.posts {
		for _, t := range post.Tags {
			if t == tag {
				filtered = append(filtered, post)
				break
			}
		}
	}

	SortByAuthoredDate(filtered)
	return filtered
}

// GetPostsByAuthor returns all blog posts by the specified author, sorted by authored date
func (s *Store) GetPostsByAuthor(author string) []*BlogPost {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []*BlogPost
	for _, post := range s.posts {
		if post.Author == author {
			filtered = append(filtered, post)
		}
	}

	SortByAuthoredDate(filtered)
	return filtered
}

// GetAllTags returns all unique tags used in blog posts
func (s *Store) GetAllTags() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tagMap := make(map[string]struct{})
	for _, post := range s.posts {
		for _, tag := range post.Tags {
			tagMap[tag] = struct{}{}
		}
	}

	tags := make([]string, 0, len(tagMap))
	for tag := range tagMap {
		tags = append(tags, tag)
	}

	return tags
}

// GetAllAuthors returns all unique authors of blog posts
func (s *Store) GetAllAuthors() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	authorMap := make(map[string]struct{})
	for _, post := range s.posts {
		authorMap[post.Author] = struct{}{}
	}

	authors := make([]string, 0, len(authorMap))
	for author := range authorMap {
		authors = append(authors, author)
	}

	return authors
}

// AddOrUpdatePage adds or updates a blog page in the store
func (s *Store) AddOrUpdatePage(page *BlogPage) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pages[page.ID] = page
}

// DeletePage deletes a blog page from the store
func (s *Store) DeletePage(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pages, id)
}

// GetPage retrieves a blog page by ID
func (s *Store) GetPage(id string) (*BlogPage, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	page, exists := s.pages[id]
	return page, exists
}

// GetAllPages returns all blog pages sorted by order
func (s *Store) GetAllPages() []*BlogPage {
	s.mu.RLock()
	defer s.mu.RUnlock()

	pages := make([]*BlogPage, 0, len(s.pages))
	for _, page := range s.pages {
		pages = append(pages, page)
	}

	SortByOrder(pages)
	return pages
}
