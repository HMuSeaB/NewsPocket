package parser

import (
	"testing"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/config"
	"github.com/HMuSeaB/NewsPocket/internal/fetcher"
)

func TestParseAllFiltersOldJSONAPIItemsWhenTimeExists(t *testing.T) {
	t.Parallel()

	p := New(200, 24)
	items := p.ParseAll([]fetcher.FetchResult{
		{
			Source: config.Source{Name: "json-source", Category: "test"},
			Entries: []fetcher.Entry{
				{
					Title:      "stale",
					SourceType: "json_api",
					Published:  time.Now().UTC().Add(-48 * time.Hour),
				},
			},
		},
	}, 10)

	if len(items) != 0 {
		t.Fatalf("expected stale JSON API item to be filtered, got %d items", len(items))
	}
}

func TestParseAllKeepsJSONAPIItemsWithoutTime(t *testing.T) {
	t.Parallel()

	p := New(200, 24)
	items := p.ParseAll([]fetcher.FetchResult{
		{
			Source: config.Source{Name: "json-source", Category: "test"},
			Entries: []fetcher.Entry{
				{
					Title:      "timeless",
					SourceType: "json_api",
				},
			},
		},
	}, 10)

	if len(items) != 1 {
		t.Fatalf("expected timeless JSON API item to remain included, got %d items", len(items))
	}
}
