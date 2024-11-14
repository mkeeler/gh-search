package requests

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

const (
	ghAPIVersion = "2022-11-28"
)

type RequestParams struct {
	Owner     string
	Extension string
	RepoQuery string
	Query     string
	Filename  string
	Topic     string
	Repo      string
	Token     string
	Page      int
}

func (rp RequestParams) LoggerAttributes() []any {
	attrs := make([]any, 0, 8)
	if rp.Owner != "" {
		attrs = append(attrs, slog.String("owner", rp.Owner))
	}
	if rp.Extension != "" {
		attrs = append(attrs, slog.String("extension", rp.Extension))
	}
	if rp.Query != "" {
		attrs = append(attrs, slog.String("query", rp.Query))
	}
	if rp.Filename != "" {
		attrs = append(attrs, slog.String("filename", rp.Filename))
	}
	if rp.Topic != "" {
		attrs = append(attrs, slog.String("topic", rp.Topic))
	}
	if rp.Repo != "" {
		attrs = append(attrs, slog.String("repo", rp.Repo))
	}
	if rp.Page != 0 {
		attrs = append(attrs, slog.Int("page", rp.Page))
	}

	return attrs
}

func RepoSearchRequest(rp RequestParams) (*http.Request, error) {
	return commonRequest("/search/repositories", rp.Page, buildRepoSearchQuery(rp), rp.Token)
}

func buildRepoSearchQuery(rp RequestParams) string {
	parts := make([]string, 0, 5)
	if rp.RepoQuery != "" {
		parts = append(parts, rp.RepoQuery)
	}

	if rp.Topic != "" {
		parts = append(parts, rp.Topic)
	}

	if rp.Owner != "" {
		parts = append(parts, "org:"+rp.Owner)
	}

	return strings.Join(parts, " ")
}

func CodeSearchRequest(rp RequestParams) (*http.Request, error) {
	return commonRequest("/search/code", rp.Page, buildCodeSearchQuery(rp), rp.Token)
}

func buildCodeSearchQuery(rp RequestParams) string {
	parts := make([]string, 0, 5)
	parts = append(parts, rp.Query)

	if rp.Owner != "" {
		parts = append(parts, "org:"+rp.Owner)
	}

	if rp.Repo != "" {
		parts = append(parts, "repo:"+rp.Repo)
	}

	if rp.Extension != "" {
		parts = append(parts, "extension:"+rp.Extension)
	}

	if rp.Filename != "" {
		parts = append(parts, "filename:"+rp.Filename)
	}

	return strings.Join(parts, " ")
}

func commonRequest(path string, page int, query string, token string) (*http.Request, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("https://api.github.com%s", path), nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	q := req.URL.Query()
	if page > 0 {
		q.Add("page", strconv.FormatInt(int64(page), 10))
	}

	q.Add("q", query)
	// max amount GH will allow per page
	q.Add("per_page", "100")

	req.URL.RawQuery = q.Encode()
	req.Header.Add("X-GitHub-Api-Version", ghAPIVersion)
	req.Header.Add("Accept", "application/vnd.github+json")
	req.Header.Add("Authorization", "Bearer "+token)

	return req, nil
}
