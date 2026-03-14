package search

import (
	"strings"

	"github.com/rafamoreira/trove/internal/diag"
	"github.com/rafamoreira/trove/internal/vault"
)

func SearchVault(v *vault.Vault, query string, opts vault.ListOptions) ([]vault.SearchResult, []diag.Warning, error) {
	items, warnings, err := v.List(opts)
	if err != nil {
		return nil, nil, err
	}

	needle := strings.ToLower(strings.TrimSpace(query))
	if needle == "" {
		return nil, warnings, nil
	}

	results := make([]vault.SearchResult, 0)
	for _, item := range items {
		body, err := item.Body()
		if err != nil {
			return nil, warnings, err
		}

		matches := make([]vault.SearchMatch, 0)
		lines := strings.Split(body, "\n")
		for idx, line := range lines {
			if strings.Contains(strings.ToLower(line), needle) {
				matches = append(matches, vault.SearchMatch{
					Line:    idx + 1,
					Context: line,
				})
			}
		}

		if len(matches) == 0 {
			if strings.Contains(strings.ToLower(item.Description), needle) || strings.Contains(strings.ToLower(item.ID), needle) {
				matches = append(matches, vault.SearchMatch{
					Line:    0,
					Context: item.Description,
				})
			} else {
				for _, tag := range item.Tags {
					if strings.Contains(strings.ToLower(tag), needle) {
						matches = append(matches, vault.SearchMatch{
							Line:    0,
							Context: tag,
						})
						break
					}
				}
			}
		}

		if len(matches) > 0 {
			results = append(results, vault.SearchResult{
				Snippet: item,
				Matches: matches,
			})
		}
	}

	return results, warnings, nil
}
