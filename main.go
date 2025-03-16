package main

import (
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	// urlExample := "postgres://user:pass@localhost:5432/db"
	pgPool, err := pgxpool.New(context.Background(), os.Getenv("PG"))
	if err != nil {
		log.Fatal(err)
	}

	srvc := service{db: pgPool}

	r := mux.NewRouter()
	r.HandleFunc("/self-check", srvc.selfCheck).Methods(http.MethodGet)
	r.HandleFunc("/items", srvc.addItem).Methods("POST")
	r.HandleFunc("/items/{id}", srvc.getItem).Methods("GET")
	r.HandleFunc("/items", srvc.getAllItems).Methods("GET")

	s := &http.Server{
		Addr:           ":8080",
		Handler:        r,
		ReadTimeout:    1 * time.Second,
		WriteTimeout:   1 * time.Second,
		MaxHeaderBytes: 1 << 10,
	}
	log.Fatal(s.ListenAndServe())
}

type Item struct {
	PK   int    `json:"pk"`
	Data string `json:"data"`
}

type service struct {
	db *pgxpool.Pool
}

func (s *service) selfCheck(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	err := s.db.Ping(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(err.Error()))
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func (s *service) addItem(w http.ResponseWriter, r *http.Request) {
	var item Item
	if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	row := s.db.QueryRow(context.Background(), "INSERT INTO my_table (data) VALUES ($1) RETURNING pk", item.Data)
	if err := row.Scan(&item.PK); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(item); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *service) getItem(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var item Item
	row := s.db.QueryRow(context.Background(), "SELECT pk, data FROM my_table WHERE pk = $1", id)
	if err := row.Scan(&item.PK, &item.Data); err != nil {
		http.Error(w, "Item not found", http.StatusNotFound)
		return
	}
	if err := json.NewEncoder(w).Encode(item); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *service) getAllItems(w http.ResponseWriter, r *http.Request) {
	rows, err := s.db.Query(context.Background(), "SELECT pk, data FROM my_table")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	items := []Item{}
	for rows.Next() {
		var item Item
		if err := rows.Scan(&item.PK, &item.Data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		items = append(items, item)
	}
	if err := json.NewEncoder(w).Encode(items); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
