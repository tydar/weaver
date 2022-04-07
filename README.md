# weaver

Minimal custom static-site generator for my site [https://blog.drnll.com](https://blog.drnll.com). I made it because many SSG added a lot of overhead for making rare changes to my site (other than adding posts).

Put markdown-formatted posts in a directory called `posts` at the root level of this project. Posts should look like this:

```markdown
---
title: "This is a post"
date: 2021-01-11
tags:
    - posting
    - ssg
layout: layouts/post.njk
---

Layout is not currently used; it is a leftover of my previous SSG. The above section, bounded by `---`, is a YAML-formatted frontmatter. Currently all fields are required.
```

The site uses the default [Pico.css](https://picocss.com) styles with small changes. Any CSS changes are placed in `static/custom.css` and are copied into the output folder when the site is generated.
