package util

import (
	"encoding/csv"
	"io"
	"log"
)

func ProcessCSV(cr *csv.Reader) (ch chan []string) {
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
