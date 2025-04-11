package internal

import (
	"sort"
)

// BlogPage represents a blog page from the Kubernetes CRD
type BlogPage struct {
	ID      string
	Title   string
	Content string
	Order   int
}

// BlogPages is a slice of BlogPage that can be sorted by Order
type BlogPages []*BlogPage

// Implement sort.Interface for BlogPages
func (b BlogPages) Len() int           { return len(b) }
func (b BlogPages) Swap(i, j int)      { b[i], b[j] = b[j], b[i] }
func (b BlogPages) Less(i, j int) bool { return b[i].Order < b[j].Order }

// SortByOrder sorts the blog pages by Order in ascending order
func SortByOrder(pages []*BlogPage) {
	sort.Sort(BlogPages(pages))
}