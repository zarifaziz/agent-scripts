# Build & Debug Guide

## Running the Service

- Launch with log filtering via `./run` or `DEBUG_PREFIX=<tag> ./run`(see when to use DEBUG_PREFIX below)
- Rerunning the server handles killing+restarting the process.

## Targeted Logging

- Attach a structured log with a `prefix` field when debugging/testing specific flows/group of logs
  ```go
  log.From(ctx).Info("worksheet skeleton built",
      zap.String("prefix", "spec"),
      zap.Any("data", skeleton),
  )
  ```
- Start the service with the matching `DEBUG_PREFIX` value (`DEBUG_PREFIX=spec ./run`) to stream only events where `key=="prefix"` and `value=="spec"`. Pick any tag that suits the scenario; `spec` above is just an illustration.

## Worksheet Plan Scripts

- The shell harnesses under `testing/worksheet_plan/` drive end-to-end flows. Common entry points:
  - `./create_worksheet_plan_test.sh` – smoke test that creates a plan and waits until it’s ready.
  - `./test_update_worksheet_plan.sh`, `./verify_v2_spec.sh`, and `./test_all_cases.sh` for broader regression coverage.
- Each script assumes the local server is running and uses helpers from the same folder; check `testing/worksheet_plan/README.md` for payload examples (`--skill`, `--node-count`, difficulty flags) and environment prerequisites.

## Database Inspection

- Run ad-hoc queries with `cypher-safe --preset resources-dev`. Invoke `cypher-safe` skill for detailed instructions.
- When a Cypher query of roughly 10+ lines fails, dump the full statement plus parameters to the log and invoke `db-oracle` skill for troubleshooting before retrying.
