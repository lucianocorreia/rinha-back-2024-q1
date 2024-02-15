package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	port := getEnv("PORT", "8080")
	dsn := getEnv("DSN", "postgres://postgres:postgres@localhost/rinha?sslmode=disable")

	pc, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		panic(err)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), pc)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	srv := http.Server{
		Addr:    "0.0.0.0:" + port,
		Handler: routes(pool),
	}

	log.Println("Server running on port", port)
	log.Fatalln(srv.ListenAndServe())
}

func routes(pool *pgxpool.Pool) http.Handler {

	return nil
}

func getEnv(key, value string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return value
	}

	return v
}
