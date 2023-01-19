package main

// Autor: Daniel Adam, Matrikel: m29062
// letzte Änderung: 2023-01-13
// Package Main beschreibt wie die Webapplikation aufgebaut werden soll. Es liefert den Verweis für die zugehörigen go.sum und go.mod
// Dateien zu dem Programm.
import (
	"archive/zip"
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
	"path"
	"path/filepath"
	"strconv"
	"time"

	"github.com/russross/blackfriday/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/gridfs"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Beschreibt wie die HTML-Seiten gespeichert werden sollen.
// Title - Titel der Page
// LastChange - Zeit der letzten Änderung
// Content - Inhalt der Page, gespeichert als zusammengesetztes HTML Template
type Page struct {
	Title      string
	LastChange time.Time
	Content    template.HTML
}

// Ein Slice von Page ergibt Pages. Sorgt für übersichtlicherer Implementation eines Page Arrays im Code
type Pages []Page

// Variablen, welche im gesamten Code Anwendung finden
var (
	srcDir       = flag.String("src", "./seiten", "Inhalte-Dir.")        // Das Quellverzeichnis für die Markdown-Files
	tmpDir       = flag.String("tmp", "./templates", "Template-Dir.")    // Das Verzeichnis für die einzelnen Templates
	statDir      = flag.String("static", "./static/html/", "Static-Dir") // Das Verzeichnis in dem die HTML-Seiten nach Erstellung gespeichert werden sollen
	ps           Pages                                                   //Speicherstelle für unsere generierten Pages
	user         = "root"                                                //Username für die Datenbank
	userpassword = "rootpassword"                                        //Passwort für die Datenbank (Sicherheitsrisiko)
	databasename = "gomdb"                                               //Name der Datenbank - MUSS mit dem Namen des Dockercontainers übereinstimmen
	port         = "27017"                                               //auf welchem Port läuft die Datenbank
)

// Hauptfunktion des Programmes. Hier kommt alles zusammen und wird mit den individuellen Spezifikationen ausgeführt.
// Zuerst wird eine Verbindung zur Datenbank hergestellt und die dort gespeicherten Markdown Dateien heruntergeladen.
// Diese werden anschließend mit ihren jeweiligen Templates zu HTML-Templates zusammengesetzt und abgespeichert.
// Danach erfolgt die Umwandlung und Speicherung dieser in statische HTML - Pages im zugehörgigen Verzeichnis
// Nachdem das Setup erfolgreich war werden die http-Funktionen mit ihren zugehörigen Handlern verbunden
// Zum Schluss wird das ganze dem Port 9000 zugeordnet und eine entsprechende Statusmeldung geloggt.
// Die Website ist nun erreichbar.
func main() {

	client, ctx := InitiateMongoClient()

	coll := client.Database("portfolio").Collection("fs.files")
	coll.DeleteMany(ctx, bson.M{}) //Lösche alle alten Daten

	coll = (*mongo.Collection)(client.Database("portfolio").Collection("fs.chunks"))
	coll.DeleteMany(ctx, bson.M{}) //Lösche alle alten Daten

	file, err := os.Stat("raw/seiten.zip")

	if err != nil {
		fmt.Println(err)
	}

	filename := path.Base(file.Name())
	UploadFile(file.Name(), filename, "raw/", "portfolio", client)

	file, err = os.Stat("raw/templates.zip")

	if err != nil {
		fmt.Println(err)
	}

	filename = path.Base(file.Name())
	UploadFile(file.Name(), filename, "raw/", "portfolio", client)

	//Aufsetzen der Datenbank abgeschlossen

	DownloadFile("raw/", "seiten.zip", "portfolio", client, "fs.files")
	DownloadFile("raw/", "templates.zip", "portfolio", client, "fs.files")

	unzipFile("raw/seiten.zip", "")
	unzipFile("raw/templates.zip", "")

	flag.Parse()
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	log.Println(fs)

	err = loadPages(*srcDir) //lade alle Pages für schnelleren Zugriff vor

	if err != nil {
		log.Println("Error in Loading the Pages: %w", err)
	}

	generateStaticPage(ps, "static.index.templ.html", *statDir, "index.html") //generiere statsiche HTML Pages

	for i := 1; i <= len(ps); i++ {
		generateStaticPage(ps[i-1], "page.templ.html", *statDir, "project"+strconv.Itoa(i)+".html")
	}

	http.HandleFunc("/", makeIndexHandler())     // hole den Index für alle Pages, beim öffnen der Website
	http.HandleFunc("/page/", makePageHandler()) // hole die individuelle Page beim zugreifen auf die Seite

	log.Print("Listening on Port 9000....")
	err = http.ListenAndServe(":9000", nil) //Warte am Port 9000 auf Zugriffe
	if err != nil {
		log.Fatal(err)
	}
}

func UploadFile(file, filename string, directory string, databasename string, client *mongo.Client) {

	data, err := ioutil.ReadFile(directory + file)
	if err != nil {
		log.Fatal(err)
	}
	conn := client
	bucket, err := gridfs.NewBucket(
		conn.Database(databasename),
	)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	uploadStream, err := bucket.OpenUploadStream(
		filename,
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer uploadStream.Close()

	fileSize, err := uploadStream.Write(data)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	log.Printf("Write file to DB was successful. File size: %d M\n", fileSize)
}

func unzipFile(filename string, dst string) {

	archive, err := zip.OpenReader(filename)

	if err != nil {
		log.Println("Fehler im Entpacken der Datei: %w", filename, err)
	}

	defer archive.Close()

	for _, f := range archive.File {
		filePath := filepath.Join(dst, f.Name)
		log.Println("entpacke Datei", filePath)

		if f.FileInfo().IsDir() {
			fmt.Println("erstelle Directory...")
			os.MkdirAll(filePath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			panic(err)
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			panic(err)
		}

		fileInArchive, err := f.Open()
		if err != nil {
			panic(err)
		}

		if _, err := io.Copy(dstFile, fileInArchive); err != nil {
			panic(err)
		}

		dstFile.Close()
		fileInArchive.Close()
	}

}

// Initialisierung eines MongoDB Clienten mit manuell spezifizierten Daten
// Gibt den daraus resultierenden Pointer auf den Clienten zurück
// returns - *mongo.Client - Pointer auf einen MongoDb Client
func InitiateMongoClient() (*mongo.Client, context.Context) {
	var err error
	var client *mongo.Client
	uri := "mongodb://" + user + ":" + userpassword + "@" + databasename + ":" + port
	opts := options.Client()
	opts.SetDirect(true)
	opts.ApplyURI(uri)
	opts.SetMaxPoolSize(5)
	ctx := context.Background()
	if client, err = mongo.Connect(ctx, opts); err != nil {
		fmt.Println(err.Error())
	}
	return client, ctx
}

// lädt eine manuell spezifizierte Datei aus einer MongoDB - Datenbank in der Collection "fs.files" herunter und verpackt sie in ein Verzeichnis
// directory - wo soll die File gespeichert werden (wichtig: / muss mit an den letzten Teil angehängt werden)
// fileName - Name der runterzuladenden Datei
// databasename - Name der Datenbank
// conn - Pointer auf einen MongoDB - Clienten
// coll - Name der Collection aus der die Datei geladen werden soll
func DownloadFile(directory string, fileName string, databasename string, conn *mongo.Client, coll string) {

	db := conn.Database(databasename)
	fsFiles := db.Collection(coll)
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

// http.Handlerfunktion, die auf Basis der renderPage() und getPages() Funktion eine Indexseite generiert und diese als Response in
// den Responsewriter w schreibt
// returns http.HandlerFunc - Http Handlerfunktion, die das oben beschriebene ausführt
func makeIndexHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ps := getPages()

		err := renderPage(w, ps, "index.templ.html")

		if err != nil {
			log.Println(err)
		}

	}
}

// http.Handlerfunktion, die auf Basis der renderPage() Funktion eine spezifische Seite anhand der aus der Request r resultierenden URL generiert
// Die entstehende Seite wird mit Hilfe des Responsewriter w zurückgegeben
// returns - http.HandlerFunc - Http Handlerfunktion, die sich um die oben beschriebenen Schritte kümmert
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

// Baut basierend auf den Templates "base.temp.html", "header.temp.html", "footer.temp.html" und einer Contentseite eine HTML-Seite auf
// Diese Seite wird anschließend an einen Writer w übergeben
// w - Writer für Dateien
// data - Dateninterface
// content - Inhalt
// returns - error - mögliche Fehlerinformationen
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

// Funktion zur Generierung einer statischen HTML Page. Wie bei renderPage() werden Templates als Basis verwendet.
// Hier finden aber die Templates "static.base.templ.html" und "static.header.templ.html" Anwendung. Dies geschieht aufgrund der Verzeichnisstruktur, welche in den normalen Templates anders strukturiert ist und daher in den statischen Pages zu einem nicht Finden von Images und Stylesheets führt
// Die feritgen HTML Seiten werden hier jedoch nicht an einen Writer zurück gegeben, sondern als statische HTML Seiten in ein manuell spezifiziertes Verzeichnis eingetragen
// data - Interface, welches die Daten für die Templates trägt
// content - Content der Page
// directory - wo soll die statische HTML Seite erstellt werden
// name - Name der Datei
// returns - error - mögliche Fehlerinformationen
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

// gibt eine einzelne Seite aus dem Slice "pages" anhand des Namens zurück - lineare Suche
// name - Name der Datei
// returns - Page - die gefundene Page
// returns - error - mögliche Fehlerinformationen
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

// Setzt den Inhalt einer Page p anhand der Datei in einem übergebenen Filepath. Nutzt für den Content die blackfriday Markdown Engine
// fpath - Verzeichnisstruktur der gesuchten File
// returns - p - Page mit Daten
// returns - error - Fehler, sollte es zu einem Fehler gekommen sein
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

// gibt die Pages ps zurück
// returns - Pages - Pages
func getPages() Pages {
	return ps
}

// lädt alle Pages in einem Verzeichnis und übergibt sie der Methode loadPages()
// src - Quellverzeichnis
// returns - error - mögliche Fehlerinformationen
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
