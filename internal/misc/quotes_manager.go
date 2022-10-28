package misc

import (
	"encoding/csv"
	"fmt"
	"io"
	"math/rand"

	log "github.com/sirupsen/logrus"
)

type Quote struct {
	Text   string `json:"text"`
	Author string `json:"author"`
	Genre  string `json:"genre"`
}

type QuotesManager struct {
	Quotes        []*Quote
	AuthorsQuotes map[string][]*Quote
	GenresQuotes  map[string][]*Quote
}

func NewQuoteManager(quotesCsvReader *csv.Reader) (*QuotesManager, error) {
	qm := &QuotesManager{}
	qm.AuthorsQuotes = make(map[string][]*Quote)
	qm.GenresQuotes = make(map[string][]*Quote)

	log.Println("reading quotes CSV ...")

	quotesCsvReader.Comma = ';'
	for {
		record, err := quotesCsvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if len(record) != 3 {
			return nil, fmt.Errorf("record [%s] does not have 3 elements", record)
		}

		// QUOTE;AUTHOR;GENRE
		quoteText := record[0]
		author := record[1]
		genre := record[2]

		quote := &Quote{
			Text:   quoteText,
			Author: author,
			Genre:  genre,
		}
		qm.Quotes = append(qm.Quotes, quote)

		qm.AuthorsQuotes[author] = append(qm.AuthorsQuotes[author], quote)
		qm.GenresQuotes[genre] = append(qm.GenresQuotes[genre], quote)
	}

	log.Printf("quotes CSV read %d quotes", len(qm.Quotes))

	return qm, nil
}

func (qm *QuotesManager) RandomQuote() *Quote {
	index := rand.Float64() * float64(len(qm.Quotes))
	return qm.Quotes[int(index)]
}
