# Bloggernetes

A Kubernetes-native blog platform that watches BlogPost and BlogPage CRDs and exposes them as a web server with Markdown rendering and RSS support.

## Features

- Watches for BlogPost and BlogPage CRDs in a Kubernetes cluster
- Keeps all posts and pages in memory, indexed by ID
- Orders posts by their authored date and pages by their order
- Allows viewing posts from a global view, and filtered by tag or author
- Renders blog post and page content as Markdown
- Provides an RSS feed for blog posts
- Shows extracts of blog posts on index pages
- Modern and beautiful UI using Tailwind CSS with responsive design
- Automatically detects if running in a cluster and uses the pod's service account
- Allows specifying context via flag if not running in a cluster
- Takes flags for namespace to watch and blog name customization

## Architecture

The application consists of the following components:

1. **BlogPost CRD**: Defines the structure of a blog post in Kubernetes
2. **BlogPage CRD**: Defines the structure of a static page in Kubernetes
3. **Controller**: Watches for changes to BlogPost and BlogPage resources and updates the in-memory store
4. **Store**: Keeps all posts and pages in memory, indexed by ID
5. **Web Server**: Exposes the blog posts and pages as a web server with routes for viewing all posts, posts by tag, posts by author, and individual pages

## Future Improvements

I'd like to add the following features in the future:
* Support for multiple blogs, with a Blog CRD to define each blog, which would have a LabelSelector to select the
  BlogPost and BlogPage resources for that blog.
* Make the front-end API-driven instead of using server-side rendering, allowing for switching out the front-end.
* Localization support.

More meta features:
* CI/CD pipeline for building and deploying artifacts.
* Unit tests for the controller and store.

Not planned:
* Comments and other user-generated content, although I'm thinking of options that would make sense.
* Authentication / WYSIWYG creation of posts and pages.

## Running the Application

### Prerequisites

You should be able to build directly with Go, but this repository has a
Nix shell set up to make it easier to get started in a replicable way.

- Nix (https://nixos.org/download.html)

Optionally, if you have direnv installed, you can use it to automatically enter the Nix shell when you `cd` into
the directory. To do this, run:

```
direnv allow
```

### Installation

1. Clone the repository:
   ```
   git clone https://github.com/davies-barnard/bloggernetes.git
   cd bloggernetes
   ```

2. Build the application:
   ```
   bazel build //cmd
   ```

### Running Locally

To run the application locally, you need to specify the path to your kubeconfig file and the namespace to watch:

```
bazel run //cmd -- --namespace=default
```

You can also specify a different Kubernetes context:

```
bazel run //cmd -- --namespace=default --context=my-context
```

### Running in a Kubernetes Cluster

When running in a Kubernetes cluster, the application will automatically use the pod's service account.

### Building and Using the Docker Image

To push the image to a Docker registry:

```
bazel run //cmd:push_image --tag=latest
```

You can also specify a different repository by setting the `repository` attribute in the `oci_push` rule in `cmd/BUILD.bazel`.

## Command Line Flags

- `--namespace`: Namespace to watch for BlogPost and BlogPage resources (default: "default")
- `--addr`: Address to listen on for HTTP requests (default: ":8080")
- `--blog-name`: Name of the blog (default: "Bloggernetes")
- `--kubeconfig`: Path to kubeconfig file (default: "$HOME/.kube/config")
- `--context`: Kubernetes context to use

## Creating a BlogPost

To create a BlogPost, apply a YAML file like the following:

```yaml
apiVersion: alpha.bloggernetes.davies.me.uk/v1
kind: BlogPost
metadata:
  name: my-first-post
spec:
  id: my-first-post
  title: My First Blog Post
  body: |
    # Hello World

    This is my first blog post using Bloggernetes!
  author: user@example.com
  metaDescription: A sample blog post using Bloggernetes
  tags:
    - sample
    - hello-world
  authoredDate: "2023-06-01T12:00:00Z"
```

Apply it to your cluster:

```
kubectl apply -f my-first-post.yaml
```

## Creating a BlogPage

To create a BlogPage, apply a YAML file like the following:

```yaml
apiVersion: alpha.bloggernetes.davies.me.uk/v1
kind: BlogPage
metadata:
  name: about-page
spec:
  id: about
  title: About Us
  content: |
    # About Our Blog

    This is a static page created using Bloggernetes!

    ## Our Mission

    To provide a simple, Kubernetes-native blogging platform.
  order: 1
```

Apply it to your cluster:

```
kubectl apply -f about-page.yaml
```

The `order` field determines the position of the page in the navigation bar. Pages are sorted by their order value in ascending order.

## Accessing the Blog

Once the application is running, you can access the blog at:

```
http://localhost:8080
```

The RSS feed is available at:

```
http://localhost:8080/rss.xml
```

You can use this URL in any RSS reader to subscribe to the blog.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
