package paginate

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"strconv"

	"github.com/mkeeler/gh-search/internal/logging"
	"github.com/mkeeler/gh-search/internal/ratelimit"
)

var (
	linkRe = regexp.MustCompile(`<.*?[\?&]page=(\d+).*?>; rel="([^"]+)"`)
)

type QueryBuilder func(page int) (*http.Request, error)

type ResultsAggregator interface {
	AggregateResultsFromBody(ctx context.Context, rdr io.Reader) error
}

func Paginate(ctx context.Context, qb QueryBuilder, ra ResultsAggregator) error {
	logger := logging.FromContext(ctx)
	logger.Log(ctx, logging.LevelTrace, "paginating query")
	done := false
	page := 1
	for !done {
		logger.LogAttrs(ctx, logging.LevelTrace, "building query for page", slog.Int("page", page))
		req, err := qb(page)
		if err != nil {
			return err
		}

		logger.LogAttrs(ctx, logging.LevelTrace, "requesting page", slog.String("url", req.URL.String()), slog.Int("page", page))
		resp, err := ratelimit.RateLimitRequest(ctx, req)
		if err != nil {
			return err
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("HTTP request failed with code %d: %s", resp.StatusCode, body)
		}

		err = ra.AggregateResultsFromBody(ctx, resp.Body)
		if err != nil {
			return err
		}

		links := make(map[string]int)
		for _, hdr := range resp.Header.Values("Link") {
			for _, groups := range linkRe.FindAllStringSubmatch(hdr, -1) {
				page, err := strconv.ParseInt(groups[1], 10, 32)
				if err != nil {
					return fmt.Errorf("HTTP response contained link with an unparseable page number - %q: %w", groups[1], err)
				}
				links[groups[2]] = int(page)
			}
		}

		if nextPage, found := links["next"]; found {
			page = nextPage
		} else {
			done = true
		}
	}

	return nil
}
