# blogsync

Fetches a list of articles from a blog JSON endpoint, renders them into an HTML
`<table>`, and syncs that table into a markdown file (e.g. a GitHub profile
`README.md`) by replacing the existing `<table id="...">...</table>` block.
If the rendered table is identical to what's already committed, it exits
without touching git. Intended to run on a schedule as a Kubernetes CronJob.

## Configuration

All configuration is via environment variables.

| Var | Required | Default | Purpose |
|---|---|---|---|
| `BLOG_ENDPOINT` | yes | — | JSON articles endpoint to fetch |
| `BLOG_BASE_URL` | yes | — | Base URL used to build article links: `{base}/{topicSlug}/{slug}` |
| `REPO_URL` | yes | — | Git clone URL, e.g. `https://github.com/<owner>/<repo>.git` |
| `MARKDOWN_FILE_PATH` | yes | — | Path within the repo to the markdown file to update |
| `TABLE_ID` | yes | — | The `id` attribute of the table to find and replace |
| `REPO_BRANCH` | no | `main` | Branch to clone and push |
| `GIT_TOKEN` | no | — | If set, injected as HTTPS auth into `REPO_URL` for clone and push |
| `GIT_REMOTE` | no | `origin` | Remote name to push to |
| `GIT_COMMIT_USER_NAME` | no | `blogsync-bot` | Commit author name |
| `GIT_COMMIT_USER_EMAIL` | no | `blogsync-bot@users.noreply.github.com` | Commit author email |
| `GIT_COMMIT_MESSAGE` | no | `chore: sync blog table` | Commit message |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | no | — | OTLP gRPC endpoint for traces/metrics, e.g. `http://otel-collector:4317`. Must include a scheme. Leave unset to disable telemetry entirely |
| `OTEL_SERVICE_NAME` | no | `blogsync` | `service.name` resource attribute on all spans/metrics |
| `LOG_LEVEL` | no | `info` | `debug`, `info`, `warn`, or `error` |
| `LOG_FORMAT` | no | `json` | `json` or `text` |

The article JSON is expected in this shape:

```json
{
  "articles": [
    {
      "title": "...",
      "description": "...",
      "image": "https://.../image.jpg",
      "priority": 20,
      "slug": "my-article",
      "publishedAt": 1784073600,
      "hidden": false,
      "topicSlug": "japan"
    }
  ]
}
```

Articles with `hidden: true` are skipped. The rest are sorted by `priority`
descending (ties broken by `publishedAt` descending) and laid out 3 per row.

## Observability

Logs, traces, and metrics follow standard OpenTelemetry conventions so they
show up in Grafana without any extra setup:

- **Logs** are structured JSON to stdout (set `LOG_FORMAT=text` for
  human-readable output locally). Every log line emitted during a run carries
  `trace_id` and `span_id`, so once Loki ingests them you can add a derived
  field on `trace_id` pointing at Tempo for logs → traces click-through.
- **Traces** are pushed via OTLP/gRPC to `OTEL_EXPORTER_OTLP_ENDPOINT` (unset
  = disabled, no-op tracer). Each run produces one trace rooted at
  `blogsync.Run`, with child spans:

  | Span | Attributes |
  |---|---|
  | `blogsync.FetchArticles` | `articles.total`, `articles.visible` |
  | `blogsync.CloneRepo` | `repo.branch` |
  | `blogsync.CommitAndPush` | `repo.remote`, `repo.branch` (only present when the table actually changed) |

  A failed run marks the relevant span `status=ERROR` with the error message.
- **Metrics** are pushed via OTLP/gRPC to the same endpoint:

  | Metric | Type | Description |
  |---|---|---|
  | `blogsync.run.duration` | histogram (seconds) | Full run duration, tagged `outcome=updated\|unchanged\|error` |
  | `blogsync.articles.visible` | gauge | Non-hidden articles returned by the endpoint on the last run |
  | `blogsync.table.updates` | counter | Incremented each time a run commits and pushes a changed table |

  Since this runs as a short-lived CronJob rather than a long-running server,
  metrics aren't scraped/pushed on an interval — they're flushed once, synchronously,
  right before the process exits.

Because a CronJob runs infrequently, don't expect dense time-series panels;
the more useful Grafana views are a Tempo trace list for `blogsync.Run` (to
see what a given run actually did and how long each stage took) and a table
of recent `blogsync.run.duration` data points broken down by `outcome`.

## GitHub setup

The program needs push access to the repo containing the markdown file. The
simplest option is a fine-grained personal access token scoped to just that
repo:

1. Go to **GitHub → Settings → Developer settings → Personal access tokens →
   Fine-grained tokens → Generate new token**.
2. Set **Resource owner** to the account/org that owns the target repo, and
   under **Repository access** select **Only select repositories**, then pick
   the repo you want the table synced into (e.g. your profile README repo).
3. Under **Permissions → Repository permissions**, set **Contents** to
   **Read and write**. No other permissions are needed.
4. Generate the token and copy it — you won't be able to see it again.
5. Store it as a Kubernetes secret (see below) and reference it via the
   `GIT_TOKEN` env var. The program injects it into `REPO_URL` automatically
   as `https://x-access-token:<token>@github.com/...`, so `REPO_URL` itself
   just needs to be the plain HTTPS clone URL.

If you'd rather use an SSH deploy key instead of a token: add a deploy key
with write access under the target repo's **Settings → Deploy keys**, mount
the private key into the container (e.g. at `/root/.ssh/id_ed25519` with a
matching `known_hosts`), set `REPO_URL` to the `git@github.com:...` SSH form,
and leave `GIT_TOKEN` unset.

## Running locally

```sh
go build -o blogsync ./cmd/blogsync

BLOG_ENDPOINT="https://example.com/api/articles" \
BLOG_BASE_URL="https://blog.example.com" \
REPO_URL="https://github.com/<owner>/<repo>.git" \
GIT_TOKEN="<token>" \
MARKDOWN_FILE_PATH="README.md" \
TABLE_ID="blogsync" \
./blogsync
```

## Docker

Build the image:

```sh
docker build -t blogsync .
```

Run it:

```sh
docker run --rm \
  -e BLOG_ENDPOINT="https://example.com/api/articles" \
  -e BLOG_BASE_URL="https://blog.example.com" \
  -e REPO_URL="https://github.com/<owner>/<repo>.git" \
  -e GIT_TOKEN="<token>" \
  -e MARKDOWN_FILE_PATH="README.md" \
  -e TABLE_ID="blogsync" \
  -e OTEL_EXPORTER_OTLP_ENDPOINT="http://otel-collector:4317" \
  blogsync
```

## Deploying as a Kubernetes CronJob

Create the secret holding the token:

```sh
kubectl create secret generic blogsync-git-token \
  --from-literal=token='<your-token>'
```

Example CronJob manifest:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: blogsync
spec:
  schedule: "0 6 * * *"
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: Never
          containers:
            - name: blogsync
              image: <your-registry>/blogsync:latest
              env:
                - name: BLOG_ENDPOINT
                  value: "https://example.com/api/articles"
                - name: BLOG_BASE_URL
                  value: "https://blog.example.com"
                - name: REPO_URL
                  value: "https://github.com/<owner>/<repo>.git"
                - name: MARKDOWN_FILE_PATH
                  value: "README.md"
                - name: TABLE_ID
                  value: "blogsync"
                - name: OTEL_EXPORTER_OTLP_ENDPOINT
                  value: "http://otel-collector.observability:4317"
                - name: GIT_TOKEN
                  valueFrom:
                    secretKeyRef:
                      name: blogsync-git-token
                      key: token
```
