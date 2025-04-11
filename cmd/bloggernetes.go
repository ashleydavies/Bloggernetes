package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/ashleydavies/bloggernetes/internal"
	"github.com/charmbracelet/log"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// Options holds the command line options for the application
type Options struct {
	Namespace   string
	Kubeconfig  string
	ContextName string
	Addr        string
	BlogName    string
}

// parseFlags parses the command line flags and returns the options
func parseFlags() *Options {
	opts := &Options{}

	flag.StringVar(&opts.Namespace, "namespace", "default", "Namespace to watch for BlogPost resources")
	flag.StringVar(&opts.Addr, "addr", ":8080", "Address to listen on for HTTP requests")
	flag.StringVar(&opts.BlogName, "blog-name", "Bloggernetes", "Name of the blog")

	// Determine if we're running in a cluster
	if isRunningInCluster() {
		// In-cluster, no need for kubeconfig
		log.Info("Running in-cluster, using service account")
	} else {
		// Not in-cluster, need kubeconfig
		if home := homedir.HomeDir(); home != "" {
			flag.StringVar(&opts.Kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "Path to kubeconfig file")
		} else {
			flag.StringVar(&opts.Kubeconfig, "kubeconfig", "", "Path to kubeconfig file")
		}
		flag.StringVar(&opts.ContextName, "context", "", "Kubernetes context to use")
	}

	flag.Parse()
	return opts
}

// isRunningInCluster returns true if the application is running in a Kubernetes cluster
func isRunningInCluster() bool {
	_, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount/token")
	return err == nil
}

// setupSignalHandler sets up signal handling for graceful shutdown
func setupSignalHandler(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(ctx)

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-signalCh
		log.Info("Received shutdown signal")
		cancel()
	}()

	return ctx, cancel
}

// createKubernetesClient creates a Kubernetes client based on the options
func createKubernetesClient(opts *Options) (dynamic.Interface, error) {
	var config *rest.Config
	var err error

	if isRunningInCluster() {
		// In-cluster configuration
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create in-cluster config: %w", err)
		}
	} else {
		// Out-of-cluster configuration
		loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
		loadingRules.ExplicitPath = opts.Kubeconfig

		configOverrides := &clientcmd.ConfigOverrides{}
		if opts.ContextName != "" {
			configOverrides.CurrentContext = opts.ContextName
		}

		kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)
		config, err = kubeConfig.ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to create out-of-cluster config: %w", err)
		}
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	return dynamicClient, nil
}

// Components holds the application components
type Components struct {
	Store      *internal.Store
	Controller *internal.Controller
	Server     *internal.Server
}

// createComponents creates the application components based on the options and the Kubernetes client
func createComponents(opts *Options, client dynamic.Interface) (*Components, error) {
	// Create store
	store := internal.NewStore()

	// Create controller
	controller := internal.NewController(client, store, opts.Namespace)

	// Create server
	server, err := internal.NewServer(store, opts.Addr, opts.BlogName)
	if err != nil {
		return nil, fmt.Errorf("failed to create server: %w", err)
	}

	return &Components{
		Store:      store,
		Controller: controller,
		Server:     server,
	}, nil
}

// startApplication starts the application components
func startApplication(ctx context.Context, components *Components) error {
	// Start controller in a goroutine
	go func() {
		if err := components.Controller.Start(ctx); err != nil {
			log.Error("Controller error", "error", err)
		}
	}()

	// Start server
	log.Info("Starting server", "address", components.Server.Addr)
	if err := components.Server.Start(ctx); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

func main() {
	// Parse command line flags
	opts := parseFlags()

	// Create a context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	ctx, cancel = setupSignalHandler(ctx)
	defer cancel()

	// Create Kubernetes client
	client, err := createKubernetesClient(opts)
	if err != nil {
		log.Fatal("Failed to create Kubernetes client", "error", err)
	}

	// Create application components
	components, err := createComponents(opts, client)
	if err != nil {
		log.Fatal("Failed to create application components", "error", err)
	}

	// Start application
	if err := startApplication(ctx, components); err != nil {
		log.Fatal("Failed to start application", "error", err)
	}

	// Wait for context to be cancelled (e.g., by Ctrl+C)
	<-ctx.Done()
	log.Info("Shutting down gracefully...")
}
