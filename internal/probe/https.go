package probe

import (
	"context"
	"net/http"
	"time"

	"github.com/yosh3289/speedforce/internal/core"
)

type HTTPSProber struct {
	client *http.Client
}

func NewHTTPSProber(client *http.Client) *HTTPSProber {
	return &HTTPSProber{client: client}
}

func (p *HTTPSProber) Probe(ctx context.Context, name, rawURL string) core.ProbeResult {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, nil)
	if err != nil {
		return core.ProbeResult{Name: name, URL: rawURL, Err: err, Timestamp: time.Now()}
	}
	req.Header.Set("User-Agent", "SpeedForce/1.0 (+probe)")

	resp, err := p.client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return core.ProbeResult{
			Name: name, URL: rawURL, Err: err,
			LatencyMs: latency, Timestamp: time.Now(),
		}
	}
	defer resp.Body.Close()
	return core.ProbeResult{
		Name: name, URL: rawURL, StatusCode: resp.StatusCode,
		LatencyMs: latency, Timestamp: time.Now(),
	}
}
