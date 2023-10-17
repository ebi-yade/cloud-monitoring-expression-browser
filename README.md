# cloud-monitoring-expression-browser
An easy web UI for building PromQL queries for Google Cloud Monitoring (formerly Stackdriver Monitoring)

## Usage

```shell
#!/usr/bin/env bash

export GOOGLE_PROJECT_ID=your-project-id
export METRIC_PREFIXES=run.googleapis.com # For details: https://github.com/prometheus-community/stackdriver_exporter#flags
docker compose up -d
```