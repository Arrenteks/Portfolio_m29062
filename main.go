package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/russross/blackfriday/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Page struct {
	Title      string
	LastChange time.Time
	Content    template.HTML
}

type Pages []Page

var (
	srcDir       = flag.String("src", "./seiten", "Inhalte-Dir.")
	tmpDir       = flag.String("tmp", "./templates", "Template-Dir.")
	statDir      = flag.String("static", "./static/html/", "Static-Dir")
	ps           Pages
	user         = "root"
	userpassword = "rootpassword"
)

func main() {

	client := InitiateMongoClient()
	coll := client.Database("portfolio").Collection("fs.files")
	count, err := coll.EstimatedDocumentCount(context.TODO())
	if err != nil {
		fmt.Println("Error in Counting the number of ")
	}

	for i := 1; int64(i) <= count; i++ {
		DownloadFile("seiten/", "project"+strconv.Itoa(i)+".md", "portfolio", client)
	}

	flag.Parse()
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	log.Println(fs)

	err = loadPages(*srcDir) //preload all pages for faster access time

	if err != nil {
		log.Println("Error in Loading the Pages: %w", err)
	}

	generateStaticPage(ps, "static.index.templ.html", *statDir, "index.html")

	for i := 1; i <= len(ps); i++ {
		generateStaticPage(ps[i-1], "page.templ.html", *statDir, "project"+strconv.Itoa(i)+".html")
	}

	http.HandleFunc("/", makeIndexHandler())
	http.HandleFunc("/page/", makePageHandler())

	log.Print("Listening on Port 9000....")
	err = http.ListenAndServe(":9000", nil)
	if err != nil {
		log.Fatal(err)
	}
}

func InitiateMongoClient() *mongo.Client {
	var err error
	var client *mongo.Client
	uri := "mongodb://" + user + ":" + userpassword + "@localhost:27017"
	opts := options.Client()
	opts.ApplyURI(uri)
	opts.SetMaxPoolSize(5)
	if client, err = mongo.Connect(context.Background(), opts); err != nil {
		fmt.Println(err.Error())
	}
	return client
}

func DownloadFile(directory string, fileName string, databasename string, conn *mongo.Client) {

	db := conn.Database(databasename)
	fsFiles := db.Collection("fs.files")
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	var results bson.M
	err := fsFiles.FindOne(ctx, bson.M{}).Decode(&results)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(results)

	bucket, _ := gridfs.NewBucket(
		db,
	)
	var buf bytes.Buffer
	dStream, err := bucket.DownloadToStreamByName(fileName, &buf)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("File size to download: %v\n", dStream)
	ioutil.WriteFile(directory+fileName, buf.Bytes(), 0600)

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

func generateStaticPage(data interface{}, content string, directory string, name string) error {
	temp, err := template.ParseFiles(
		filepath.Join(*tmpDir, "static.base.templ.html"),
		filepath.Join(*tmpDir, "static.header.templ.html"),
		filepath.Join(*tmpDir, "footer.templ.html"),
		filepath.Join(*tmpDir, content),
	)

	if err != nil {
		return fmt.Errorf("generateStaticPage.ParseFiles: %w", err)
	}

	file, err := os.Create(directory + name)

	if err != nil {
		return fmt.Errorf("generateStaticPage.Create: %w", err)
	}

	err = temp.ExecuteTemplate(file, "base", data)

	if err != nil {
		return fmt.Errorf("generateStaticPage.ExecuteTemplate: %w", err)
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
