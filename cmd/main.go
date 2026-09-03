package main

import (
	"embed"
	"log"
	"net/http"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

//go:embed static/*
var staticFiles embed.FS

func main() {
	if err := os.MkdirAll("db", 0o755); err != nil {
		log.Fatal(err)
	}
	db, err := openDB("db/sonic.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	pw, err := ensurePassword(db)
	if err != nil {
		log.Fatal(err)
	}
	if pw != "" {
		log.Printf("first run: the shared password is %s — write it down, it is logged once", pw)
	}

	// Said once at startup as well as at every refusal: the Pi has no RTC, so
	// this is the state to recognise before wondering where the rent went.
	if !clockSane(time.Now) {
		logClockUnset(time.Now)
	}

	log.Printf("listening on :8080 (schema version %d)", schemaVersion)
	log.Fatal(http.ListenAndServe(":8080", newApp(db, time.Now)))
}
