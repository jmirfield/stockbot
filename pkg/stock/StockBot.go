package stock

import (
	"encoding/csv"
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
	guard := make(chan struct{}, maxWorkers)
	return Manager{reader, worker{&wg, nil, guard}}
}

// Start begins the process of reading through the csv file,
// creating a new worker that gets sent off as a go routine,
// and retrieves the given ticker stock data
func (s *Manager) Start() chan []Stock {
	ch := util.ProcessCSV(s.Reader)
	stockch := make(chan []Stock)

	cnt := 0
	for data := range ch {
		if cnt > 0 {
			break
		}
		s.Guard <- struct{}{} // would block if guard channel is already filled
		ticker := data[0]
		worker := s.newWorker(stockch)
		go worker.start(ticker)
		time.Sleep(time.Second * 12)
		cnt++
	}

	go func() {
		s.WaitGroup.Wait()
		close(stockch)
	}()

	return stockch
}

func (s *Manager) newWorker(ch chan []Stock) worker {
	s.WaitGroup.Add(1)
	return worker{s.WaitGroup, ch, s.Guard}
}

type worker struct {
	WaitGroup    *sync.WaitGroup
	StockChannel chan []Stock
	Guard        chan struct{}
}

func (sw *worker) start(ticker string) {
	defer sw.WaitGroup.Done()
	stocks := requestStockHistory(ticker)
	<-sw.Guard
	sw.StockChannel <- stocks
}
