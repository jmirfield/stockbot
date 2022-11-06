package main

import (
	"log"
	"os"

	"github.com/jmirfield/stockbot/pkg/stock"
)

func main() {
	csvf, err := os.Open("./screener.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer csvf.Close()

	manager := stock.NewManager(csvf, 10)
	stockchannel := manager.Start()
	stock.WriteToJSON(stockchannel)
}
