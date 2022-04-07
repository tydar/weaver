package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/gomarkdown/markdown"
	"gopkg.in/yaml.v2"
)

func main() {
	_, devPrs := os.LookupEnv("WEAVER_DEV")

	tmpl, err := template.ParseFiles("templates/base_template.html", "templates/post_template.html")
	if err != nil {
		panic(fmt.Errorf("template.ParseFiles: %v", err))
	}

	indexTemplate, err := template.ParseFiles("templates/base_template.html", "templates/index_template.html")
	if err != nil {
		panic(fmt.Errorf("template.ParseFiles: %v", err))
	}

	archiveTemplate, err := template.ParseFiles("templates/base_template.html", "templates/archive_template.html")
	if err != nil {
		panic(fmt.Errorf("template.ParseFiles: %v", err))
	}

	tagTemplate, err := template.ParseFiles("templates/base_template.html", "templates/tag_template.html")
	if err != nil {
		panic(fmt.Errorf("template.ParseFiles: %v", err))
	}

	// create directory structure
	path := "output/tag"
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		panic(err)
	}

	index, tags, err := buildPosts(tmpl)
	if err != nil {
		log.Fatalf("buildPosts: %v", err)
	}

	if err := buildIndexPage(index, indexTemplate); err != nil {
		log.Fatalf("buildIndexPage: %v", err)
	}

	if err := buildArchivePage(index, archiveTemplate); err != nil {
		log.Fatalf("buildArchivePage: %v", err)
	}

	if err := buildTagsPages(index, tags, tagTemplate); err != nil {
		log.Fatalf("buildTagsPages: %v", err)
	}

	if err := linkCSSToOutput(); err != nil {
		log.Fatalf("linkCSSToOutput: %v", err)
	}

	if devPrs {
		fs := http.FileServer(http.Dir("./output"))
		http.Handle("/", fs)
		if err := http.ListenAndServe(":3000", nil); err != nil {
			panic(err)
		}
	}
}

// types

// data used to sort for indexes, tags
// Path should be relative path from the location the index is stored. will start by assuming a flat output dir
type sortedPost struct {
	Path  string
	Title string
	Tags  []string
	Date  time.Time
}

// front matter struct
type frontMatter struct {
	Title  string
	Date   time.Time
	Tags   []string
	Layout string // likely to be unused for now, but will set up to parse because it is currently present
}

// functions

// buildPosts iterates over each post in the `/posts/` dir and writes it to the `/html/posts` dir
// has to do these things:
//
// 1) get out date metadata for ordering
// 2) get out tag metadata for linking and template
// 3) get out title data for formatting
//
// then pass an object into a template
//
// also need to maintain and return a sorted list of all posts for the index
// and sorted lists per tag for the tag pages.
// main sorted list can be []sortObj
// tag lists can be represented as map[tag][]int (each int being a key for the main sorted list)
// once the main, sorted index exists can traverse it one more time to generate each tag list
func buildPosts(t *template.Template) ([]sortedPost, map[string][]int, error) {
	theFs := os.DirFS("./posts/")
	markdownFiles, err := fs.Glob(theFs, "*.md")
	if err != nil {
		return []sortedPost{}, map[string][]int{}, fmt.Errorf("fs.Glob: %v", err)
	}

	index := make([]sortedPost, len(markdownFiles))
	for n := range markdownFiles {
		filename := markdownFiles[n]
		filenameWithoutExt := filename[:len(filename)-len(filepath.Ext(filename))]
		path := filepath.Join("./output/", filenameWithoutExt+".html")

		f, err := os.Open(filepath.Join("./posts/", markdownFiles[n]))
		if err != nil {
			return []sortedPost{}, map[string][]int{}, fmt.Errorf("os.Open: %v", err)
		}

		content, err := io.ReadAll(f)
		if err != nil {
			return []sortedPost{}, map[string][]int{}, fmt.Errorf("io.ReadAll: %v", err)
		}

		post, fm, err := buildPost(string(content), t)
		if err != nil {
			return []sortedPost{}, map[string][]int{}, fmt.Errorf("buildPost: %v", err)
		}

		fOut, err := os.Create(path)
		if err != nil {
			return []sortedPost{}, map[string][]int{}, fmt.Errorf("os.Create: %v", err)
		}

		_, err = fmt.Fprint(fOut, post)
		if err != nil {
			return []sortedPost{}, map[string][]int{}, fmt.Errorf("fmt.Fprint: %v", err)
		}

		f.Close()
		fOut.Close()

		index[n] = sortedPost{
			Path:  filenameWithoutExt + ".html",
			Date:  fm.Date,
			Tags:  fm.Tags,
			Title: fm.Title,
		}
	}

	sortIndexByDate(index)
	tags := generateTagsMap(index)
	return index, tags, nil
}

func buildPost(content string, tmpl *template.Template) (string, frontMatter, error) {
	fm, rest, err := extractFrontmatter(content)
	if err != nil {
		return "", frontMatter{}, fmt.Errorf("extractFrontmatter: %v", err)
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
		return "", frontMatter{}, fmt.Errorf("template.Execute: %v", err)
	}

	return sb.String(), fm, nil
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

func sortIndexByDate(index []sortedPost) {
	for i := 1; i < len(index); i++ {
		for j := i; j > 0 && index[j-1].Date.Before(index[j].Date); j-- {
			temp := index[j]
			index[j] = index[j-1]
			index[j-1] = temp
		}
	}
}

func buildIndexPage(index []sortedPost, tmpl *template.Template) error {
	sliceLen := 5
	if len(index) < 5 {
		sliceLen = len(index)
	}
	mostRecent := index[:sliceLen]
	path := filepath.Join("./output/", "index.html")

	fOut, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("os.Create: %v", err)
	}
	defer fOut.Close()

	templateData := struct {
		Posts []sortedPost
		Flash string
	}{
		Posts: mostRecent,
		Flash: "",
	}

	if err := tmpl.ExecuteTemplate(fOut, "base", templateData); err != nil {
		return fmt.Errorf("tmpl.ExecuteTemplate: %v", err)
	}

	return nil
}

func buildArchivePage(index []sortedPost, tmpl *template.Template) error {
	path := filepath.Join("./output/", "archive.html")

	fOut, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("os.Create: %v", err)
	}
	defer fOut.Close()

	templateData := struct {
		Posts []sortedPost
		Flash string
	}{
		Posts: index,
		Flash: "",
	}

	if err := tmpl.ExecuteTemplate(fOut, "base", templateData); err != nil {
		return fmt.Errorf("tmpl.ExecuteTemplate: %v", err)
	}

	return nil
}

// generateTagsMap takes a []sortedPost and returns a map[string][]int
// where each key is a tag and each value is a slice of indicies in the index slice
// for posts which have that tag
func generateTagsMap(index []sortedPost) map[string][]int {
	tags := make(map[string][]int)

	for i := range index {
		post := index[i]
		for j := range post.Tags {
			tag := post.Tags[j]
			v, prs := tags[tag]
			if !prs {
				tags[tag] = []int{i}
			} else {
				tags[tag] = append(v, i)
			}
		}
	}
	return tags
}

func buildTagsPages(index []sortedPost, tags map[string][]int, tmpl *template.Template) error {
	for k, v := range tags {
		path := filepath.Join("./output/tag/", k+".html")
		posts := make([]sortedPost, len(v))
		for i := range v {
			posts[i] = index[v[i]]
		}

		templateData := struct {
			Posts []sortedPost
			Flash string
			Tag   string
		}{
			Posts: posts,
			Flash: "",
			Tag:   k,
		}

		fOut, err := os.Create(path)
		if err != nil {
			return fmt.Errorf("os.Create: %v", err)
		}
		defer fOut.Close()

		if err := tmpl.ExecuteTemplate(fOut, "base", templateData); err != nil {
			return fmt.Errorf("tmpl.ExecuteTemplate: %v", err)
		}
	}
	return nil
}

func linkCSSToOutput() error {
	theFs := os.DirFS("./static/")
	theCssFiles, err := fs.Glob(theFs, "*.css")

	if err != nil {
		return fmt.Errorf("fs.Glob: %v", err)
	}

	for i := range theCssFiles {
		thePath := filepath.Join("./static/", theCssFiles[i])
		theNewPath := filepath.Join("./output/", theCssFiles[i])
		if err := os.Link(thePath, theNewPath); err != nil {
			return fmt.Errorf("os.Link: %v", err)
		}
	}
	return nil
}
