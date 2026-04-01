package search

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type bingProvider struct {
	httpClient *http.Client
	baseURL    string
}

func NewBingProvider() Provider {
	return &bingProvider{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		baseURL:    "https://www.bing.com/search",
	}
}

func (p *bingProvider) Info() ProviderInfo {
	return ProviderInfo{
		Name:        "bing",
		Description: "Microsoft Bing HTML search",
	}
}

func (p *bingProvider) Search(ctx context.Context, req Request) (Page, error) {
	collected := make([]Result, 0, req.Limit+1)
	offset := req.Offset
	hasNextPage := false

	for len(collected) < req.Limit+1 {
		results, nextPage, err := p.fetch(ctx, req.Query, offset)
		if err != nil {
			return Page{}, err
		}
		if len(results) == 0 {
			hasNextPage = false
			break
		}

		hasNextPage = nextPage
		remaining := req.Limit + 1 - len(collected)
		if len(results) > remaining {
			results = results[:remaining]
		}
		collected = append(collected, results...)

		offset += len(results)
		if !nextPage {
			break
		}
	}

	hasNext := false
	if len(collected) > req.Limit {
		hasNext = true
		collected = collected[:req.Limit]
	} else if len(collected) == req.Limit && hasNextPage {
		hasNext = true
	}

	return Page{
		Provider: "bing",
		Results:  collected,
		HasNext:  hasNext,
	}, nil
}

func (p *bingProvider) fetch(ctx context.Context, query string, offset int) ([]Result, bool, error) {
	params := url.Values{}
	params.Set("q", query)
	params.Set("first", strconv.Itoa(offset+1))
	params.Set("count", "50")
	params.Set("mkt", "fr-FR")
	params.Set("setlang", "fr-FR")
	params.Set("cc", "fr")

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, false, err
	}

	httpReq.Header.Set("User-Agent", "Mozilla/5.0 (compatible; search-api/2.0)")
	httpReq.Header.Set("Accept-Language", "fr-FR,fr;q=0.9,en;q=0.7")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false, fmt.Errorf("bing returned status %d", resp.StatusCode)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, false, err
	}

	results := make([]Result, 0, 16)
	doc.Find("li.b_algo").Each(func(_ int, selection *goquery.Selection) {
		titleNode := selection.Find("h2 a").First()
		title := strings.TrimSpace(titleNode.Text())
		link, _ := titleNode.Attr("href")
		description := strings.TrimSpace(selection.Find(".b_caption p").First().Text())
		if title == "" || link == "" {
			return
		}

		results = append(results, Result{
			Title:       title,
			URL:         normalizeBingURL(strings.TrimSpace(link)),
			Description: description,
			Provider:    "bing",
		})
	})

	return results, doc.Find("a.sb_pagN").Length() > 0, nil
}

func normalizeBingURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return raw
	}

	target := parsed.Query().Get("u")
	if target == "" {
		return raw
	}

	if strings.HasPrefix(target, "a1") {
		target = target[2:]
	}

	decoded, err := base64.RawStdEncoding.DecodeString(target)
	if err == nil && strings.HasPrefix(string(decoded), "http") {
		return string(decoded)
	}

	decoded, err = base64.StdEncoding.DecodeString(target)
	if err == nil && strings.HasPrefix(string(decoded), "http") {
		return string(decoded)
	}

	return raw
}
