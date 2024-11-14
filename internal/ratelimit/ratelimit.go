package ratelimit

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/mkeeler/gh-search/internal/logging"
)

func RateLimitRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
	logger := logging.FromContext(ctx)
	for {
		logger.Debug("performing HTTP request", slog.String("url", req.URL.String()))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode == http.StatusForbidden {
			remainingStr := resp.Header.Get("X-Ratelimit-Remaining")
			remaining, err := strconv.ParseInt(remainingStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse X-RateLimit-Remaining header value %q: %w", remainingStr, err)
			}

			if remaining > 0 {
				// got the 403 for another reason so return the error
				return resp, nil
			}

			resetDateStr := resp.Header.Get("X-Ratelimit-Reset")
			resetDate, err := strconv.ParseInt(resetDateStr, 10, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse X-RateLimit-Reset header value %q: %w", remainingStr, err)
			}

			waitSeconds := resetDate - time.Now().Unix()

			logger.Debug("rate limit hit", slog.Int64("wait-seconds", waitSeconds))

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Duration(waitSeconds) * time.Second):
			}
		} else {
			return resp, nil
		}
	}
}
