package xchange

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	dproxy "github.com/koron/go-dproxy"
)

type Xchange interface {
	From() string
	To() string
	Rate() float64
}

func ExecExchange(x Xchange, amount float64) float64 {
	return amount * x.Rate()
}

type simpleXchange struct {
	from string
	to   string
	rate float64
}

func (sx *simpleXchange) From() string {
	return sx.from
}

func (sx *simpleXchange) To() string {
	return sx.to
}

func (sx *simpleXchange) Rate() float64 {
	return sx.rate
}

type chainedXchange struct {
	beforeXchange Xchange
	afterXchange  Xchange
}

var errUnmatchChain = errors.New("before and after must satisfy before.To() == after.From()")

func chain(before, after Xchange) (Xchange, error) {
	if before.To() != after.From() {
		return nil, errUnmatchChain
	}
	return &chainedXchange{
		beforeXchange: before,
		afterXchange:  after,
	}, nil
}

func (cx *chainedXchange) From() string {
	return cx.beforeXchange.From()
}

func (cx *chainedXchange) To() string {
	return cx.afterXchange.To()
}

func (cx *chainedXchange) Rate() float64 {
	return cx.beforeXchange.Rate() * cx.afterXchange.Rate()
}

const queryFormat = `select Rate from yahoo.finance.xchange where pair = "%s%s"`

var errInvalidCode = errors.New("invalid currency code")

func getISOXchange(from, to string) (Xchange, error) {
	if !isValidCode(from) || !isValidCode(to) {
		return nil, errInvalidCode
	}
	params := url.Values{}
	params.Add("q", fmt.Sprintf(queryFormat, from, to))
	params.Add("format", "json")
	params.Add("env", "store://datatables.org/alltableswithkeys")
	qs := strings.Replace(params.Encode(), "+", "%20", -1)
	resp, err := http.Get("https://query.yahooapis.com/v1/public/yql?" + qs)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bodyData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var returnedJSON interface{}
	if err := json.Unmarshal(bodyData, &returnedJSON); err != nil {
		return nil, err
	}
	rateStr, err := dproxy.New(returnedJSON).M("query").M("results").M("rate").M("Rate").String()
	if err != nil {
		return nil, err
	}
	r, err := strconv.ParseFloat(rateStr, 64)
	if err != nil {
		return nil, err
	}
	return &simpleXchange{from, to, r}, nil
}

func mustChain(before Xchange, bErr error, after Xchange, aErr error) (Xchange, error) {
	if bErr != nil {
		return nil, bErr
	}
	if aErr != nil {
		return nil, aErr
	}
	x, err := chain(before, after)
	if err != nil {
		return nil, err
	}
	return x, nil
}

func GetXtendXchange(from, to string) (Xchange, error) {
	if from == to {
		return &simpleXchange{from, to, 1.0}, nil
	} else if from == "r" {
		x, err := GetXtendXchange("JPY", to)
		return mustChain(
			&simpleXchange{"r", "JPY", 60000.0},
			nil,
			x,
			err,
		)
	} else if to == "r" {
		x, err := GetXtendXchange(from, "JPY")
		return mustChain(
			x,
			err,
			&simpleXchange{"JPY", "r", 1.0 / 60000.0},
			nil,
		)
	}
	return getISOXchange(from, to)
}
