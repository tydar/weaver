# weaver

Minimal custom static-site generator for my site [https://blog.drnll.com](https://blog.drnll.com).

### spec

Given a `/posts/` directory with a number of blog posts saved as Markdown files, this program will generate:

1) An index page with a welcome message, a list of links to the latest posts (e.g. posts sorted by descending date)
2) One HTML page per blog post with: title, metadata, content, and links to previous and next posts.
3) Tag pages which link to each post based on its metadata in the same descending date order

The site will have a navigation bar highlighting the current page you are viewing.
