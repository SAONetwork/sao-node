package gql

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type coinPriceResponse struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

type CoinPriceArgs struct {
	Symbol string
}

func (r *resolver) CoinPrice(ctx context.Context, args CoinPriceArgs) (string, error) {
	// Check cache first
	if x, found := r.cache.Get(args.Symbol); found {
		return x.(string), nil
	}

	// Create a new HTTP client with a timeout
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Make a request to the Binance API using the client
	resp, err := client.Get(fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%s", args.Symbol))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var res coinPriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", err
	}

	// Cache the price
	r.cache.Set(args.Symbol, res.Price, 1*time.Minute)

	return res.Price, nil
}

