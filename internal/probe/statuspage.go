package probe

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/yosh3289/speedforce/internal/core"
)

type StatuspageProber struct {
	client *http.Client
}

func NewStatuspageProber(client *http.Client) *StatuspageProber {
	return &StatuspageProber{client: client}
}

type statuspageDoc struct {
	Status struct {
		Indicator   string `json:"indicator"`
		Description string `json:"description"`
	} `json:"status"`
}

func (p *StatuspageProber) Probe(ctx context.Context, name, url string) core.StatuspageResult {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := p.client.Do(req)
	if err != nil {
		return core.StatuspageResult{Name: name, Err: err, FetchedAt: time.Now()}
	}
	defer resp.Body.Close()

	var doc statuspageDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return core.StatuspageResult{Name: name, Err: err, FetchedAt: time.Now()}
	}
	return core.StatuspageResult{
		Name:        name,
		Indicator:   core.StatuspageIndicator(doc.Status.Indicator),
		Description: doc.Status.Description,
		FetchedAt:   time.Now(),
	}
}
