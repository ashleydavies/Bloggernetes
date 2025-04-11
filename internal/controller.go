package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/log"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	"k8s.io/client-go/tools/cache"
)

// BlogPostResource defines the GVR for BlogPost CRD
var BlogPostResource = schema.GroupVersionResource{
	Group:    "alpha.bloggernetes.davies.me.uk",
	Version:  "v1",
	Resource: "blogposts",
}

// BlogPageResource defines the GVR for BlogPage CRD
var BlogPageResource = schema.GroupVersionResource{
	Group:    "alpha.bloggernetes.davies.me.uk",
	Version:  "v1",
	Resource: "blogpages",
}

// Controller watches for BlogPost and BlogPage CRD changes and updates the store
type Controller struct {
	client    dynamic.Interface
	store     *Store
	namespace string
	stopCh    chan struct{}
}

// NewController creates a new controller for watching BlogPost CRDs
func NewController(client dynamic.Interface, store *Store, namespace string) *Controller {
	return &Controller{
		client:    client,
		store:     store,
		namespace: namespace,
		stopCh:    make(chan struct{}),
	}
}

// Start starts the controller
func (c *Controller) Start(ctx context.Context) error {
	log.Info("Starting controller", "namespace", c.namespace)

	// Create a factory for dynamic informers
	factory := dynamicinformer.NewFilteredDynamicSharedInformerFactory(
		c.client,
		time.Minute*30,
		c.namespace,
		nil,
	)

	// Create an informer for BlogPost resources
	postInformer := factory.ForResource(BlogPostResource).Informer()

	// Add event handlers for BlogPost resources
	postInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handlePostAdd,
		UpdateFunc: c.handlePostUpdate,
		DeleteFunc: c.handlePostDelete,
	})

	// Create an informer for BlogPage resources
	pageInformer := factory.ForResource(BlogPageResource).Informer()

	// Add event handlers for BlogPage resources
	pageInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    c.handlePageAdd,
		UpdateFunc: c.handlePageUpdate,
		DeleteFunc: c.handlePageDelete,
	})

	// Start the informers
	go postInformer.Run(c.stopCh)
	go pageInformer.Run(c.stopCh)

	// Wait for the informers to sync
	if !cache.WaitForCacheSync(c.stopCh, postInformer.HasSynced, pageInformer.HasSynced) {
		return fmt.Errorf("failed to sync informer caches")
	}

	log.Info("Controller started successfully")

	// Wait for context cancellation
	<-ctx.Done()
	close(c.stopCh)
	return nil
}

// Stop stops the controller
func (c *Controller) Stop() {
	close(c.stopCh)
}

// handlePostAdd handles the addition of a new BlogPost
func (c *Controller) handlePostAdd(obj interface{}) {
	post, err := convertToBlogPost(obj)
	if err != nil {
		log.Error("Failed to convert BlogPost", "error", err)
		return
	}

	log.Info("BlogPost added", "id", post.ID, "title", post.Title)
	c.store.AddOrUpdatePost(post)
}

// handlePostUpdate handles the update of an existing BlogPost
func (c *Controller) handlePostUpdate(oldObj, newObj interface{}) {
	post, err := convertToBlogPost(newObj)
	if err != nil {
		log.Error("Failed to convert BlogPost", "error", err)
		return
	}

	log.Info("BlogPost updated", "id", post.ID, "title", post.Title)
	c.store.AddOrUpdatePost(post)
}

// handlePostDelete handles the deletion of a BlogPost
func (c *Controller) handlePostDelete(obj interface{}) {
	post, err := convertToBlogPost(obj)
	if err != nil {
		log.Error("Failed to convert BlogPost", "error", err)
		return
	}

	log.Info("BlogPost deleted", "id", post.ID, "title", post.Title)
	c.store.DeletePost(post.ID)
}

// handlePageAdd handles the addition of a new BlogPage
func (c *Controller) handlePageAdd(obj interface{}) {
	page, err := convertToBlogPage(obj)
	if err != nil {
		log.Error("Failed to convert BlogPage", "error", err)
		return
	}

	log.Info("BlogPage added", "id", page.ID, "title", page.Title)
	c.store.AddOrUpdatePage(page)
}

// handlePageUpdate handles the update of an existing BlogPage
func (c *Controller) handlePageUpdate(oldObj, newObj interface{}) {
	page, err := convertToBlogPage(newObj)
	if err != nil {
		log.Error("Failed to convert BlogPage", "error", err)
		return
	}

	log.Info("BlogPage updated", "id", page.ID, "title", page.Title)
	c.store.AddOrUpdatePage(page)
}

// handlePageDelete handles the deletion of a BlogPage
func (c *Controller) handlePageDelete(obj interface{}) {
	page, err := convertToBlogPage(obj)
	if err != nil {
		log.Error("Failed to convert BlogPage", "error", err)
		return
	}

	log.Info("BlogPage deleted", "id", page.ID, "title", page.Title)
	c.store.DeletePage(page.ID)
}

// convertToBlogPost converts an unstructured object to a BlogPost
func convertToBlogPost(obj interface{}) (*BlogPost, error) {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("object is not an Unstructured")
	}

	// No need to extract metadata for this conversion

	// Extract spec
	spec, found, err := unstructured.NestedMap(unstructuredObj.Object, "spec")
	if err != nil || !found {
		return nil, fmt.Errorf("spec not found in BlogPost: %v", err)
	}

	// Extract fields from spec
	id, _ := spec["id"].(string)
	title, _ := spec["title"].(string)
	body, _ := spec["body"].(string)
	author, _ := spec["author"].(string)
	metaDescription, _ := spec["metaDescription"].(string)

	// Extract tags
	var tags []string
	if tagsInterface, ok := spec["tags"].([]interface{}); ok {
		for _, tag := range tagsInterface {
			if tagStr, ok := tag.(string); ok {
				tags = append(tags, tagStr)
			}
		}
	}

	// Parse dates
	authoredDate, err := parseDate(spec["authoredDate"])
	if err != nil {
		return nil, fmt.Errorf("failed to parse authoredDate: %v", err)
	}

	var updatedDate *time.Time
	if updatedDateStr, ok := spec["updatedDate"].(string); ok && updatedDateStr != "" {
		parsed, err := parseDate(updatedDateStr)
		if err == nil {
			updatedDate = &parsed
		}
	}

	return &BlogPost{
		ID:              id,
		Title:           title,
		Body:            body,
		Author:          author,
		MetaDescription: metaDescription,
		Tags:            tags,
		AuthoredDate:    authoredDate,
		UpdatedDate:     updatedDate,
	}, nil
}

// parseDate parses a date string from the CRD
func parseDate(dateInterface interface{}) (time.Time, error) {
	dateStr, ok := dateInterface.(string)
	if !ok {
		return time.Time{}, fmt.Errorf("date is not a string")
	}

	// Try RFC3339 format (used by Kubernetes)
	return time.Parse(time.RFC3339, dateStr)
}

// convertToBlogPage converts an unstructured object to a BlogPage
func convertToBlogPage(obj interface{}) (*BlogPage, error) {
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		return nil, fmt.Errorf("object is not an Unstructured")
	}

	// Extract spec
	spec, found, err := unstructured.NestedMap(unstructuredObj.Object, "spec")
	if err != nil || !found {
		return nil, fmt.Errorf("spec not found in BlogPage: %v", err)
	}

	// Extract fields from spec
	id, _ := spec["id"].(string)
	title, _ := spec["title"].(string)
	content, _ := spec["content"].(string)
	order, _ := spec["order"].(int64) // Kubernetes stores numbers as int64

	return &BlogPage{
		ID:      id,
		Title:   title,
		Content: content,
		Order:   int(order), // Convert int64 to int
	}, nil
}
