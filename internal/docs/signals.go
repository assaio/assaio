package docs

import "github.com/assaio/assaio/internal/signal"

// Signal is one entry of the signal catalog (ADR 0008). ZeroMeans travels with it because a
// reference that lists what can be reported without saying what a zero means would publish the
// exact confusion the catalog was built to prevent.
type Signal struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	Unit      string   `json:"unit"`
	Status    string   `json:"status"`
	Grains    []string `json:"grains"`
	ZeroMeans string   `json:"zeroMeans"`
	Describe  string   `json:"describe"`
	// Sources names the in-tree tools that can answer this signal, from the depth matrix.
	// Empty means no shipped source can, which is a fact about assaio worth publishing.
	Sources []string `json:"sources"`
}

func signals() []Signal {
	catalog := signal.Catalog()
	out := make([]Signal, 0, len(catalog))
	for _, s := range catalog {
		out = append(out, Signal{
			ID:        s.ID,
			Title:     s.Title,
			Unit:      s.Unit,
			Status:    s.Status,
			Grains:    s.Grains,
			ZeroMeans: s.ZeroMeans,
			Describe:  s.Describe,
			Sources:   sourcesAnswering(s.ID),
		})
	}
	return out
}
