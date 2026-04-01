package search

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	DefaultProvider = "duckduckgo"
	AutoProvider    = "auto"
	MaxLimit        = 25
)

type Provider interface {
	Info() ProviderInfo
	Search(ctx context.Context, req Request) (Page, error)
}

type Service struct {
	providers map[string]Provider
	order     []string
}

type unsupportedProviderError struct {
	name string
}

func (e unsupportedProviderError) Error() string {
	return fmt.Sprintf("unsupported provider: %s", e.name)
}

func IsUnsupportedProvider(err error) bool {
	var target unsupportedProviderError
	return errors.As(err, &target)
}

func NewService() *Service {
	registered := []Provider{
		NewBingProvider(),
		NewDuckDuckGoProvider(),
	}

	providers := make(map[string]Provider, len(registered))
	order := make([]string, 0, len(registered))
	for _, provider := range registered {
		info := provider.Info()
		providers[info.Name] = provider
		order = append(order, info.Name)
	}

	return &Service{
		providers: providers,
		order:     order,
	}
}

func (s *Service) Providers() []ProviderInfo {
	out := make([]ProviderInfo, 0, len(s.providers)+1)
	out = append(out, ProviderInfo{
		Name:        AutoProvider,
		Description: "Essaie les providers disponibles dans l'ordre jusqu'au premier succès",
	})

	names := append([]string(nil), s.order...)
	sort.Strings(names)
	for _, name := range names {
		out = append(out, s.providers[name].Info())
	}

	return out
}

func (s *Service) Search(ctx context.Context, requestedProvider string, req Request) (Page, error) {
	if req.Limit < 1 || req.Limit > MaxLimit {
		return Page{}, fmt.Errorf("limit must be between 1 and %d", MaxLimit)
	}
	if req.Offset < 0 {
		return Page{}, fmt.Errorf("offset must be >= 0")
	}

	name := strings.ToLower(strings.TrimSpace(requestedProvider))
	if name == "" {
		name = DefaultProvider
	}
	if name == AutoProvider {
		return s.searchAuto(ctx, req)
	}

	provider, ok := s.providers[name]
	if !ok {
		return Page{}, unsupportedProviderError{name: requestedProvider}
	}

	return provider.Search(ctx, req)
}

func (s *Service) searchAuto(ctx context.Context, req Request) (Page, error) {
	var errs []string
	for _, name := range s.order {
		page, err := s.providers[name].Search(ctx, req)
		if err == nil {
			return page, nil
		}
		errs = append(errs, fmt.Sprintf("%s: %v", name, err))
	}

	return Page{}, fmt.Errorf("all providers failed: %s", strings.Join(errs, "; "))
}
