package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type (
	Transacao struct {
		ClienteID int    `json:"cliente_id"`
		Valor     int    `json:"valor"`
		Tipo      string `json:"tipo"`
		Descricao string `json:"descricao"`
	}

	TransacaoResponse struct {
		Limite int `json:"limite"`
		Saldo  int `json:"saldo"`
	}
)

func main() {
	port := getEnv("PORT", "3000")
	dsn := getEnv("DSN", "postgres://postgres:postgres@db/rinha?sslmode=disable")

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
	mux := http.NewServeMux()

	mux.HandleFunc("POST /clientes/{id}/transacoes", handleTransacoes(pool))
	mux.HandleFunc("GET /clientes/{id}/extrato", handleExtrato(pool))

	return mux
}

func handleTransacoes(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		p := r.PathValue("id")
		cid, err := strconv.Atoi(p)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if cid > 5 {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var trb Transacao
		if err := json.NewDecoder(r.Body).Decode(&trb); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if (trb.Tipo != "c" && trb.Tipo != "d") || trb.Valor <= 0 || len(trb.Descricao) == 0 || len(trb.Descricao) > 10 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		trb.ClienteID = cid
		if trb.Tipo == "d" {
			trb.Valor = -trb.Valor
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var limite, saldo *int
		err = pool.QueryRow(ctx, "CALL criar_tr($1, $2, $3, $4)", trb.ClienteID, trb.Valor, trb.Tipo, trb.Descricao).Scan(&limite, &saldo)
		if err != nil {
			fmt.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if limite == nil || saldo == nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		jsonResponse := TransacaoResponse{
			Limite: *limite,
			Saldo:  *saldo,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(jsonResponse)
	}
}

func handleExtrato(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotImplemented)
	}
}

func getEnv(key, value string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return value
	}

	return v
}
