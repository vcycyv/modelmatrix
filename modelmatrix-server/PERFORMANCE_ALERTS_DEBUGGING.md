# Debugging: No Performance Alerts After Build + Evaluation

If you rebuilt the model and ran a performance evaluation but still see **0 Active Alerts**, use this guide to find where the pipeline stops.

## 1. Check server logs (most important)

Run the API server with logs visible (e.g. `go run cmd/api/main.go` or your usual start). Then:

### After the build completes (training callback)

Look for one of:

- **Success:** `Created initial performance baseline for model <uuid> (4 metrics)`  
  → Baseline was created from training metrics.
- **Skip:** `Build callback: performanceService is nil, skipping auto-baseline for model <uuid>`  
  → Server was built/run without performance service wired to build service (e.g. old binary). Rebuild and restart.
- **Skip:** `Build callback: no metrics in callback, skipping auto-baseline for model <uuid>`  
  → Compute service did not send `metrics` in the build callback. Check modelmatrix-compute training callback payload.
- **Skip:** `Build callback: no numeric metrics after conversion ... skipping auto-baseline`  
  → Metrics were sent but not as numbers (e.g. wrong type). Check callback JSON.
- **Error:** `Failed to create initial performance baseline for model <uuid>: ...`  
  → CreateBaseline failed (e.g. DB, validation). Fix the reported error.

### After the performance evaluation completes (evaluation callback)

Look for:

- **RecordPerformance:** `RecordPerformance: model=<uuid> baselines=N thresholds=M records=K (with drift: D)`
  - If **baselines=0** → This model has no baselines. Either auto-baseline was skipped (see above) or the model was created before auto-baseline existed. Fix by setting a baseline manually (Set Baseline in UI) or by ensuring build callback creates it.
  - If **thresholds=0** → No thresholds for this model. Default thresholds are created when you create a baseline. If you have baselines but 0 thresholds, check for log: `Failed to initialize default thresholds when recording`.
  - If **with drift: 0** → Drift is not computed. That usually means no baseline for the metrics being recorded (baseline metric names must match evaluation metric names, e.g. `f1_score`, `accuracy`).
- **Alert created:** `Created warning alert for model <uuid>: ...` or `Created critical alert for model <uuid>: ...`  
  → Alert was created; it should appear in the UI (refresh or re-open the Performance tab).

If you see `checkAndCreateAlert: skip metric X (no drift or baseline)` then that metric has no baseline or no drift. If you see `checkAndCreateAlert: metric X drift Y% below threshold` then drift was below the threshold (no alert).

## 2. Verify via API (optional)

Use the model ID of the **new** model (from the build you just ran).

- **Baselines:**  
  `GET /api/models/{model_id}/performance/baselines`  
  Expect a non-empty list with entries for e.g. `accuracy`, `f1_score`, `precision`, `recall`.

- **Thresholds:**  
  `GET /api/models/{model_id}/performance/thresholds`  
  Expect entries for the same metric names (warning/critical %).

- **Alerts:**  
  `GET /api/models/{model_id}/performance/alerts`  
  After an evaluation that should breach, you should see active alerts here.

## 3. Quick checklist

| Step | What to check |
|------|----------------|
| Build completed | Training callback reached server; no error in build callback handler. |
| Auto-baseline | Log shows "Created initial performance baseline" for this model. |
| Baselines exist | GET baselines returns ≥1 entry for this model. |
| Run evaluation | You clicked "Run Evaluation" and it completed (evaluation status "Completed"). |
| Drift computed | Log shows "with drift: &gt; 0" in RecordPerformance. |
| Thresholds exist | GET thresholds returns entries; or log shows thresholds &gt; 0 in RecordPerformance. |
| Breach | Current metric differs from baseline by more than threshold (e.g. F1 drop &gt; 10% for warning). |

## 4. What to report back

To narrow it down, please share:

1. **Exact log lines** from the API server from:
   - Right after the build completes (training callback).
   - Right after the performance evaluation completes (evaluation callback), including the `RecordPerformance: model=... baselines=... thresholds=... records=... (with drift: ...)` line.
2. **GET baselines** response for the new model ID (redact IDs if needed): empty or list of metrics?
3. **GET thresholds** response for the new model ID: empty or list?
4. Whether you restarted the API server after pulling the latest code (so that auto-baseline and performance wiring are active).

With that, we can see whether the issue is: no baseline, no thresholds, no drift, or drift not breaching.
