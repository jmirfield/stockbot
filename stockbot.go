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
	"time"
)

func main() {
	csvf, err := os.Open("./screener.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer csvf.Close()

	manager := newStockManger(csvf, 10)
	err = manager.start()
	if err != nil {
		log.Fatal(err)
	}
}

type stockManager struct {
	Reader     *csv.Reader
	WaitGroup  *sync.WaitGroup
	MaxWorkers int
}

func newStockManger(f *os.File, maxWorkers int) stockManager {
	reader := csv.NewReader(f)
	var wg sync.WaitGroup
	return stockManager{reader, &wg, maxWorkers}
}

func (s *stockManager) start() error {
	ch := processCSV(s.Reader)

	stockch := make(chan []stock)
	guard := make(chan struct{}, s.MaxWorkers)

	for data := range ch {
		guard <- struct{}{} // would block if guard channel is already filled
		ticker := data[0]
		s.WaitGroup.Add(1)
		go requestStockHistory(ticker, stockch, s.WaitGroup, guard)
		time.Sleep(time.Millisecond * 100)
	}

	go func() {
		s.WaitGroup.Wait()
		close(stockch)
	}()

	err := s.writeToFile(stockch)

	if err != nil {
		return err
	}
	return nil
}

func (s *stockManager) writeToFile(ch chan []stock) error {
	newf, err := os.Create("stockdata.json")
	if err != nil {
		return err
	}
	defer newf.Close()

	fmt.Fprint(newf, "[")
	defer fmt.Fprint(newf, "]")

	cnt := 0
	for stock := range ch {
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

	fmt.Printf("%d stocks", cnt)
	return nil
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

type stock struct {
	Ticker string  `json:"ticker"`
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int     `json:"volume"`
}

func requestStockHistory(ticker string, c chan []stock, wg *sync.WaitGroup, guard chan struct{}) {
	defer wg.Done()
	base := fmt.Sprintf("https://www.alphavantage.co/query?function=TIME_SERIES_DAILY&symbol=%v&datatype=csv&outputsize=full&apikey=apikey", ticker)

	resp, err := http.Get(base)

	if err != nil {
		log.Println(err)
	}

	reader := csv.NewReader(resp.Body)
	reader.LazyQuotes = true
	ch := processCSV(reader)

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
	<-guard
	c <- dataSlice
}
