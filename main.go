package main

import (
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/russross/blackfriday/v2"
)

type Page struct {
	Title      string
	LastChange time.Time
	Content    template.HTML
}

type Pages []Page

var (
	srcDir = flag.String("src", "./seiten", "Inhalte-Dir.")
	tmpDir = flag.String("tmp", "./templates", "Template-Dir.")
	ps     Pages
)

func main() {
	flag.Parse()
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	log.Println(fs)

	err := loadPages(*srcDir) //preload all pages for faster access time

	if err != nil {
		log.Println("Error in Loading the Pages: %w", err)
	}

	http.HandleFunc("/", makeIndexHandler())
	http.HandleFunc("/page/", makePageHandler())

	log.Print("Listening on Port 9000....")
	err = http.ListenAndServe(":9000", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func makeIndexHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ps := getPages()

		err := renderPage(w, ps, "index.templ.html")
		if err != nil {
			log.Println(err)
		}
	}
}

func makePageHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f := r.URL.Path[len("/page/"):]
		fpath := filepath.Join(*srcDir, f)
		log.Print(fpath)
		p, err := getPage(fpath)

		if err != nil {
			log.Println(err)
		}

		err = renderPage(w, p, "page.templ.html")
		if err != nil {
			log.Println(err)
		}
	}
}

func renderPage(w io.Writer, data interface{}, content string) error {
	temp, err := template.ParseFiles(
		filepath.Join(*tmpDir, "base.templ.html"),
		filepath.Join(*tmpDir, "header.templ.html"),
		filepath.Join(*tmpDir, "footer.templ.html"),
		filepath.Join(*tmpDir, content),
	)

	if err != nil {
		return fmt.Errorf("renderPage.Parsefiles: %w", err)
	}

	err = temp.ExecuteTemplate(w, "base", data)
	if err != nil {
		return fmt.Errorf("renderPage.ExecuteTemplate: %w", err)
	}
	return nil
}

func getPage(name string) (Page, error) {
	var page Page
	fi, err := os.Stat(name)

	if err != nil {
		return page, fmt.Errorf("getPage: %w", err)
	}

	for i := 0; i < len(ps); i++ {
		if ps[i].Title == fi.Name() {
			page = ps[i]
		}
	}

	log.Print(page.Title)

	return page, nil
}

func loadPage(fpath string) (Page, error) {
	var p Page
	fi, err := os.Stat(fpath)

	if err != nil {
		return p, fmt.Errorf("loadPage: %w", err)
	}

	p.Title = fi.Name()
	p.LastChange = fi.ModTime()
	b, err := ioutil.ReadFile(fpath)
	if err != nil {
		return p, fmt.Errorf("loadPage.ReadFile: %w", err)
	}

	p.Content = template.HTML(blackfriday.Run(b))

	return p, nil
}

func getPages() Pages {
	return ps
}

func loadPages(src string) error {
	fs, err := ioutil.ReadDir(src)
	if err != nil {
		return fmt.Errorf("loadPages.ReadDir: %w", err)
	}

	for _, f := range fs {
		if f.IsDir() {
			continue
		}
		fpath := filepath.Join(src, f.Name())
		p, err := loadPage(fpath)
		if err != nil {
			return fmt.Errorf("loadPages.loadPage: %w", err)
		}

		ps = append(ps, p)
	}

	return nil
}
