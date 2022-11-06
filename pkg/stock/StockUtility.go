package stock

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/jmirfield/stockbot/internal/util"
)

// Stock is structured stock data
type Stock struct {
	Ticker string  `json:"ticker"`
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int     `json:"volume"`
}

func requestStockHistory(ticker string) []Stock {
	base := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=%v&datatype=csv&outputsize=full&apikey=apikey", ticker)

	resp, err := http.Get(base)

	if err != nil {
		log.Println(err)
	}

	reader := csv.NewReader(resp.Body)
	reader.LazyQuotes = true
	ch := util.ProcessCSV(reader)

	var dataSlice []Stock
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
		stock := Stock{ticker, date, open, high, low, close, volume}
		dataSlice = append(dataSlice, stock)
	}
	return dataSlice
}

// WriteToJSON wil read from stock chan and start
// writing the data to a json file
func WriteToJSON(s chan []Stock) error {
	newf, err := os.Create("stockdata.json")
	if err != nil {
		return err
	}
	defer newf.Close()

	fmt.Fprint(newf, "[")
	defer fmt.Fprint(newf, "]")

	cnt := 0
	for stock := range s {
		if stock == nil {
			continue
		}
		if cnt > 0 {
			fmt.Fprint(newf, ",")
		}
		jsonstock, err := json.Marshal(stock)
		if err != nil {
			return err
		}
		fmt.Fprint(newf, string(jsonstock))
		cnt++
	}

	fmt.Printf("%d stocks found\n", cnt)
	return nil
}
