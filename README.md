# gh-search
GitHub Code Search CLI

## Usage

```
NAME:
   gh-search - Search code on GitHub

USAGE:
   gh-search [global options] <QUERY TEXT>

GLOBAL OPTIONS:
   --help, -h  show help

   Authentication

   --token value  GitHub API token to use to authorize the query (probably only pass this in as an environment variable for security reasons) [$GITHUB_TOKEN, $GH_TOKEN]

   Logging

   --log-json         (default: false)
   --log-level value  Logging Level [TRACE, DEBUG, INFO, WARN, ERROR] (default: INFO) [$GH_SEARCH_LOG_LEVEL]

   Output Formatting

   --format value  Output format [json, pretty] (default: json) [$GH_SEARCH_FORMAT]

   Query Arguments

   --extension value, -e value  File extension of files to query within [$GH_SEARCH_EXTENSION]
   --filename value, -f value   File name of files to query within [$GH_SEARCH_FILENAME]
   --owner value, -o value      Repo owner of files to query within [$GH_SEARCH_OWNER]
   --repo value, -r value       Repository to scope queries to [$GH_SEARCH_REPO]
   --repo-query value           Query to search for within repository metadata to limit the repositories queried [$GH_SEARCH_REPO_QUERY]
   --topic value, -t value      Repo topic to scope queries to [$GH_SEARCH_TOPIC]
```

## Why not just use `gh`

1. Rate Limiting - The `gh` cli doesn't take rate limiting into account and so attempting to use it for scripting gets very difficult with needing to retry requests after waiting the indicated amount of time.
2. Auto-Pagination - The `gh` cli has some support for pagination but without also supporting rate limiting you very quickly run into problems.
3. You want to search for code in repos that have specific topics or meet other criteria that you could pass to the repository search.
