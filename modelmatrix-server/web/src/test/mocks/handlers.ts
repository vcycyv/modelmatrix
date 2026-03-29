import { http, HttpResponse } from 'msw';

const TEST_TOKEN = 'test-jwt-token';

const testUser = {
  username: 'michael.jordan',
  full_name: 'Michael Jordan',
  email: 'michael.jordan@example.org',
  groups: ['modelmatrix_admin'],
};

const testCollection = {
  id: 'col-1',
  name: 'Test Collection',
  description: 'A test collection',
  datasource_count: 0,
  created_by: 'michael.jordan',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

const testDatasource = {
  id: 'ds-1',
  name: 'Test Datasource',
  type: 'csv',
  collection_id: 'col-1',
  column_count: 10,
  created_by: 'michael.jordan',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

const testModel = {
  id: 'model-1',
  name: 'Test Model',
  algorithm: 'random_forest',
  model_type: 'regression',
  status: 'draft',
  build_id: 'build-1',
  datasource_id: 'ds-1',
  target_column: 'BAD',
  version: 1,
  created_by: 'michael.jordan',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

const testBuild = {
  id: 'build-1',
  name: 'Test Build',
  status: 'pending',
  model_type: 'regression',
  algorithm: 'random_forest',
  datasource_id: 'ds-1',
  created_by: 'michael.jordan',
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

function apiResponse<T>(data: T, status = 200) {
  return HttpResponse.json({ code: 200, msg: 'success', data }, { status });
}

export const handlers = [
  // Auth
  http.post('/api/auth/login', async ({ request }) => {
    const body = await request.json() as { username?: string; password?: string };
    if (body.username === 'michael.jordan' && body.password === '111222333') {
      return apiResponse({ token: TEST_TOKEN, ...testUser });
    }
    return HttpResponse.json({ code: 401, msg: 'Invalid credentials' }, { status: 401 });
  }),

  http.post('/api/auth/refresh', () => {
    return apiResponse({ token: TEST_TOKEN });
  }),

  // Collections
  http.get('/api/collections', () => {
    return apiResponse({ collections: [testCollection], total: 1 });
  }),

  http.post('/api/collections', async ({ request }) => {
    const body = await request.json() as { name?: string };
    return HttpResponse.json(
      { code: 200, msg: 'success', data: { ...testCollection, name: body.name ?? testCollection.name } },
      { status: 201 }
    );
  }),

  http.delete('/api/collections/:id', () => {
    return new HttpResponse(null, { status: 204 });
  }),

  // Datasources
  http.get('/api/datasources', () => {
    return apiResponse({ datasources: [testDatasource], total: 1 });
  }),

  http.get('/api/datasources/:id/columns', () => {
    return apiResponse([
      { id: 'col-1', name: 'BAD', data_type: 'numeric', role: 'target' },
      { id: 'col-2', name: 'LOAN', data_type: 'numeric', role: 'input' },
    ]);
  }),

  // Models
  http.get('/api/models', () => {
    return apiResponse({ models: [testModel], total: 1 });
  }),

  http.get('/api/models/:id', () => {
    return apiResponse({ ...testModel, variables: [], files: [] });
  }),

  http.post('/api/models/:id/score', async () => {
    return apiResponse({ job_id: 'job-1', status: 'accepted', message: 'Scoring started' });
  }),

  http.post('/api/models/:id/retrain', async () => {
    return HttpResponse.json(
      { code: 200, msg: 'success', data: { ...testBuild, id: 'retrain-build-1', name: 'Retrain: Test Model' } },
      { status: 202 }
    );
  }),

  // Builds
  http.get('/api/builds', () => {
    return apiResponse({ builds: [testBuild], total: 1 });
  }),

  http.post('/api/builds', async ({ request }) => {
    const body = await request.json() as { name?: string };
    return HttpResponse.json(
      { code: 200, msg: 'success', data: { ...testBuild, name: body.name ?? testBuild.name } },
      { status: 201 }
    );
  }),

  http.post('/api/builds/:id/start', () => {
    return apiResponse({ ...testBuild, status: 'running' });
  }),

  // Folders
  http.get('/api/folders', () => {
    return apiResponse([]);
  }),

  // Projects
  http.get('/api/projects', () => {
    return apiResponse([]);
  }),

  // Search
  http.get('/api/search', () => {
    return apiResponse({ query: '', total: 0, results: [] });
  }),
];
