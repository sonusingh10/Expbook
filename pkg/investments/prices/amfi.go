package prices

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mayswind/ezbookkeeping/pkg/investments"
)

// DefaultAmfiURL is the public AMFI file with the latest NAV of every Indian mutual fund scheme
const DefaultAmfiURL = "https://www.amfiindia.com/spages/NAVAll.txt"

// Amfi fetches the latest NAV of Indian mutual funds, keyed by scheme code, from the public AMFI file (no API key)
type Amfi struct {
	URL    string
	Client *http.Client
}

// Name returns the price source name
func (a *Amfi) Name() string {
	return "amfi"
}

// Fetch downloads the file once and looks up every requested scheme code
func (a *Amfi) Fetch(ctx context.Context, identifiers []string) map[string]Result {
	results := make(map[string]Result, len(identifiers))

	navs, err := a.download(ctx)

	for _, identifier := range identifiers {
		if err != nil {
			results[identifier] = Result{Err: err}
			continue
		}

		quote, ok := navs[strings.TrimSpace(identifier)]

		if !ok {
			results[identifier] = Result{Err: ErrNoQuote}
			continue
		}

		results[identifier] = Result{Quote: quote}
	}

	return results
}

func (a *Amfi) download(ctx context.Context) (map[string]Quote, error) {
	target := a.URL

	if target == "" {
		target = DefaultAmfiURL
	}

	client := a.Client

	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)

	if err != nil {
		return nil, err
	}

	request.Header.Set("User-Agent", "Mozilla/5.0 (compatible; Expbook/1.0)")

	response, err := client.Do(request)

	if err != nil {
		return nil, err
	}

	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("amfi returned http status %d", response.StatusCode)
	}

	return ParseAmfiNav(io.LimitReader(response.Body, 20<<20))
}

// ParseAmfiNav parses the semicolon separated NAVAll.txt format:
// Scheme Code;ISIN Div Payout/ISIN Growth;ISIN Div Reinvestment;Scheme Name;Net Asset Value;Date
// Header and category lines have no semicolons and are skipped. Rows with a missing or "N.A." NAV are skipped.
func ParseAmfiNav(reader io.Reader) (map[string]Quote, error) {
	navs := map[string]Quote{}
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)

	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ";")

		if len(fields) < 6 {
			continue
		}

		code := strings.TrimSpace(fields[0])

		if code == "" || code == "Scheme Code" {
			continue
		}

		nav, err := investments.ParseScaled(strings.TrimSpace(fields[4]))

		if err != nil || nav <= 0 {
			continue
		}

		navTime := time.Now().Unix()

		if parsed, err := time.Parse("02-Jan-2006", strings.TrimSpace(fields[5])); err == nil {
			navTime = parsed.Add(12 * time.Hour).Unix()
		}

		navs[code] = Quote{Price: nav, Time: navTime, Currency: "INR"}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if len(navs) == 0 {
		return nil, fmt.Errorf("amfi file contained no NAV rows")
	}

	return navs, nil
}
