package main

import (
	"database/sql"
	"embed"
	"io/fs"
	"log"
	"net/http"

	_ "modernc.org/sqlite"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	db, err := sql.Open("sqlite", "sonic.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	staticFS, _ := fs.Sub(staticFiles, "static")
	http.Handle("/", http.FileServer(http.FS(staticFS)))

	http.HandleFunc("/api/expenses", func(w http.ResponseWriter, r *http.Request) {
		// ...
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
