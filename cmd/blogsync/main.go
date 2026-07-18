package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/wamphlett/blogsync/cfg"
	"github.com/wamphlett/blogsync/pkg/blog"
	"github.com/wamphlett/blogsync/pkg/gitops"
	"github.com/wamphlett/blogsync/pkg/markdown"
	"github.com/wamphlett/blogsync/pkg/metrics"
	"github.com/wamphlett/blogsync/pkg/render"
	"github.com/wamphlett/blogsync/pkg/telemetry"
)

func main() {
	config, err := cfg.New()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	var level slog.Level
	if err := level.UnmarshalText([]byte(config.Logging.Level)); err != nil {
		level = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if config.Logging.Format == "text" {
		handler = slog.NewTextHandler(os.Stdout, opts)
	} else {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	}
	logger := slog.New(telemetry.NewTraceHandler(handler)).With("service", config.Otel.ServiceName)
	slog.SetDefault(logger)
	telemetry.SetGRPCLogger()

	ctx := context.Background()

	shutdownTelemetry, err := telemetry.Setup(ctx, config.Otel.Endpoint, config.Otel.ServiceName)
	if err != nil {
		logger.Error("failed to initialise telemetry", "error", err)
		os.Exit(1)
	}

	exitCode := 0
	if err := run(ctx, config, metrics.NewClient()); err != nil {
		logger.Error("run failed", "error", err)
		exitCode = 1
	}

	// Flush explicitly (rather than via defer) since this is a one-shot CLI:
	// os.Exit would otherwise skip any deferred flush of pending spans/metrics.
	if err := shutdownTelemetry(ctx); err != nil {
		logger.Error("failed to shut down telemetry", "error", err)
	}

	os.Exit(exitCode)
}

func run(ctx context.Context, config *cfg.Config, metricsClient *metrics.Client) (err error) {
	tracer := otel.Tracer(telemetry.InstrumentationName)
	ctx, span := tracer.Start(ctx, "blogsync.Run", trace.WithAttributes(
		attribute.String("repo.url", config.Repo.URL),
		attribute.String("table.id", config.Repo.TableID),
	))
	defer span.End()

	start := time.Now()
	outcome := "error"
	defer func() { metricsClient.Run(ctx, start, outcome) }()

	logger := slog.Default()

	logger.InfoContext(ctx, "fetching articles", "endpoint", config.Blog.Endpoint)
	articles, err := blog.Fetch(ctx, tracer, config.Blog.Endpoint)
	if err != nil {
		return err
	}
	metricsClient.ArticlesVisible(ctx, len(articles))
	logger.InfoContext(ctx, "found visible articles", "count", len(articles))

	newTable, err := render.Table(config.Repo.TableID, config.Blog.BaseURL, articles)
	if err != nil {
		return err
	}

	repoURL, err := gitops.AuthenticatedRepoURL(config.Repo.URL, config.Git.Token)
	if err != nil {
		return err
	}

	tmpDir, err := os.MkdirTemp("", "blogsync-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	logger.InfoContext(ctx, "cloning repo", "repo", config.Repo.URL, "branch", config.Repo.Branch)
	if err := gitops.Clone(ctx, tracer, repoURL, config.Repo.Branch, tmpDir); err != nil {
		return err
	}

	mdPath := filepath.Join(tmpDir, config.Repo.MarkdownFilePath)
	original, err := os.ReadFile(mdPath)
	if err != nil {
		return err
	}

	updated, err := markdown.ReplaceTable(string(original), config.Repo.TableID, newTable)
	if err != nil {
		return err
	}

	if updated == string(original) {
		outcome = "unchanged"
		logger.InfoContext(ctx, "no changes detected, exiting")
		return nil
	}

	if err := os.WriteFile(mdPath, []byte(updated), 0o644); err != nil {
		return err
	}

	logger.InfoContext(ctx, "changes detected, committing and pushing", "remote", config.Git.Remote, "branch", config.Repo.Branch)
	if err := gitops.CommitAndPush(
		ctx, tracer,
		tmpDir,
		config.Repo.MarkdownFilePath,
		config.Git.Remote,
		config.Repo.Branch,
		config.Git.CommitUserName,
		config.Git.CommitUserEmail,
		config.Git.CommitMessage,
	); err != nil {
		return err
	}

	metricsClient.TableUpdated(ctx)
	outcome = "updated"
	logger.InfoContext(ctx, "done")
	return nil
}
