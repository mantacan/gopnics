package main

import (
	"net"
	"net/http"
	"sync"
	"time"
	"os"
	"log" 
	"fmt" 
	"database/sql"
	"encoding/json"

	_ "github.com/lib/pq"
)

var (
	ipRequests    = sync.Map{}
	totalRequests = 0
	mutex         = sync.Mutex{}
	resetInterval = time.Minute
	ipLimit       = 10
	globalLimit   = 100
)

type UpdateList struct {
	ID          int
	Date        string
	Title       string
	Description string
}

func resetRateLimit() {
	for {
		time.Sleep(resetInterval)

		ipRequests.Range(func(key, value interface{}) bool {
			ipRequests.Delete(key)
			return true
		})

		mutex.Lock()
		totalRequests = 0
		mutex.Unlock()
	}
}

func rateLimit(ip string) bool {
	mutex.Lock()
	defer mutex.Unlock()

	val, _ := ipRequests.Load(ip)
	count, ok := val.(int)
	if !ok {
		count = 0
	}

	if count >= ipLimit {
		return false
	}

	if totalRequests >= globalLimit {
		return false
	}

	ipRequests.Store(ip, count+1)
	totalRequests++

	return true
}

func connectDB() (*sql.DB, error) {
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
	)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}

func getUpdates(db *sql.DB) ([]UpdateList, error) {
	rows, err := db.Query("SELECT id, date, title, description FROM updates")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []UpdateList
	for rows.Next() {
		var item UpdateList
		if err := rows.Scan(&item.ID, &item.Date, &item.Title, &item.Description); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func updateList(w http.ResponseWriter, r *http.Request) {

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}

	if !rateLimit(ip) {
		http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
		return
	}

	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method allowed", http.StatusMethodNotAllowed)
		return
	}

	db, err := connectDB()
	if err != nil {
		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	updates, err := getUpdates(db)
	if err != nil {
		http.Error(w, "Failed to get updates", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updates)
}

func main() {
	log.Println("Starting server on port 8080...")
	go resetRateLimit()
	http.HandleFunc("/update", updateList)
	http.ListenAndServe(":8080", nil)
}
