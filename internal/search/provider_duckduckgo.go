package search

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type duckDuckGoProvider struct {
	httpClient *http.Client
	baseURL    string
}

func NewDuckDuckGoProvider() Provider {
	return &duckDuckGoProvider{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    "https://html.duckduckgo.com/html/",
	}
}

func (p *duckDuckGoProvider) Info() ProviderInfo {
	return ProviderInfo{
		Name:        "duckduckgo",
		Description: "DuckDuckGo HTML search",
	}
}

func (p *duckDuckGoProvider) Search(ctx context.Context, req Request) (Page, error) {
	form := url.Values{}
	form.Set("q", req.Query)

	skip := req.Offset
	collected := make([]Result, 0, req.Limit+1)

	for {
		results, nextForm, err := p.fetch(ctx, form)
		if err != nil {
			return Page{}, err
		}
		if len(results) == 0 {
			break
		}

		if skip >= len(results) {
			skip -= len(results)
		} else {
			results = results[skip:]
			skip = 0
			remaining := req.Limit + 1 - len(collected)
			if remaining > 0 {
				if len(results) > remaining {
					results = results[:remaining]
				}
				collected = append(collected, results...)
			}
		}

		if len(collected) >= req.Limit+1 || len(nextForm) == 0 {
			form = nextForm
			break
		}

		form = nextForm
	}

	hasNext := len(collected) > req.Limit
	if hasNext {
		collected = collected[:req.Limit]
	}

	return Page{
		Provider: "duckduckgo",
		Results:  collected,
		HasNext:  hasNext,
	}, nil
}

func (p *duckDuckGoProvider) fetch(ctx context.Context, form url.Values) ([]Result, url.Values, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, nil, err
	}

	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (compatible; search-api/2.0)")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("duckduckgo returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, nil, err
	}

	if doc.Find(".anomaly-modal__title").Length() > 0 {
		return nil, nil, fmt.Errorf("duckduckgo anti-bot challenge triggered")
	}

	results := make([]Result, 0, 10)
	doc.Find("#links .result").Each(func(_ int, selection *goquery.Selection) {
		if selection.Find(".badge--ad").Length() > 0 {
			return
		}

		titleNode := selection.Find(".result__title a").First()
		title := strings.TrimSpace(titleNode.Text())
		link, _ := titleNode.Attr("href")
		description := strings.TrimSpace(selection.Find(".result__snippet").First().Text())
		if title == "" || link == "" {
			return
		}

		results = append(results, Result{
			Title:       title,
			URL:         strings.TrimSpace(link),
			Description: description,
			Provider:    "duckduckgo",
		})
	})

	nextForm := url.Values{}
	nextNode := doc.Find(".nav-link form").Last()
	nextNode.Find("input").Each(func(_ int, input *goquery.Selection) {
		name, ok := input.Attr("name")
		if !ok || name == "" {
			return
		}

		value, _ := input.Attr("value")
		nextForm.Set(name, value)
	})

	if len(nextForm) == 0 {
		nextForm = nil
	}

	return results, nextForm, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
