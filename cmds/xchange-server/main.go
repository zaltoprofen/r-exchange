package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"strconv"

	xchange "github.com/zaltoprofen/r-exchange"
)

var (
	addr = flag.String("addr", ":8080", "binding address")
)

func init() {
	flag.Parse()
}

func main() {
	http.HandleFunc("/exchange.json", exchange)
	http.ListenAndServe(*addr, nil)
}

var errUnsatisfied = errors.New("required parameters are not filled")

func require(ss ...string) error {
	for _, s := range ss {
		if s == "" {
			return errUnsatisfied
		}
	}
	return nil
}

func exchange(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(405)
		w.Write([]byte("405 Method Not Allowed"))
		return
	}
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(400)
		w.Write([]byte("400 Bad Request"))
		return
	}
	params := r.Form
	from := params.Get("from")
	to := params.Get("to")
	amountStr := params.Get("amount")
	if require(from, to, amountStr) != nil {
		w.WriteHeader(400)
		fmt.Fprintln(w, "400 Bad Request: from, to and amount are required parameter")
		return
	}
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintln(w, "400 Bad Request: amount must be float")
		return
	}
	x, err := xchange.GetXtendXchange(from, to)
	if err != nil {
		w.WriteHeader(400)
		fmt.Fprintln(w, "500 Internal Server Error:", err)
		return
	}
	exchanged := xchange.ExecExchange(x, amount)
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.Encode(struct {
		From   string  `json:"from"`
		To     string  `json:"to"`
		Rate   float64 `json:"Rate"`
		Amount float64 `json:"amount"`
	}{
		From:   from,
		To:     to,
		Rate:   x.Rate(),
		Amount: exchanged,
	})
}
