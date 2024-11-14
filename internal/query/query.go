package query

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/mkeeler/gh-search/internal/logging"
	"github.com/mkeeler/gh-search/internal/paginate"
	"github.com/mkeeler/gh-search/internal/requests"
)

type QueryResults struct {
	RepositoryHits int
	FileHits       int
	Repositories   map[string][]string
}

func (qr *QueryResults) AggregateResultsFromBody(ctx context.Context, rdr io.Reader) error {
	var resp codeSearchResponseFormat[codeSearchResponseFile]
	err := json.NewDecoder(rdr).Decode(&resp)
	if err != nil {
		return fmt.Errorf("Error decoding body as json: %w", err)
	}

	logger := logging.FromContext(ctx)
	for _, item := range resp.Items {
		logger.Debug("search hit",
			slog.String("path", item.Path),
			slog.String("repo", item.Repo.FullName),
		)
		repoFiles, found := qr.Repositories[item.Repo.FullName]
		if !found {
			qr.RepositoryHits += 1
		}
		qr.FileHits += 1

		repoFiles = append(repoFiles, item.Path)
		qr.Repositories[item.Repo.FullName] = repoFiles
	}

	return nil
}

type repoQueryResults struct {
	repositories []string
}

func (qr *repoQueryResults) AggregateResultsFromBody(ctx context.Context, rdr io.Reader) error {
	var resp codeSearchResponseFormat[codeSearchResponseRepository]
	err := json.NewDecoder(rdr).Decode(&resp)
	if err != nil {
		return fmt.Errorf("Error decoding body as json: %w", err)
	}

	logger := logging.FromContext(ctx)
	for _, item := range resp.Items {
		logger.Debug("search hit", slog.String("repo", item.FullName))
		qr.repositories = append(qr.repositories, item.FullName)
	}

	return nil
}

func ExecuteQuery(ctx context.Context, rp requests.RequestParams) (*QueryResults, error) {
	// If we want to filter by repo topic then we must first do a repo query and then
	// separately query for results in each repo. It is less efficient and GitHub rate
	// limiting will kick in causing our overall query to take much longer.
	if rp.Topic != "" {
		return queryPerRepo(ctx, rp)
	}

	return globalQuery(ctx, rp)
}

func queryPerRepo(ctx context.Context, rp requests.RequestParams) (*QueryResults, error) {
	logger := logging.FromContext(ctx).With(
		rp.LoggerAttributes()...,
	)
	logger.Info("executing query per discovered repository")
	ctx = logging.WithContext(ctx, logger)

	results := QueryResults{
		Repositories: make(map[string][]string),
	}

	// First find list of repositories
	logger.Info("searching for list of repositories matching query params")
	var repoResults repoQueryResults
	err := paginate.Paginate(ctx, func(page int) (*http.Request, error) {
		params := rp
		params.Page = page
		return requests.RepoSearchRequest(params)
	}, &repoResults)
	if err != nil {
		return nil, fmt.Errorf("failed to search for repositories: %w", err)
	}

	for _, repo := range repoResults.repositories {
		logger := logger.With(slog.String("repo", repo))
		logger.Info("executing query for repository")
		ctx := logging.WithContext(ctx, logger)
		err := paginate.Paginate(ctx, func(page int) (*http.Request, error) {
			params := rp
			params.Page = page
			params.Repo = repo
			return requests.CodeSearchRequest(params)
		}, &results)
		if err != nil {
			return nil, fmt.Errorf("failed to search for code within repo %s: %w", repo, err)
		}
	}

	return &results, nil
}

func globalQuery(ctx context.Context, rp requests.RequestParams) (*QueryResults, error) {
	logger := logging.FromContext(ctx).With(
		rp.LoggerAttributes()...,
	)
	logger.Info("executing query in multi-repo mode")

	ctx = logging.WithContext(ctx, logger)

	results := QueryResults{
		Repositories: make(map[string][]string),
	}

	err := paginate.Paginate(ctx, func(page int) (*http.Request, error) {
		rp.Page = page
		return requests.CodeSearchRequest(rp)
	}, &results)

	return &results, err
}

type codeSearchAggregator struct {
	results QueryResults
}

type codeSearchResponseFormat[T any] struct {
	TotalCount int `json:"total_count"`
	Items      []T `json:"items"`
}

type codeSearchResponseFile struct {
	Name string                       `json:"name"`
	Path string                       `json:"path"`
	Repo codeSearchResponseRepository `json:"repository"`
}

type codeSearchResponseRepository struct {
	Name     string                  `json:"name"`
	FullName string                  `json:"full_name"`
	Owner    codeSearchResponseOwner `json:"owner"`
}

type codeSearchResponseOwner struct {
	Name string `json:"login"`
}
