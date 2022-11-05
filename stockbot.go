package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"

	"github.com/jmirfield/stockbot/internal/util"
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
	Reader *csv.Reader
	stockWorker
}

func newStockManger(f *os.File, maxWorkers int) stockManager {
	var wg sync.WaitGroup
	reader := csv.NewReader(f)
	stockch := make(chan []stock)
	guard := make(chan struct{}, maxWorkers)
	return stockManager{reader, stockWorker{&wg, stockch, guard}}
}

func (s *stockManager) start() error {
	ch := util.ProcessCSV(s.Reader)

	for data := range ch {
		s.Guard <- struct{}{} // would block if guard channel is already filled
		ticker := data[0]
		worker := s.newStockWorker()
		go worker.start(ticker)
		// time.Sleep(time.Second * 20)
	}

	go func() {
		s.WaitGroup.Wait()
		close(s.StockChannel)
	}()

	err := s.writeToFile()
	if err != nil {
		return err
	}

	return nil
}

func (s *stockManager) writeToFile() error {
	newf, err := os.Create("stockdata.json")
	if err != nil {
		return err
	}
	defer newf.Close()

	fmt.Fprint(newf, "[")
	defer fmt.Fprint(newf, "]")

	cnt := 0
	for stock := range s.StockChannel {
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

func (s *stockManager) newStockWorker() stockWorker {
	s.WaitGroup.Add(1)
	return stockWorker{s.WaitGroup, s.StockChannel, s.Guard}
}

type stockWorker struct {
	WaitGroup    *sync.WaitGroup
	StockChannel chan []stock
	Guard        chan struct{}
}

func (sw *stockWorker) start(ticker string) {
	defer sw.WaitGroup.Done()
	stocks := requestStockHistory(ticker)
	<-sw.Guard
	sw.StockChannel <- stocks
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
