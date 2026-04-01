package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"

	"search-api/internal/search"
)

type healthOutput struct {
	Body struct {
		OK bool `json:"ok"`
	}
}

type providersOutput struct {
	Body struct {
		DefaultProvider string                `json:"default_provider"`
		Providers       []search.ProviderInfo `json:"providers"`
	}
}

type searchOutput struct {
	Body struct {
		Mode              string          `json:"mode"`
		Query             string          `json:"query"`
		Page              int             `json:"page"`
		Limit             int             `json:"limit"`
		Count             int             `json:"count"`
		RequestedProvider string          `json:"requested_provider"`
		Provider          string          `json:"provider"`
		HasPrev           bool            `json:"has_prev"`
		HasNext           bool            `json:"has_next"`
		PrevPage          *int            `json:"prev_page,omitempty"`
		NextPage          *int            `json:"next_page,omitempty"`
		Answer            *search.Answer  `json:"answer,omitempty"`
		Results           []search.Result `json:"results"`
	}
}

func Run() error {
	searchService := search.NewService()
	mux := http.NewServeMux()

	config := huma.DefaultConfig("Simple Search API", "2.0.0")
	config.Info.Description = "API de recherche modulaire avec pagination, choix du provider et Swagger UI."
	config.DocsPath = "/docs"
	config.OpenAPIPath = "/openapi.json"
	config.DocsRenderer = huma.DocsRendererSwaggerUI

	api := humago.New(mux, config)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		http.Redirect(w, r, "/docs", http.StatusTemporaryRedirect)
	})

	huma.Get(api, "/health", func(ctx context.Context, input *struct{}) (*healthOutput, error) {
		out := &healthOutput{}
		out.Body.OK = true
		return out, nil
	})

	huma.Get(api, "/providers", func(ctx context.Context, input *struct{}) (*providersOutput, error) {
		out := &providersOutput{}
		out.Body.DefaultProvider = search.DefaultProvider
		out.Body.Providers = searchService.Providers()
		return out, nil
	})

	huma.Get(api, "/search", func(ctx context.Context, input *struct {
		Q        string `query:"q" doc:"Texte de recherche" example:"golang"`
		Page     int    `query:"page" doc:"Numéro de page, à partir de 1" default:"1"`
		Limit    int    `query:"limit" doc:"Nombre de résultats par page (1-25)" default:"10"`
		Provider string `query:"provider" doc:"Provider: duckduckgo, bing, auto" default:"duckduckgo"`
		Mode     string `query:"mode" doc:"Mode de réponse: results ou answer" default:"results"`
	}) (*searchOutput, error) {
		q := strings.TrimSpace(input.Q)
		if q == "" {
			return nil, huma.Error400BadRequest("missing query parameter: q")
		}
		if input.Page < 1 {
			return nil, huma.Error400BadRequest("page must be >= 1")
		}
		if input.Limit < 1 || input.Limit > search.MaxLimit {
			return nil, huma.Error400BadRequest(fmt.Sprintf("limit must be between 1 and %d", search.MaxLimit))
		}
		mode := strings.ToLower(strings.TrimSpace(input.Mode))
		if mode == "" {
			mode = "results"
		}
		if mode != "results" && mode != "answer" {
			return nil, huma.Error400BadRequest("mode must be one of: results, answer")
		}

		requestedProvider := strings.TrimSpace(input.Provider)
		if requestedProvider == "" {
			requestedProvider = search.DefaultProvider
		}

		offset := (input.Page - 1) * input.Limit
		pageResult, err := searchService.Search(ctx, requestedProvider, search.Request{
			Query:  q,
			Offset: offset,
			Limit:  input.Limit,
		})
		if err != nil {
			switch {
			case search.IsUnsupportedProvider(err):
				return nil, huma.Error400BadRequest(err.Error())
			default:
				return nil, huma.Error502BadGateway("search provider failed: " + err.Error())
			}
		}

		out := &searchOutput{}
		out.Body.Mode = mode
		out.Body.Query = q
		out.Body.Page = input.Page
		out.Body.Limit = input.Limit
		out.Body.Count = len(pageResult.Results)
		out.Body.RequestedProvider = requestedProvider
		out.Body.Provider = pageResult.Provider
		out.Body.HasPrev = input.Page > 1
		out.Body.HasNext = pageResult.HasNext
		out.Body.Results = pageResult.Results
		if mode == "answer" {
			out.Body.Answer = search.BuildAnswer(q, pageResult.Results)
		}

		if out.Body.HasPrev {
			prevPage := input.Page - 1
			out.Body.PrevPage = &prevPage
		}
		if out.Body.HasNext {
			nextPage := input.Page + 1
			out.Body.NextPage = &nextPage
		}

		return out, nil
	})

	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	addr := ":" + port
	fmt.Printf("listening on %s\n", addr)
	fmt.Printf("docs: http://127.0.0.1:%s/docs\n", port)
	fmt.Printf("openapi: http://127.0.0.1:%s/openapi.json\n", port)

	return http.ListenAndServe(addr, cors(mux))
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
