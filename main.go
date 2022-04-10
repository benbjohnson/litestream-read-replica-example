package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	// Parse command line flags.
	dsn := flag.String("dsn", "", "datasource name")
	addr := flag.String("addr", ":8080", "bind address")
	flag.Parse()
	if *dsn == "" {
		return fmt.Errorf("flag required: -dsn DSN")
	}

	// Ensure app name, primary region & current region are both set via env vars.
	if os.Getenv("FLY_APP_NAME") == "" {
		return fmt.Errorf("FLY_APP_NAME must be set")
	} else if os.Getenv("FLY_REGION") == "" {
		return fmt.Errorf("FLY_REGION must be set")
	} else if os.Getenv("FLY_PRIMARY_REGION") == "" {
		return fmt.Errorf("FLY_PRIMARY_REGION must be set")
	}

	// Set "readonly" mode if this is a replica.
	if !isPrimary() {
		*dsn += "?mode=ro"
	}

	// Open local SQLite database connection.
	db, err := sql.Open("sqlite3", *dsn)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	// Build schema if this is the primary node.
	if isPrimary() {
		if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS records (id INTEGER PRIMARY KEY, value TEXT NOT NULL)`); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	}

	// Start HTP server.
	log.Printf("listening on %s", *addr)

	return http.ListenAndServe(*addr, &Handler{db: db})
}

// Handler represents an HTTP handler.
type Handler struct {
	db *sql.DB
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// We only support the root path.
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}

	// Our endpoint only supports a GET to read data and POST to write data.
	switch r.Method {
	case "GET":
		h.serveGet(w, r)
	case "POST":
		h.servePost(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// serveGet returns a list of all records that have been created.
func (h *Handler) serveGet(w http.ResponseWriter, r *http.Request) {
	// Fetch all records from the database.
	rows, err := h.db.QueryContext(r.Context(), `SELECT id, value FROM records ORDER BY id`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Read each row and print it to the response body.
	var n int
	for ; rows.Next(); n++ {
		var id int
		var value string
		if err := rows.Scan(&id, &value); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		fmt.Fprintf(w, "Record #%d: %q\n", id, value)
	}
	if err := rows.Close(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return a message if no records are found.
	if n == 0 {
		fmt.Fprintf(w, "No records found. Please POST to add one.\n")
		return
	}
}

// servePost creates a new record using the contents of the request body.
func (h *Handler) servePost(w http.ResponseWriter, r *http.Request) {
	// If this is not the primary, redirect to the primary.
	if !isPrimary() {
		log.Printf("redirecting to primary")
		w.Header().Set("fly-replay", "region="+os.Getenv("FLY_PRIMARY_REGION"))
		return
	}

	// Read body from HTTP request.
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if len(body) == 0 {
		http.Error(w, "request body required", http.StatusBadRequest)
		return
	}

	// Insert record  into the database.
	result, err := h.db.ExecContext(r.Context(), `INSERT INTO records (value) VALUES (?)`, string(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Report the record ID back to the user.
	id, err := result.LastInsertId()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "Record #%d successfully inserted.\n", id)
}

// isPrimary returns true if the current region is the primary region.
func isPrimary() bool {
	return os.Getenv("FLY_PRIMARY_REGION") == os.Getenv("FLY_REGION")
}
