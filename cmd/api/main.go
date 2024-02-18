package main

import (
	"context"
	"encoding/json"
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

	UltimasTransacoesResponse struct {
		Valor       int       `json:"valor"`
		Tipo        string    `json:"tipo"`
		Descricao   string    `json:"descricao"`
		RealizadaEm time.Time `json:"realizada_em"`
	}

	ExtratoResponse struct {
		Saldo struct {
			Total       int    `json:"total"`
			DataExtrato string `json:"data_extrato"`
			Limite      int    `json:"limite"`
		} `json:"saldo"`
		UltimasTransacoes []UltimasTransacoesResponse `json:"ultimas_transacoes"`
	}
)

const (
	SQL_LISTAR_CLIENTES   = "SELECT saldo, limite FROM clientes WHERE id = $1"
	SQL_LISTAR_TRANSACOES = "SELECT valor, tipo, descricao, realizado_em AS realizado_em FROM transacoes  WHERE cliente_id = $1 ORDER BY realizado_em DESC LIMIT 10"
)

func main() {
	port := getEnv("PORT", "3000")
	dsn := getEnv("DSN", "postgres://postgres:postgres@db/rinha?sslmode=disable")

	pc, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		panic(err)
	}
	pc.MaxConns = 100

	pool, err := pgxpool.NewWithConfig(context.Background(), pc)
	if err != nil {
		panic(err)
	}
	defer pool.Close()

	srv := http.Server{
		Addr:    "0.0.0.0:" + port,
		Handler: routes(pool),
	}

	log.Println("Server running on port ", port)
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
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		if cid > 5 {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var trb Transacao
		if err := json.NewDecoder(r.Body).Decode(&trb); err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		if (trb.Tipo != "c" && trb.Tipo != "d") || trb.Valor <= 0 || len(trb.Descricao) == 0 || len(trb.Descricao) > 10 {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		trb.ClienteID = cid
		if trb.Tipo == "d" {
			trb.Valor = -trb.Valor
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var limite, saldo *int
		err = pool.QueryRow(ctx, "CALL criar_tr($1, $2, $3, $4)", trb.ClienteID, trb.Valor, trb.Tipo, trb.Descricao).Scan(&saldo, &limite)
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
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
		p := r.PathValue("id")
		cid, err := strconv.Atoi(p)
		if err != nil {
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}

		if cid > 5 {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var saldo, limite *int
		err = pool.QueryRow(ctx, SQL_LISTAR_CLIENTES, cid).Scan(&saldo, &limite)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if saldo == nil || limite == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		var ultimasTransacoes []UltimasTransacoesResponse
		rows, err := pool.Query(ctx, SQL_LISTAR_TRANSACOES, cid)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var utr UltimasTransacoesResponse
			err = rows.Scan(&utr.Valor, &utr.Tipo, &utr.Descricao, &utr.RealizadaEm)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			ultimasTransacoes = append(ultimasTransacoes, utr)
		}

		var extratoResponse ExtratoResponse
		extratoResponse.Saldo.Total = *saldo
		extratoResponse.Saldo.Limite = *limite
		extratoResponse.Saldo.DataExtrato = time.Now().UTC().String()
		extratoResponse.UltimasTransacoes = ultimasTransacoes

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(extratoResponse)
	}
}

func getEnv(key, value string) string {
	v, ok := os.LookupEnv(key)
	if !ok {
		return value
	}

	return v
}
