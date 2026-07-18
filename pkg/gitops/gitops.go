package gitops

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// AuthenticatedRepoURL injects an HTTPS auth token into repoURL when one is
// provided and the URL doesn't already carry credentials. SSH remotes are
// left untouched (auth is expected to come from a mounted key in that case).
func AuthenticatedRepoURL(repoURL, token string) (string, error) {
	if token == "" {
		return repoURL, nil
	}

	u, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("parsing repo URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return repoURL, nil
	}
	if u.User != nil {
		return repoURL, nil
	}

	u.User = url.UserPassword("x-access-token", token)
	return u.String(), nil
}

// runGit runs a git command in dir, returning combined output on failure.
func runGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// Clone shallow-clones repoURL at branch into dir.
func Clone(ctx context.Context, tracer trace.Tracer, repoURL, branch, dir string) (err error) {
	ctx, span := tracer.Start(ctx, "blogsync.CloneRepo", trace.WithAttributes(attribute.String("repo.branch", branch)))
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		}
	}()

	_, err = runGit(ctx, "", "clone", "--depth", "1", "--branch", branch, repoURL, dir)
	return err
}

// CommitAndPush stages filePath, commits with the given author/message, and
// pushes to remote/branch. Assumes the working tree already has changes.
func CommitAndPush(ctx context.Context, tracer trace.Tracer, dir, filePath, remote, branch, userName, userEmail, message string) (err error) {
	ctx, span := tracer.Start(ctx, "blogsync.CommitAndPush", trace.WithAttributes(
		attribute.String("repo.remote", remote),
		attribute.String("repo.branch", branch),
	))
	defer span.End()
	defer func() {
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		}
	}()

	if _, err = runGit(ctx, dir, "add", "--", filePath); err != nil {
		return err
	}
	if _, err = runGit(ctx, dir,
		"-c", "user.name="+userName,
		"-c", "user.email="+userEmail,
		"commit", "-m", message,
	); err != nil {
		return err
	}
	if _, err = runGit(ctx, dir, "push", remote, branch); err != nil {
		return err
	}
	return nil
}
