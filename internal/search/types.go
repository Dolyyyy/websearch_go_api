package search

type Result struct {
	Title       string `json:"title" example:"The Go Programming Language"`
	URL         string `json:"url" example:"https://go.dev/"`
	Description string `json:"description" example:"Go is an open source programming language..."`
	Provider    string `json:"provider" example:"bing"`
}

type Request struct {
	Query  string
	Offset int
	Limit  int
}

type Page struct {
	Provider string
	Results  []Result
	HasNext  bool
}

type ProviderInfo struct {
	Name        string `json:"name" example:"bing"`
	Description string `json:"description" example:"Microsoft Bing HTML search"`
}

type Answer struct {
	Text   string `json:"text" example:"Demain à Thouars, attention, fortes rafales. Les temperatures varieront entre 8 et 12 C."`
	Title  string `json:"title" example:"Meteo Thouars Demain (79100) - Deux-Sevres - La Chaine Meteo"`
	URL    string `json:"url" example:"https://www.lachainemeteo.com/meteo-france/ville-12021/previsions-meteo-thouars-demain"`
	Source string `json:"source" example:"duckduckgo"`
}
