# Integration test behavior coverage

Generated for tracking API-level coverage. **Base path:** `/api` (see `cmd/api/main.go`). Integration suites live under `tests/integration/`.

**Legend**

- **Risk:** rough business/operational impact if broken (not a security audit).
- **Unit tests:** presence of focused `_test.go` coverage under `internal/module/**` (controllers, application, domain).
- **Integration:** HTTP tests against a real router + PostgreSQL (+ LDAP, MinIO); some flows use an in-process **noop compute** client unless `TEST_COMPUTE_URL` / `TEST_INTEGRATION_COMPUTE_CONTAINER` is set.
- **Status:** `covered` | `partial` | `gap` | `n/a` (intentionally out of scope).

---

| Feature | Endpoint / Entry | Risk | Unit Tests | Integration Test | Status | Notes |
|--------|-------------------|------|------------|------------------|--------|--------|
| **Auth** |
| LDAP login | `POST /api/auth/login` | High | yes | yes | covered | Happy path, wrong password, unknown user, missing fields (`auth_test.go`) |
| JWT refresh | `POST /api/auth/refresh` | Medium | yes | yes | covered | With token; 401 without token |
| **Platform** |
| Health check | `GET /api/health` | Medium | no | yes | covered | Same handler as prod (`internal/httpserver/health.go`); `health_test.go` |
| Swagger UI | `GET /swagger/*` | Low | n/a | no | n/a | Usually not exercised in integration |
| **Collections** |
| Create collection | `POST /api/collections` | Medium | yes | yes | covered | `collections_test.go`; unauthorized |
| List collections | `GET /api/collections` | Low | yes | yes | covered | Pagination; also DB-seeded list in `fixtures_integration_test.go` |
| Get collection | `GET /api/collections/:id` | Low | yes | yes | covered | Create-then-get; fixture builder path |
| Update collection | `PUT /api/collections/:id` | Medium | yes | yes | covered | |
| Delete collection | `DELETE /api/collections/:id` | High | yes | yes | covered | 204 + follow-up GET 404 |
| **Datasources** |
| Create datasource (CSV upload) | `POST /api/datasources` | High | yes | yes | covered | Multipart, column detection; validation + auth negative tests (`datasources_test.go`) |
| Create datasource (PostgreSQL) | `POST /api/datasources` | High | yes | yes | covered | Requires reachable DB/table as in test (`datasets` / `iris`); missing `connection_config` 400 |
| List datasources | `GET /api/datasources` | Low | yes | yes | covered | Collection-scoped list in `TestDatasourceHTTPLifecycle` |
| Get datasource | `GET /api/datasources/:id` | Low | yes | yes | covered | CSV create→get in lifecycle test; fixtures path still covers DB-seeded |
| Update datasource | `PUT /api/datasources/:id` | Medium | yes | yes | covered | Lifecycle test |
| Delete datasource | `DELETE /api/datasources/:id` | High | yes | yes | covered | Lifecycle test; admin |
| List columns | `GET /api/datasources/:id/columns` | Low | yes | partial | partial | Inline in GET detail; `setup.go` for builds |
| Bulk update column roles | `PUT /api/datasources/:id/columns/roles` | Medium | yes | partial | partial | `setup.go` for builds |
| Create columns | `POST /api/datasources/:id/columns` | Medium | yes | no | gap | |
| Update column role | `PUT /api/datasources/:id/columns/:column_id/role` | Medium | yes | yes | covered | Lifecycle test |
| Data preview | `GET /api/datasources/:id/preview` | Low | yes | yes | covered | Lifecycle test |
| **Folders** |
| List root folders | `GET /api/folders` | Low | yes | yes | covered | |
| Create folder / subfolder | `POST /api/folders` | Medium | yes | yes | covered | Missing name 400; `parent_id` nesting |
| Get folder | `GET /api/folders/:id` | Low | yes | yes | covered | `TestGetFolder_OK` |
| Update folder | `PUT /api/folders/:id` | Medium | yes | yes | covered | |
| Delete folder | `DELETE /api/folders/:id` | High | yes | yes | covered | |
| Folder contents count | `GET /api/folders/:id/contents-count` | Low | yes | yes | covered | |
| Folder children | `GET /api/folders/:id/children` | Low | yes | yes | covered | |
| Projects in folder | `GET /api/folders/:id/projects` | Low | yes | yes | covered | |
| Builds in folder | `GET /api/folders/:id/builds` | Low | yes | no | gap | |
| Models in folder | `GET /api/folders/:id/models` | Low | yes | no | gap | |
| Add build to folder | `POST /api/folders/:id/builds` | Medium | yes | no | gap | |
| **Projects** |
| List projects | `GET /api/projects` | Low | yes | yes | covered | Root projects only (handler); `TestListAndGetProject` |
| Create project (root / in folder) | `POST /api/projects` | Medium | yes | yes | covered | `folder_id` variant |
| Get project | `GET /api/projects/:id` | Low | yes | yes | covered | `TestListAndGetProject` |
| Update project | `PUT /api/projects/:id` | Medium | yes | yes | covered | |
| Delete project | `DELETE /api/projects/:id` | High | yes | yes | covered | |
| Builds in project | `GET /api/projects/:id/builds` | Low | yes | no | gap | |
| Models in project | `GET /api/projects/:id/models` | Low | yes | no | gap | |
| Add build to project | `POST /api/projects/:id/builds` | Medium | yes | no | gap | |
| **Search** |
| Global search | `GET /api/search` | Low | yes | yes | covered | `q`, `type`, empty query tolerance, no hits, unauthorized (`search_test.go`) |
| **Builds** |
| List builds | `GET /api/builds` | Low | yes | yes | covered | `TestListBuilds` |
| Create build | `POST /api/builds` | High | yes | yes | covered | Missing name 400; bad datasource 404 (`builds_test.go`) |
| Get build | `GET /api/builds/:id` | Low | yes | yes | covered | Unknown id 404 |
| Update build | `PUT /api/builds/:id` | Medium | yes | yes | covered | |
| Delete build | `DELETE /api/builds/:id` | High | yes | yes | covered | |
| Start build | `POST /api/builds/:id/start` | High | yes | yes | covered | Mock/noop compute + callback; optional **real** compute when `TEST_COMPUTE_URL` set (`TestStartBuild_RealCompute`) |
| Cancel build | `POST /api/builds/:id/cancel` | Medium | yes | yes | covered | Happy path; cancel after complete rejected |
| Training callback | `POST /api/builds/callback` | Critical | yes | yes | covered | Unauthenticated callback; completed + failed job paths |
| **Models** |
| List models | `GET /api/models` | Low | yes | yes | covered | Pagination (`models_test.go`) |
| Get model | `GET /api/models/:id` | Low | yes | yes | covered | Unknown id 404 |
| Update model | `PUT /api/models/:id` | Medium | yes | yes | covered | |
| Delete model | `DELETE /api/models/:id` | High | yes | yes | covered | 409 when active |
| Activate / deactivate | `POST /api/models/:id/activate`, `.../deactivate` | High | yes | yes | covered | Double-activate → 400 |
| Score model (async) | `POST /api/models/:id/score` | High | yes | no | gap | Covered in unit/controller tests; not integration |
| Score callback | `POST /api/models/:id/score/callback` | High | yes | no | gap | |
| Retrain | `POST /api/models/:id/retrain` | High | yes | yes | covered | 202; unknown model 404 (`model_versions_test.go`) |
| Model file content | `GET /api/models/:id/files/:fileId/content` | Medium | yes | no | gap | |
| **Model versions** |
| Create snapshot | `POST /api/models/:id/versions` | Medium | yes | yes | covered | `TestCreateVersion_WithoutUploadedArtifactsFails` — mock callback paths lack real objects; expect **500** with `version file` in msg (explicit) |
| List versions | `GET /api/models/:id/versions` | Low | yes | yes | covered | |
| Get version | `GET /api/models/:id/versions/:versionId` | Low | yes | partial | partial | Not-found only (`TestGetVersionNotFound`); 200 needs stored version + files |
| Restore version | `POST /api/models/:id/versions/:versionId/restore` | Medium | yes | no | gap | Needs successful snapshot first |
| **Performance monitoring** |
| Performance summary | `GET /api/models/:id/performance` | Low | yes | yes | covered | `performance_test.go` |
| Create/list baselines | `POST` / `GET .../performance/baselines` | Medium | yes | yes | covered | |
| Record performance + history | `POST .../record`, `GET .../history` | Medium | yes | yes | covered | |
| Alerts | `GET .../performance/alerts` | Low | yes | yes | covered | |
| Update alert | `PUT .../performance/alerts/:alertId` | Medium | yes | no | gap | |
| Thresholds | `GET .../performance/thresholds` | Low | yes | yes | covered | |
| Update thresholds | `PUT .../performance/thresholds` | Medium | yes | no | gap | |
| Start evaluation | `POST .../performance/evaluate` | High | yes | yes | covered | Asserts 201 or 202 |
| List evaluations | `GET .../performance/evaluations` | Low | yes | yes | covered | |
| Get evaluation by id | `GET .../performance/evaluations/:evaluationId` | Low | yes | yes | covered | `TestGetEvaluationByID` |
| Metric time series | `GET .../performance/metrics/:metricName/series` | Low | yes | yes | covered | `TestGetMetricTimeSeries` |
| Evaluation callback | `POST /api/models/:id/performance/evaluations/:evaluationId/callback` | High | yes | no | gap | |
| Global threshold defaults (read) | `GET /api/performance/threshold-defaults` | Low | yes | yes | covered | `task_type` query |
| Global threshold defaults (admin write) | `PUT /api/performance/threshold-defaults` | Medium | yes | no | gap | Admin-only |

---

## How to extend this doc

When you add or change an integration test, update the row’s **Status** and **Notes**. When a feature gains only unit coverage, keep **Integration Test** as `no` until an HTTP test exists.
