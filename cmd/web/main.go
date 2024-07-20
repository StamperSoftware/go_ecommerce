package main

import (
	"ecommerce/internal/driver"
	"ecommerce/internal/models"
	"flag"
	"fmt"
	"github.com/joho/godotenv"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"
)

const version = "1.0.0"
const cssVersion = "1"

type config struct {
	port int
	env  string
	api  string
	db   struct {
		dsn string
	}
	stripe struct {
		secret string
		key    string
	}
}

type application struct {
	config        config
	infoLog       *log.Logger
	errorLog      *log.Logger
	templateCache map[string]*template.Template
	version       string
	DB            models.DBModel
}

func (app *application) serve() error {
	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", app.config.port),
		Handler:           app.routes(),
		IdleTimeout:       30 * time.Second,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      5 * time.Second,
	}

	app.infoLog.Println(fmt.Sprintf("Starting http server in %s on port %d", app.config.env, app.config.port))

	return srv.ListenAndServe()
}

func main() {
	var cfg config
	flag.IntVar(&cfg.port, "port", 4000, "Sever port to listen on")
	flag.StringVar(&cfg.env, "env", "development", "Application Environment {development|production}")
	flag.StringVar(&cfg.api, "api", "http://localhost:4001/api", "URL for api")
	flag.Parse()
	godotenv.Load()
	cfg.stripe.key = os.Getenv("STRIPE_KEY")
	cfg.stripe.secret = os.Getenv("STRIPE_SECRET")
	cfg.db.dsn = os.Getenv("DSN")

	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stdout, "Error\t", log.Ldate|log.Ltime|log.Lshortfile)

	conn, err := driver.OpenDB(cfg.db.dsn)

	if err != nil {
		errorLog.Fatal(err)
	}

	defer conn.Close()

	templateCache := make(map[string]*template.Template)

	app := &application{
		config:        cfg,
		infoLog:       infoLog,
		errorLog:      errorLog,
		templateCache: templateCache,
		version:       version,
		DB:            models.DBModel{DB: conn},
	}

	err = app.serve()

	if err != nil {
		app.errorLog.Println(err)
		log.Fatal(err)
	}
}
