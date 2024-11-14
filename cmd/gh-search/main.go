package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	"github.com/mkeeler/gh-search/internal/logging"
	"github.com/mkeeler/gh-search/internal/query"
	"github.com/mkeeler/gh-search/internal/requests"
)

type command struct {
	requests.RequestParams
	logger       *slog.Logger
	logLevel     slog.Level
	logJson      bool
	outputFormat string
}

var logLevelStrings = map[string]slog.Level{
	"info":    slog.LevelInfo,
	"debug":   slog.LevelDebug,
	"warn":    slog.LevelWarn,
	"warning": slog.LevelWarn,
	"err":     slog.LevelError,
	"error":   slog.LevelError,
	"trace":   logging.LevelTrace,
}

type mutualExclusivityError struct {
	firstFlag   string
	currentFlag string
}

func (e mutualExclusivityError) Error() string {
	return fmt.Sprintf("The %q flag cannot be used with the %q flag", e.firstFlag, e.currentFlag)
}

func checkExclusivity(ctx *cli.Context, existingFlag, currentFlag string) error {
	if ctx.IsSet(existingFlag) {
		return mutualExclusivityError{firstFlag: existingFlag, currentFlag: currentFlag}
	}
	return nil
}

func main() {
	var cmd command
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "extension",
				Destination: &cmd.Extension,
				Category:    "Query Arguments",
				Aliases:     []string{"e"},
				Value:       "",
				Usage:       "File extension of files to query within",
				EnvVars:     []string{"GH_SEARCH_EXTENSION"},
				Action: func(ctx *cli.Context, value string) error {
					return checkExclusivity(ctx, "filename", "extension")
				},
			},
			&cli.StringFlag{
				Name:        "owner",
				Destination: &cmd.Owner,
				Category:    "Query Arguments",
				Aliases:     []string{"o"},
				Value:       "",
				Usage:       "Repo owner of files to query within",
				EnvVars:     []string{"GH_SEARCH_OWNER"},
			},
			&cli.StringFlag{
				Name:        "filename",
				Destination: &cmd.Filename,
				Category:    "Query Arguments",
				Aliases:     []string{"f"},
				Value:       "",
				Usage:       "File name of files to query within",
				EnvVars:     []string{"GH_SEARCH_FILENAME"},
				Action: func(ctx *cli.Context, value string) error {
					return checkExclusivity(ctx, "extension", "filename")
				},
			},
			&cli.StringFlag{
				Name:        "topic",
				Destination: &cmd.Topic,
				Category:    "Query Arguments",
				Aliases:     []string{"t"},
				Value:       "",
				Usage:       "Repo topic to scope queries to",
				EnvVars:     []string{"GH_SEARCH_TOPIC"},
				Action: func(ctx *cli.Context, value string) error {
					return checkExclusivity(ctx, "repo", "topic")
				},
			},
			&cli.StringFlag{
				Name:        "repo",
				Destination: &cmd.Repo,
				Category:    "Query Arguments",
				Aliases:     []string{"r"},
				Value:       "",
				Usage:       "Repository to scope queries to",
				EnvVars:     []string{"GH_SEARCH_REPO"},
				Action: func(ctx *cli.Context, value string) error {
					return checkExclusivity(ctx, "topic", "repo")
				},
			},
			&cli.StringFlag{
				Name:        "repo-query",
				Destination: &cmd.Repo,
				Category:    "Query Arguments",
				Value:       "",
				Usage:       "Query to search for within repository metadata to limit the repositories queried",
				EnvVars:     []string{"GH_SEARCH_REPO_QUERY"},
			},
			&cli.StringFlag{
				Name:        "token",
				Destination: &cmd.Token,
				Category:    "Authentication",
				Value:       "",
				Usage:       "GitHub API token to use to authorize the query (probably only pass this in as an environment variable for security reasons)",
				EnvVars:     []string{"GITHUB_TOKEN", "GH_TOKEN"},
			},
			&cli.StringFlag{
				Name:        "log-level",
				Value:       "INFO",
				DefaultText: "INFO",
				Usage:       "Logging Level [TRACE, DEBUG, INFO, WARN, ERROR]",
				EnvVars:     []string{"GH_SEARCH_LOG_LEVEL"},
				Category:    "Logging",
				Action: func(_ *cli.Context, value string) error {
					level, found := logLevelStrings[strings.ToLower(value)]
					if !found {
						return fmt.Errorf("Invalid log level: %q", value)
					}
					cmd.logLevel = level
					return nil
				},
			},
			&cli.BoolFlag{
				Name:        "log-json",
				Value:       false,
				Category:    "Logging",
				Destination: &cmd.logJson,
			},
			&cli.StringFlag{
				Name:        "format",
				Destination: &cmd.outputFormat,
				Value:       "json",
				DefaultText: "json",
				Usage:       "Output format [json, pretty]",
				EnvVars:     []string{"GH_SEARCH_FORMAT"},
				Category:    "Output Formatting",
				Action: func(_ *cli.Context, value string) error {
					if value != "json" && value != "pretty" {
						return fmt.Errorf("Invalid output format : %q", value)
					}
					return nil
				},
			},
		},
		Name:            "gh-search",
		Usage:           "Search code on GitHub",
		HideHelpCommand: true,
		Args:            true,
		ArgsUsage:       "<QUERY TEXT>",
		Action:          cmd.run,
	}

	if err := app.Run(os.Args); err != nil {
		cmd.logger.Error(err.Error())
		os.Exit(1)
	}
}

func (cmd *command) configureLogging(ctx *cli.Context) error {
	var handler slog.Handler
	if cmd.logJson {
		handler = slog.NewJSONHandler(ctx.App.ErrWriter, &slog.HandlerOptions{
			Level: cmd.logLevel,
		})
	} else {
		handler = slog.NewTextHandler(ctx.App.ErrWriter, &slog.HandlerOptions{
			Level: cmd.logLevel,
		})
	}

	cmd.logger = slog.New(handler)
	return nil
}

func (cmd *command) run(cliCtx *cli.Context) error {
	cmd.configureLogging(cliCtx)

	args := cliCtx.Args()
	if args.Len() != 1 {
		return fmt.Errorf("Exactly 1 query string must be provided for the search")
	}
	cmd.Query = args.First()

	ctx := logging.WithContext(context.Background(), cmd.logger)

	output, err := query.ExecuteQuery(ctx, cmd.RequestParams)
	if err != nil {
		return err
	}

	if cmd.outputFormat == "json" {
		enc := json.NewEncoder(cliCtx.App.Writer)
		enc.SetIndent("", "   ")
		enc.Encode(output)
	} else {
		for repo, files := range output.Repositories {
			var builder strings.Builder
			builder.WriteString(repo + ":\n")
			for _, file := range files {
				builder.WriteString("   " + file)
			}
			cliCtx.App.Writer.Write([]byte(builder.String()))
		}
	}

	return nil
}
