package misc

type Quote struct {
	Text   string `json:"text"`
	Author string `json:"author"`
	Genre  string `json:"genre"`
}

func NewQuote(text string, author string, genre string) *Quote {
	return &Quote{
		Text:   text,
		Author: author,
		Genre:  genre,
	}
}
