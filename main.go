package main

import (
	"embed"
	"log"
	"net/http"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	db, err := openDB("sonic.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	log.Printf("listening on :8080 (schema version %d)", schemaVersion)
	log.Fatal(http.ListenAndServe(":8080", newApp(db, time.Now)))
}
