package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
)

type stock struct {
	Symbol string
}

func main() {
	csvf, err := os.Open("./screener.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer csvf.Close()

	reader := csv.NewReader(csvf)
	ch := processCSV(reader)

	stockch := make(chan []stockData)
	var wg sync.WaitGroup

	for data := range ch {
		ticker := data[0]
		wg.Add(1)
		go requestStockHistory(ticker, stockch, &wg)
	}

	go func() {
		wg.Wait()
		close(stockch)
	}()

	newf, err := os.Create("stockdata.json")
	if err != nil {
		log.Fatal(err)
	}
	defer newf.Close()

	fmt.Fprint(newf, "[")
	defer fmt.Fprint(newf, "]")

	cnt := 0
	for stock := range stockch {
		if stock == nil {
			continue
		}
		if cnt > 0 {
			fmt.Fprint(newf, ",")
		}
		jsonstock, err := json.Marshal(stock)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprint(newf, string(jsonstock))
		cnt++
	}

}

func processCSV(cr *csv.Reader) (ch chan []string) {
	ch = make(chan []string, 10)
	go func() {
		//Read the header line
		if _, err := cr.Read(); err != nil {
			log.Fatal(err)
		}
		defer close(ch)

		for {
			rec, err := cr.Read()
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Fatal(err)
			}
			ch <- rec
		}
	}()
	return
}

type stockData struct {
	Ticker string  `json:"ticker"`
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int     `json:"volume"`
}

func requestStockHistory(ticker string, c chan []stockData, wg *sync.WaitGroup) {
	defer wg.Done()
	base := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=%v&datatype=csv&outputsize=full&apikey=apikey", ticker)

	resp, err := http.Get(base)

	if err != nil {
		log.Println(err)
	}

	reader := csv.NewReader(resp.Body)
	reader.LazyQuotes = true
	ch := processCSV(reader)

	var dataSlice []stockData
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
		stock := stockData{ticker, date, open, high, low, close, volume}
		dataSlice = append(dataSlice, stock)
	}
	c <- dataSlice
}
