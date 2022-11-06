package stock

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/jmirfield/stockbot/internal/util"
)

type stock struct {
	Ticker string  `json:"ticker"`
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int     `json:"volume"`
}

func requestStockHistory(ticker string) []stock {
	base := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=%v&datatype=csv&outputsize=full&apikey=apikey", ticker)

	resp, err := http.Get(base)

	if err != nil {
		log.Println(err)
	}

	reader := csv.NewReader(resp.Body)
	reader.LazyQuotes = true
	ch := util.ProcessCSV(reader)

	var dataSlice []stock
	for data := range ch {
		if len(data) < 6 {
			break
		}
		date := data[0]
		open, err := strconv.ParseFloat(data[1], 32)
		if err != nil {
			log.Println(err)
			break
		}

		high, err := strconv.ParseFloat(data[2], 32)
		if err != nil {
			log.Println(err)
			break
		}

		low, err := strconv.ParseFloat(data[3], 32)
		if err != nil {
			log.Println(err)
			break
		}

		close, err := strconv.ParseFloat(data[4], 32)
		if err != nil {
			log.Println(err)
			break
		}

		volume, err := strconv.Atoi(data[5])
		if err != nil {
			log.Println(err)
			break
		}
		stock := stock{ticker, date, open, high, low, close, volume}
		dataSlice = append(dataSlice, stock)
	}
	return dataSlice
}
