// API client for ModelMatrix backend

const API_BASE = '/api';

// Token management
let authToken: string | null = localStorage.getItem('token');

export function setToken(token: string | null) {
  authToken = token;
  if (token) {
    localStorage.setItem('token', token);
  } else {
    localStorage.removeItem('token');
  }
}

export function getToken(): string | null {
  return authToken;
}

export function isAuthenticated(): boolean {
  return !!authToken;
}

// API request helper
async function request<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...options.headers,
  };

  if (authToken) {
    (headers as Record<string, string>)['Authorization'] = `Bearer ${authToken}`;
  }

  const response = await fetch(`${API_BASE}${endpoint}`, {
    ...options,
    headers,
  });

  if (response.status === 401) {
    setToken(null);
    window.location.href = '/login';
    throw new Error('Unauthorized');
  }

  // Handle empty responses (e.g., 204 No Content from DELETE)
  const contentType = response.headers.get('content-type');
  const hasJsonContent = contentType && contentType.includes('application/json');
  const text = await response.text();
  
  let data: Record<string, unknown> | null = null;
  if (text && hasJsonContent) {
    try {
      data = JSON.parse(text);
    } catch {
      // If JSON parsing fails, treat as empty response
      data = null;
    }
  }

  if (!response.ok) {
    const errorMessage = data?.error || data?.message || data?.msg || 'Request failed';
    throw new Error(String(errorMessage));
  }

  // Return empty object for void responses
  if (!data) {
    return {} as T;
  }

  return (data.data ?? data) as T;
}

// Login response from backend (flat structure)
interface LoginResponse {
  token: string;
  username: string;
  full_name: string;
  email: string;
  groups: string[];
}

// Auth API
export const authApi = {
  login: async (username: string, password: string) => {
    const response = await request<LoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    });
    setToken(response.token);
    // Transform backend response to User format
    const user: User = {
      username: response.username,
      display_name: response.full_name,
      email: response.email,
      groups: response.groups,
    };
    return { token: response.token, user };
  },

  logout: () => {
    setToken(null);
  },

  refresh: async () => {
    return request<{ token: string }>('/auth/refresh', { method: 'POST' });
  },
};

// Folder API
export interface FolderContentsCount {
  subfolder_count: number;
  project_count: number;
  model_count: number;
  build_count: number;
}

export const folderApi = {
  getRootFolders: () => request<Folder[]>('/folders'),
  getFolder: (id: string) => request<Folder>(`/folders/${id}`),
  getChildren: (id: string) => request<Folder[]>(`/folders/${id}/children`),
  create: (data: CreateFolderRequest) => 
    request<Folder>('/folders', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: UpdateFolderRequest) =>
    request<Folder>(`/folders/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string, force?: boolean) => 
    request<void>(`/folders/${id}${force ? '?force=true' : ''}`, { method: 'DELETE' }),
  getContentsCount: (id: string) => request<FolderContentsCount>(`/folders/${id}/contents-count`),
  getBuilds: (id: string) => request<ModelBuild[]>(`/folders/${id}/builds`),
  getModels: (id: string) => request<Model[]>(`/folders/${id}/models`),
  addBuild: (folderId: string, buildId: string) =>
    request<void>(`/folders/${folderId}/builds`, { method: 'POST', body: JSON.stringify({ build_id: buildId }) }),
};

// Project API
export const projectApi = {
  getProjectsInFolder: (folderId: string) => request<Project[]>(`/folders/${folderId}/projects`),
  getRootProjects: () => request<Project[]>('/projects?root=true'),
  getProject: (id: string) => request<Project>(`/projects/${id}`),
  create: (data: CreateProjectRequest) =>
    request<Project>('/projects', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: UpdateProjectRequest) =>
    request<Project>(`/projects/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request<void>(`/projects/${id}`, { method: 'DELETE' }),
  addBuild: (projectId: string, buildId: string) =>
    request<void>(`/projects/${projectId}/builds`, { method: 'POST', body: JSON.stringify({ build_id: buildId }) }),
};

// Model Build API
export interface UpdateBuildRequest {
  name?: string;
  description?: string;
}

export const buildApi = {
  getBuildsInProject: (projectId: string) => request<ModelBuild[]>(`/projects/${projectId}/builds`),
  list: () => request<{ builds: ModelBuild[]; total: number }>('/builds'),
  get: (id: string) => request<ModelBuild>(`/builds/${id}`),
  create: (data: CreateBuildRequest) =>
    request<ModelBuild>('/builds', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: UpdateBuildRequest) =>
    request<ModelBuild>(`/builds/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  start: (id: string) => request<ModelBuild>(`/builds/${id}/start`, { method: 'POST' }),
  cancel: (id: string) => request<ModelBuild>(`/builds/${id}/cancel`, { method: 'POST' }),
  delete: (id: string) => request<void>(`/builds/${id}`, { method: 'DELETE' }),
};

// Score request/response interfaces
export interface ScoreRequest {
  datasource_id: string;
  output_collection_id: string;
  output_table_name?: string;
}

export interface ScoreResponse {
  job_id: string;
  status: string;
  message: string;
  output_datasource_id?: string;
}

// File content response
export interface FileContentResponse {
  file_id: string;
  file_name: string;
  file_type: string;
  content_type: string;
  content: string;
  size: number;
  is_text: boolean;
}

// Model API
export const modelApi = {
  getModelsInProject: (projectId: string) => request<Model[]>(`/projects/${projectId}/models`),
  list: () => request<{ models: Model[]; total: number }>('/models'),
  get: (id: string) => request<Model>(`/models/${id}`),
  getDetail: (id: string) => request<ModelDetail>(`/models/${id}`),
  update: (id: string, data: { name?: string; description?: string }) =>
    request<Model>(`/models/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  activate: (id: string) => request<Model>(`/models/${id}/activate`, { method: 'POST' }),
  deactivate: (id: string) => request<Model>(`/models/${id}/deactivate`, { method: 'POST' }),
  delete: (id: string) => request<void>(`/models/${id}`, { method: 'DELETE' }),
  score: (modelId: string, req: ScoreRequest) =>
    request<ScoreResponse>(`/models/${modelId}/score`, { method: 'POST', body: JSON.stringify(req) }),
  getFileContent: (modelId: string, fileId: string) =>
    request<FileContentResponse>(`/models/${modelId}/files/${fileId}/content`),
  retrain: (modelId: string, body?: RetrainRequest) =>
    request<ModelBuild>(`/models/${modelId}/retrain`, {
      method: 'POST',
      body: body && Object.keys(body).length > 0 ? JSON.stringify(body) : undefined,
    }),
};

// Model version types and API
export interface ModelVersion {
  id: string;
  model_id: string;
  version_number: number;
  name: string;
  description: string;
  created_by: string;
  created_at: string;
}

export interface ModelVersionDetail extends ModelVersion {
  build_id: string;
  datasource_id: string;
  project_id?: string;
  folder_id?: string;
  algorithm: string;
  model_type: string;
  target_column: string;
  status: string;
  metrics?: Record<string, number>;
  variables: ModelVariable[];
  files: ModelFile[];
}

export interface RetrainRequest {
  datasource_id?: string;
  name?: string;
  parameters?: {
    train_test_split?: number;
    hyperparameters?: Record<string, unknown>;
    random_seed?: number;
  };
}

export const versionApi = {
  list: (modelId: string, params?: { page?: number; page_size?: number }) => {
    const search = new URLSearchParams();
    if (params?.page != null) search.set('page', String(params.page));
    if (params?.page_size != null) search.set('page_size', String(params.page_size));
    const q = search.toString();
    return request<{ versions: ModelVersion[]; total: number }>(
      `/models/${modelId}/versions${q ? '?' + q : ''}`
    );
  },
  get: (modelId: string, versionId: string) =>
    request<ModelVersionDetail>(`/models/${modelId}/versions/${versionId}`),
  create: (modelId: string) =>
    request<ModelVersion>(`/models/${modelId}/versions`, { method: 'POST' }),
  restore: (modelId: string, versionId: string) =>
    request<Model>(`/models/${modelId}/versions/${versionId}/restore`, { method: 'POST' }),
};

// Performance Monitoring Types
export interface PerformanceBaseline {
  id: string;
  model_id: string;
  task_type: string;
  metric_name: string;
  metric_value: number;
  sample_count: number;
  description?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface PerformanceRecord {
  id: string;
  model_id: string;
  datasource_id: string;
  metric_name: string;
  metric_value: number;
  baseline_value?: number;
  drift_percentage?: number;
  sample_count: number;
  window_start: string;
  window_end: string;
  created_by: string;
  created_at: string;
}

export interface PerformanceAlert {
  id: string;
  model_id: string;
  record_id?: string;
  alert_type: string;
  severity: 'info' | 'warning' | 'critical';
  metric_name: string;
  baseline_value: number;
  current_value: number;
  threshold_percentage: number;
  drift_percentage: number;
  message: string;
  status: 'active' | 'acknowledged' | 'resolved';
  acknowledged_by?: string;
  acknowledged_at?: string;
  resolved_at?: string;
  created_at: string;
  updated_at: string;
}

export interface PerformanceThreshold {
  id: string;
  model_id: string;
  metric_name: string;
  warning_threshold: number;
  critical_threshold: number;
  direction: 'lower' | 'higher';
  enabled: boolean;
  consecutive_breaches: number;
  created_at: string;
  updated_at: string;
}

export interface PerformanceEvaluation {
  id: string;
  model_id: string;
  datasource_id: string;
  status: 'pending' | 'running' | 'completed' | 'failed';
  task_type: string;
  metrics?: Record<string, unknown>;
  sample_count: number;
  error_message?: string;
  started_at?: string;
  completed_at?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface PerformanceSummary {
  model_id: string;
  task_type: string;
  has_baseline: boolean;
  last_evaluation_at?: string;
  active_alerts: number;
  warning_alerts: number;
  critical_alerts: number;
  latest_metrics: Record<string, number>;
  baseline_metrics: Record<string, number>;
  drift_percentages: Record<string, number>;
  overall_health_status: 'healthy' | 'warning' | 'critical';
  record_count: number;
}

export interface MetricDataPoint {
  timestamp: string;
  value: number;
  drift_percentage?: number;
  sample_count: number;
}

export interface MetricTimeSeries {
  metric_name: string;
  baseline?: number;
  data_points: MetricDataPoint[];
}

// Performance API
export const performanceApi = {
  // Summary
  getSummary: (modelId: string) =>
    request<PerformanceSummary>(`/models/${modelId}/performance`),

  // Baselines
  getBaselines: (modelId: string) =>
    request<{ baselines: PerformanceBaseline[] }>(`/models/${modelId}/performance/baselines`),
  createBaseline: (modelId: string, data: { metrics: Record<string, number>; sample_count?: number; description?: string }) =>
    request<{ baselines: PerformanceBaseline[] }>(`/models/${modelId}/performance/baselines`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // Performance History
  getHistory: (modelId: string, params?: { metric_name?: string; limit?: number }) => {
    const searchParams = new URLSearchParams();
    if (params?.metric_name) searchParams.set('metric_name', params.metric_name);
    if (params?.limit) searchParams.set('limit', params.limit.toString());
    const query = searchParams.toString();
    return request<{ model_id: string; records: PerformanceRecord[]; total_count: number }>(
      `/models/${modelId}/performance/history${query ? '?' + query : ''}`
    );
  },

  // Record Performance
  recordPerformance: (modelId: string, data: { datasource_id: string; metrics: Record<string, number>; sample_count?: number }) =>
    request<{ model_id: string; records: PerformanceRecord[]; total_count: number }>(
      `/models/${modelId}/performance/record`,
      { method: 'POST', body: JSON.stringify(data) }
    ),

  // Metric Time Series
  getMetricTimeSeries: (modelId: string, metricName: string, limit?: number) =>
    request<MetricTimeSeries>(
      `/models/${modelId}/performance/metrics/${metricName}/series${limit ? '?limit=' + limit : ''}`
    ),

  // Evaluations
  startEvaluation: (modelId: string, data: { datasource_id: string; actual_column: string; prediction_column?: string }) =>
    request<PerformanceEvaluation>(`/models/${modelId}/performance/evaluate`, {
      method: 'POST',
      body: JSON.stringify(data),
    }),
  getEvaluations: (modelId: string, limit?: number) =>
    request<{ evaluations: PerformanceEvaluation[]; total_count: number }>(
      `/models/${modelId}/performance/evaluations${limit ? '?limit=' + limit : ''}`
    ),
  getEvaluation: (modelId: string, evaluationId: string) =>
    request<PerformanceEvaluation>(`/models/${modelId}/performance/evaluations/${evaluationId}`),

  // Alerts
  getAlerts: (modelId: string, status?: string) =>
    request<{ alerts: PerformanceAlert[]; total_count: number }>(
      `/models/${modelId}/performance/alerts${status ? '?status=' + status : ''}`
    ),
  updateAlert: (modelId: string, alertId: string, data: { status: 'acknowledged' | 'resolved' }) =>
    request<PerformanceAlert>(`/models/${modelId}/performance/alerts/${alertId}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),

  // Thresholds
  getThresholds: (modelId: string) =>
    request<{ thresholds: PerformanceThreshold[] }>(`/models/${modelId}/performance/thresholds`),
  updateThreshold: (modelId: string, data: {
    metric_name: string;
    warning_threshold?: number;
    critical_threshold?: number;
    direction?: 'lower' | 'higher';
    enabled?: boolean;
    consecutive_breaches?: number;
  }) =>
    request<PerformanceThreshold>(`/models/${modelId}/performance/thresholds`, {
      method: 'PUT',
      body: JSON.stringify(data),
    }),
};

// Datasource API
export interface DataPreview {
  columns: string[];
  rows: Record<string, unknown>[];
  total_rows: number;
  preview_max: number;
}

export const datasourceApi = {
  list: async (collectionId?: string) => {
    const params = collectionId ? `?collection_id=${collectionId}` : '';
    const result = await request<{ datasources: Datasource[]; total: number }>(`/datasources${params}`);
    return result.datasources || [];
  },
  get: (id: string) => request<Datasource>(`/datasources/${id}`),
  getColumns: (id: string) => request<Column[]>(`/datasources/${id}/columns`),
  delete: (id: string) => request<void>(`/datasources/${id}`, { method: 'DELETE' }),
  updateColumnRoles: (id: string, columns: { column_id: string; role: string }[]) =>
    request<Column[]>(`/datasources/${id}/columns/roles`, {
      method: 'PUT',
      body: JSON.stringify({ columns }),
    }),
  getPreview: (id: string, limit?: number) => 
    request<DataPreview>(`/datasources/${id}/preview${limit ? `?limit=${limit}` : ''}`),
};

export const collectionApi = {
  list: async () => {
    const result = await request<{ collections: Collection[]; total: number }>('/collections');
    return result.collections || [];
  },
  get: (id: string) => request<Collection>(`/collections/${id}`),
  create: (data: { name: string; description?: string }) => 
    request<Collection>('/collections', { method: 'POST', body: JSON.stringify(data) }),
  delete: (id: string, force?: boolean) => 
    request<void>(`/collections/${id}${force ? '?force=true' : ''}`, { method: 'DELETE' }),
};

// Types
export interface User {
  username: string;
  display_name: string;
  email: string;
  groups: string[];
}

export interface Folder {
  id: string;
  name: string;
  description?: string;
  parent_id?: string;
  path: string;
  depth: number;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface Project {
  id: string;
  name: string;
  description?: string;
  folder_id?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface ModelBuild {
  id: string;
  name: string;
  description?: string;
  datasource_id: string;
  project_id?: string;
  folder_id?: string;
  model_type: string;
  algorithm: string;
  status: 'pending' | 'running' | 'completed' | 'failed' | 'cancelled';
  parameters?: Record<string, unknown>;
  metrics?: Record<string, number>;
  error_message?: string;
  started_at?: string;
  completed_at?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface Model {
  id: string;
  name: string;
  description?: string;
  build_id: string;
  datasource_id: string;
  project_id?: string;
  folder_id?: string;
  algorithm: string;
  model_type: string;
  target_column: string;
  status: 'draft' | 'active' | 'inactive' | 'archived';
  metrics?: Record<string, number>;
  version: number;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface ModelVariable {
  id: string;
  model_id: string;
  name: string;
  data_type: string;
  role: 'input' | 'target';
  importance?: number;
  statistics?: Record<string, unknown>;
  encoding_info?: Record<string, unknown>;
  ordinal: number;
  created_at: string;
}

export interface ModelFile {
  id: string;
  model_id: string;
  file_type: string;
  file_path: string;
  file_name: string;
  file_size?: number;
  checksum?: string;
  description?: string;
  created_at: string;
}

export interface ModelDetail extends Model {
  variables: ModelVariable[];
  files: ModelFile[];
}

export interface CreateFolderRequest {
  name: string;
  description?: string;
  parent_id?: string;
}

export interface UpdateFolderRequest {
  name: string;
  description?: string;
}

export interface CreateProjectRequest {
  name: string;
  description?: string;
  folder_id?: string;
}

export interface UpdateProjectRequest {
  name: string;
  description?: string;
}

export interface CreateBuildRequest {
  name: string;
  description?: string;
  datasource_id: string;
  project_id?: string;  // Belongs to project (one-to-many)
  folder_id?: string;   // Belongs to folder (one-to-many)
  model_type: string;
  algorithm: string;    // ML algorithm: decision_tree, random_forest, xgboost
  parameters?: {
    hyperparameters?: Record<string, unknown>;
    train_test_split?: number;
    random_seed?: number;
    max_iterations?: number;
    early_stop_rounds?: number;
  };
}

export interface Collection {
  id: string;
  name: string;
  description?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  datasource_count: number;
}

export interface Datasource {
  id: string;
  name: string;
  description?: string;
  type: string;
  collection_id: string;
  collection_name?: string;
  file_path?: string;
  column_count: number;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface Column {
  id: string;
  name: string;
  data_type: string;
  role: 'input' | 'target' | 'exclude';
  description?: string;
  created_at: string;
  updated_at: string;
}

