package main

import (
	"flag"
	"fmt"
	"os"

	xchange "github.com/zaltoprofen/r-exchange"
)

var (
	from   = flag.String("from", "USD", "")
	to     = flag.String("to", "JPY", "")
	amount = flag.Float64("amount", 1.0, "")
)

func init() {
	flag.Parse()
}

func main() {
	x, err := xchange.GetXtendXchange(*from, *to)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	changed := xchange.ExecExchange(x, *amount)
	fmt.Printf("%f%s = %f%s\n", *amount, *from, changed, *to)
}
