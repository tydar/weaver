package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/gomarkdown/markdown"
	"gopkg.in/yaml.v2"
)

func main() {
	f, err := os.Open("til-nix-gomod2nix.md")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	contents, err := io.ReadAll(f)
	if err != nil {
		panic(err)
	}

	tmpl, err := template.ParseFiles("base_template.html", "post_template.html")
	if err != nil {
		panic(err)
	}

	postHtml, err := buildPost(string(contents), tmpl)
	if err != nil {
		panic(err)
	}

	fmt.Println(postHtml)
}

// buildPosts iterates over each post in the `/posts/` dir and writes it to the `/html/posts` dir
// has to do these things:
// 1) get out date metadata for ordering
// 2) get out tag metadata for linking and template
// 3) get out title data for formatting
// then pass an object into a template
func buildPosts() {
	return
}

func buildPost(content string, tmpl *template.Template) (string, error) {
	fm, rest, err := extractFrontmatter(content)
	if err != nil {
		return "", fmt.Errorf("extractFrontmatter: %v", err)
	}

	htmlContent := string(markdown.ToHTML([]byte(rest), nil, nil))
	templateData := struct {
		FM      frontMatter
		Content string
		Flash   string
	}{
		FM:      fm,
		Content: htmlContent,
		Flash:   "",
	}

	var sb strings.Builder
	if err := tmpl.ExecuteTemplate(&sb, "base", templateData); err != nil {
		return "", fmt.Errorf("template.Execute: %v", err)
	}

	return sb.String(), nil
}

type frontMatter struct {
	Title  string
	Date   time.Time
	Tags   []string
	Layout string // likely to be unused for now, but will set up to parse because it is currently present
}

// extractFrontMatter parses the front matter block between two sets of `---`
// It returns a struct of type frontMatter, ther rest of the markdown input,
// and optionally an error.
func extractFrontmatter(s string) (frontMatter, string, error) {
	splitOnDividers := strings.SplitN(s, "---\n", 3)
	if len(splitOnDividers) < 3 {
		return frontMatter{}, s, errors.New("could not find both front matter delimiters")
	}

	if splitOnDividers[0] != "" {
		return frontMatter{}, s, errors.New("data before front matter start delimiter")
	}

	var fm frontMatter
	if err := yaml.Unmarshal([]byte(splitOnDividers[1]), &fm); err != nil {
		return frontMatter{}, s, fmt.Errorf("bad front matter format: yaml.Unmarshal: %v", err)
	}

	return fm, splitOnDividers[2], nil
}
