package stock

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/jmirfield/stockbot/internal/util"
)

// Manager handles reading through csv file
// and creating/sending workers to request stock data
// api and then write to file
type Manager struct {
	Reader *csv.Reader
	worker
}

// NewManager takes in a file to be read and an int
// representing the max number of workers at any given time
// and returns a Manager which handles reading stock file
// to find tickers, and creating workers to send requests to
// stock data api and writing data to file
func NewManager(f *os.File, maxWorkers int) Manager {
	var wg sync.WaitGroup
	reader := csv.NewReader(f)
	stockch := make(chan []stock)
	guard := make(chan struct{}, maxWorkers)
	return Manager{reader, worker{&wg, stockch, guard}}
}

// Start begins the process of reading through the csv file
// and creating a new worker that gets sent off as a go routine
// and retrieves the given ticker stock data
func (s *Manager) Start() error {
	ch := util.ProcessCSV(s.Reader)

	for data := range ch {
		s.Guard <- struct{}{} // would block if guard channel is already filled
		ticker := data[0]
		worker := s.newWorker()
		go worker.start(ticker)
		time.Sleep(time.Second * 20)
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

func (s *Manager) writeToFile() error {
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

func (s *Manager) newWorker() worker {
	s.WaitGroup.Add(1)
	return worker{s.WaitGroup, s.StockChannel, s.Guard}
}

type worker struct {
	WaitGroup    *sync.WaitGroup
	StockChannel chan []stock
	Guard        chan struct{}
}

func (sw *worker) start(ticker string) {
	defer sw.WaitGroup.Done()
	stocks := requestStockHistory(ticker)
	<-sw.Guard
	sw.StockChannel <- stocks
}
