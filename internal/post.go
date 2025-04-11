package internal

import (
	"sort"
	"time"
)

// BlogPost represents a blog post from the Kubernetes CRD
type BlogPost struct {
	ID              string
	Title           string
	MetaDescription string
	Body            string
	Author          string
	Tags            []string
	AuthoredDate    time.Time
	UpdatedDate     *time.Time
}

// BlogPosts is a slice of BlogPost that can be sorted by AuthoredDate
type BlogPosts []*BlogPost

// Implement sort.Interface for BlogPosts
func (b BlogPosts) Len() int           { return len(b) }
func (b BlogPosts) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b BlogPosts) Less(i, j int) bool { return b[i].AuthoredDate.After(b[j].AuthoredDate) }

// SortByAuthoredDate sorts the blog posts by AuthoredDate in descending order (newest first)
func SortByAuthoredDate(posts []*BlogPost) {
	sort.Sort(BlogPosts(posts))
}
