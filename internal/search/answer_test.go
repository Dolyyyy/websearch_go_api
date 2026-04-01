package search

import "testing"

func TestBuildAnswerPrefersWeatherResultForWeatherQuery(t *testing.T) {
	results := []Result{
		{
			Title:       "METEO THOUARS par Meteo-France",
			URL:         "https://meteofrance.com/previsions-meteo-france/thouars/79100",
			Description: "Retrouvez les previsions meteo Thouars pour aujourd'hui, demain et jusqu'a 15 jours.",
			Provider:    "duckduckgo",
		},
		{
			Title:       "Meteo Thouars Demain (79100) - La Chaine Meteo",
			URL:         "https://www.lachainemeteo.com/meteo-france/ville-12021/previsions-meteo-thouars-demain",
			Description: "Demain a Thouars, attention, fortes rafales. Les temperatures varieront entre 8 et 12 C.",
			Provider:    "duckduckgo",
		},
	}

	answer := BuildAnswer("Meteo thouars demain", results)
	if answer == nil {
		t.Fatal("expected answer, got nil")
	}
	if answer.URL != results[1].URL {
		t.Fatalf("expected best weather result, got %s", answer.URL)
	}
}
