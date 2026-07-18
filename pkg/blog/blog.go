package blog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Article mirrors the JSON shape returned by the blog endpoint.
type Article struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Image       string            `json:"image"`
	URL         string            `json:"url"`
	Priority    int               `json:"priority"`
	Slug        string            `json:"slug"`
	PublishedAt int64             `json:"publishedAt"`
	UpdatedAt   int64             `json:"updatedAt"`
	Hidden      bool              `json:"hidden"`
	Metadata    map[string]string `json:"metadata"`
	TopicSlug   string            `json:"topicSlug"`
}

type response struct {
	Articles []Article `json:"articles"`
}

// Fetch fetches and decodes the blog endpoint, returning only non-hidden
// articles sorted by priority (descending), falling back to publishedAt
// (descending) to break ties.
func Fetch(ctx context.Context, tracer trace.Tracer, endpoint string) (_ []Article, err error) {
	ctx, span := tracer.Start(ctx, "blogsync.FetchArticles")
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		}
	}()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("building blog endpoint request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching blog endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("blog endpoint returned status %d", resp.StatusCode)
	}

	var parsed response
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil, fmt.Errorf("decoding blog endpoint response: %w", err)
	}

	visible := make([]Article, 0, len(parsed.Articles))
	for _, a := range parsed.Articles {
		if !a.Hidden {
			visible = append(visible, a)
		}
	}

	sort.SliceStable(visible, func(i, j int) bool {
		if visible[i].Priority != visible[j].Priority {
			return visible[i].Priority > visible[j].Priority
		}
		return visible[i].PublishedAt > visible[j].PublishedAt
	})

	span.SetAttributes(
		attribute.Int("articles.total", len(parsed.Articles)),
		attribute.Int("articles.visible", len(visible)),
	)

	return visible, nil
}
