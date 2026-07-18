package metrics

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"

	"github.com/wamphlett/blogsync/pkg/telemetry"
)

// Client holds all OTEL metric instruments for blogsync.
type Client struct {
	runDuration     otelmetric.Float64Histogram
	articlesVisible otelmetric.Int64Gauge
	tableUpdates    otelmetric.Int64Counter
}

func NewClient() *Client {
	meter := otel.Meter(telemetry.InstrumentationName)
	return &Client{
		runDuration: must(meter.Float64Histogram(
			"blogsync.run.duration",
			otelmetric.WithUnit("s"),
			otelmetric.WithDescription("Duration of a full blogsync run, by outcome"),
		)),
		articlesVisible: must(meter.Int64Gauge(
			"blogsync.articles.visible",
			otelmetric.WithUnit("{article}"),
			otelmetric.WithDescription("Number of non-hidden articles returned by the blog endpoint on the last run"),
		)),
		tableUpdates: must(meter.Int64Counter(
			"blogsync.table.updates",
			otelmetric.WithUnit("{update}"),
			otelmetric.WithDescription("Total number of runs that committed and pushed a changed table"),
		)),
	}
}

// Run records the duration of a full run, tagged with its outcome:
// "updated", "unchanged", or "error".
func (c *Client) Run(ctx context.Context, start time.Time, outcome string) {
	c.runDuration.Record(ctx, time.Since(start).Seconds(), otelmetric.WithAttributes(attribute.String("outcome", outcome)))
}

func (c *Client) ArticlesVisible(ctx context.Context, n int) {
	c.articlesVisible.Record(ctx, int64(n))
}

func (c *Client) TableUpdated(ctx context.Context) {
	c.tableUpdates.Add(ctx, 1)
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
