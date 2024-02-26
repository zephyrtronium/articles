/*
Copyright (c) 2021 Branden J Brown

This software is provided 'as-is', without any express or implied warranty.
In no event will the authors be held liable for any damages arising from the
use of this software.

Permission is granted to anyone to use this software for any purpose,
including commercial applications, and to alter it and redistribute it freely,
subject to the following restrictions:

	1. The origin of this software must not be misrepresented; you must not
	claim that you wrote the original software. If you use this software in a
	product, an acknowledgment in the product documentation would be
	appreciated but is not required.

	2. Altered source versions must be plainly marked as such, and must not be
	misrepresented as being the original software.

	3. This notice may not be removed or altered from any source distribution.
*/

package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/feeds"
	"golang.org/x/tools/present"
)

type manifest struct {
	Title       string    `json:"title"`
	Author      string    `json:"author"`
	Email       string    `json:"email"`
	Href        string    `json:"href"`
	Description string    `json:"description"`
	Sections    []section `json:"sections"`
}

type section struct {
	Section  string   `json:"section"`
	Articles []string `json:"articles"`
}

type metadata struct {
	Section  string
	Articles []article
}

type article struct {
	URL     string
	Title   string
	Summary string
	Time    time.Time
}

//go:embed templates/actions.html
//go:embed templates/head.html
//go:embed templates/index.html
//go:embed templates/article.html
//go:embed templates/footer.html
var static embed.FS

func main() {
	var (
		outdir string
		mf     string
	)
	flag.StringVar(&outdir, "out", "", "output directory")
	flag.StringVar(&mf, "mf", "", "manifest json")
	flag.Parse()
	if outdir == "" {
		log.Fatalln("need -out")
	}
	if mf == "" {
		log.Fatalln("need -mf")
	}
	outdir, err := filepath.Abs(outdir)
	if err != nil {
		log.Fatalln("couldn't abs outdir:", err)
	}

	s, err := os.ReadFile(mf)
	if err != nil {
		log.Fatalln("error reading manifest:", err)
	}
	var manifest manifest
	if err := json.Unmarshal(s, &manifest); err != nil {
		log.Fatalln("error unmarshaling manifest:", err)
	}

	tmpl := present.Template()
	if tmpl, err = tmpl.ParseFS(static, "templates/*.html"); err != nil {
		log.Fatalln("error parsing templates:", err)
	}

	// render articles
	if err := os.MkdirAll(filepath.Join(outdir, "articles"), 0644); err != nil {
		log.Fatalln("couldn't make articles output dir:", err)
	}
	var meta []metadata
	feed := feeds.Feed{
		Title:       manifest.Title,
		Link:        &feeds.Link{Href: manifest.Href},
		Description: manifest.Description,
		Author:      &feeds.Author{Name: manifest.Author, Email: manifest.Email},
		Created:     time.Now(),
	}
	for _, sec := range manifest.Sections {
		m := metadata{Section: sec.Section}
		for _, art := range sec.Articles {
			log.Println(sec.Section, art)
			doc, u, when, err := doArticle(outdir, art, tmpl)
			if err != nil {
				log.Println(err)
				continue
			}
			m.Articles = append(m.Articles, article{URL: u, Title: doc.Title, Summary: doc.Summary, Time: doc.Time})
			l, err := url.JoinPath(manifest.Href, u)
			if err != nil {
				log.Println(err)
				continue
			}
			fi := feeds.Item{
				Title:       doc.Title,
				Link:        &feeds.Link{Href: l},
				Author:      &feeds.Author{Name: manifest.Author, Email: manifest.Email},
				Description: doc.Summary,
				Id:          l,
				Created:     when,
			}
			feed.Items = append(feed.Items, &fi)
		}
		meta = append(meta, m)
	}
	// render index
	log.Println("index")
	if err := renderIndex(filepath.Join(outdir, "index.html"), meta, tmpl); err != nil {
		log.Println(err)
	}
	// render feed
	atom, err := feed.ToAtom()
	if err != nil {
		log.Println("generating atom feed:", err, "(continuing)")
	}
	if err := os.WriteFile(filepath.Join(outdir, "weblog.atom"), []byte(atom), 0644); err != nil {
		log.Println("writing atom feed:", err, "(continuing)")
	}
	log.Println("done")
}

func doArticle(out, in string, tmpl *template.Template) (doc *present.Doc, url string, when time.Time, err error) {
	var cwd string
	cwd, err = os.Getwd()
	if err != nil {
		return nil, "", time.Time{}, fmt.Errorf("couldn't get cwd: %w", err)
	}
	if err := os.Chdir(in); err != nil {
		return nil, "", time.Time{}, fmt.Errorf("couldn't cd to %s: %w", in, err)
	}
	defer func() {
		nerr := os.Chdir(cwd)
		if nerr != nil {
			if err == nil {
				err = fmt.Errorf("couldn't cd back to %s: %w", cwd, nerr)
				return
			}
			err = fmt.Errorf("%w; and couldn't cd back to %s: %v", err, cwd, nerr)
		}
	}()
	if doc, err = parse(in); err != nil {
		return nil, "", time.Time{}, err // error already wrapped
	}
	url = filepath.Join("articles", in+".html")
	err = render(filepath.Join(out, url), doc, tmpl) // error already wrapped
	return doc, filepath.ToSlash(url), doc.Time, err
}

func parse(in string) (*present.Doc, error) {
	art, err := os.Open(in + ".article")
	if err != nil {
		return nil, fmt.Errorf("couldn't open %[1]s/%[1]s.article: %[2]w", in, err)
	}
	defer art.Close()
	doc, err := present.Parse(art, in, 0)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse %s: %w", in, err)
	}
	return doc, nil
}

func render(out string, doc *present.Doc, tmpl *template.Template) (err error) {
	var f *os.File
	f, err = os.Create(out)
	if err != nil {
		return fmt.Errorf("couldn't create %s: %w", out, err)
	}
	defer func() {
		nerr := f.Close()
		if nerr != nil {
			if err == nil {
				err = fmt.Errorf("couldn't close %s: %w", out, nerr)
				return
			}
			err = fmt.Errorf("%w; and couldn't close %s: %v", err, out, nerr)
		}
	}()
	err = doc.Render(f, tmpl)
	return err
}

func renderIndex(out string, sections []metadata, tmpl *template.Template) (err error) {
	var f *os.File
	f, err = os.Create(out)
	if err != nil {
		return fmt.Errorf("couldn't create %s: %w", out, err)
	}
	defer func() {
		nerr := f.Close()
		if nerr != nil {
			if err == nil {
				err = fmt.Errorf("couldn't close %s: %w", out, nerr)
				return
			}
			err = fmt.Errorf("%w; and couldn't close %s: %v", err, out, nerr)
		}
	}()
	err = tmpl.ExecuteTemplate(f, "index", sections)
	return err
}
