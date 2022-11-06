package main

import (
	"log"
	"os"
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
