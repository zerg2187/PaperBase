package paperbase

// Arxiv API types

type ArxivFeed struct {
	Entries []ArxivEntry `xml:"entry"`
}

type ArxivEntry struct {
	Title   string        `xml:"title"`
	Summary string        `xml:"summary"`
	Authors []ArxivAuthor `xml:"author"`
}

type ArxivAuthor struct {
	Name string `xml:"name"`
}

// Semantic Scholar API types

type S2ExternalIds struct {
	DOI string `json:"DOI"`
}

type S2Venue struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Url  string `json:"url"`
	Issn string `json:"issn"`
}

type S2Journal struct {
	Name   string `json:"name"`
	Volume string `json:"volume"`
	Issue  string `json:"issue"`
	Pages  string `json:"pages"`
}

type S2Author struct {
	Name string `json:"name"`
}

type S2Response struct {
	ExternalIds      S2ExternalIds `json:"externalIds"`
	Title            string        `json:"title"`
	Abstract         string        `json:"abstract"`
	Authors          []S2Author    `json:"authors"`
	Venue            string        `json:"venue"`
	Year             int           `json:"year"`
	Journal          *S2Journal    `json:"journal"`
	PublicationVenue S2Venue       `json:"publicationVenue"`
	CitationStyles   struct {
		Bibtex string `json:"bibtex"`
	} `json:"citationStyles"`
}
