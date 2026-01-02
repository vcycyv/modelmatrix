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

  const data = await response.json();

  if (!response.ok) {
    throw new Error(data.error || data.message || 'Request failed');
  }

  return data.data ?? data;
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
export const folderApi = {
  getRootFolders: () => request<Folder[]>('/folders'),
  getFolder: (id: string) => request<Folder>(`/folders/${id}`),
  getChildren: (id: string) => request<Folder[]>(`/folders/${id}/children`),
  create: (data: CreateFolderRequest) => 
    request<Folder>('/folders', { method: 'POST', body: JSON.stringify(data) }),
  update: (id: string, data: UpdateFolderRequest) =>
    request<Folder>(`/folders/${id}`, { method: 'PUT', body: JSON.stringify(data) }),
  delete: (id: string) => request<void>(`/folders/${id}`, { method: 'DELETE' }),
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
export const buildApi = {
  getBuildsInProject: (projectId: string) => request<ModelBuild[]>(`/projects/${projectId}/builds`),
  list: () => request<{ builds: ModelBuild[]; total: number }>('/builds'),
  get: (id: string) => request<ModelBuild>(`/builds/${id}`),
  create: (data: CreateBuildRequest) =>
    request<ModelBuild>('/builds', { method: 'POST', body: JSON.stringify(data) }),
  start: (id: string) => request<ModelBuild>(`/builds/${id}/start`, { method: 'POST' }),
  cancel: (id: string) => request<ModelBuild>(`/builds/${id}/cancel`, { method: 'POST' }),
  delete: (id: string) => request<void>(`/builds/${id}`, { method: 'DELETE' }),
};

// Model API
export const modelApi = {
  getModelsInProject: (projectId: string) => request<Model[]>(`/projects/${projectId}/models`),
  list: () => request<{ models: Model[]; total: number }>('/models'),
  get: (id: string) => request<Model>(`/models/${id}`),
  activate: (id: string) => request<Model>(`/models/${id}/activate`, { method: 'POST' }),
  deactivate: (id: string) => request<Model>(`/models/${id}/deactivate`, { method: 'POST' }),
  delete: (id: string) => request<void>(`/models/${id}`, { method: 'DELETE' }),
};

// Datasource API
export const datasourceApi = {
  list: async (collectionId?: string) => {
    const params = collectionId ? `?collection_id=${collectionId}` : '';
    const result = await request<{ datasources: Datasource[]; total: number }>(`/datasources${params}`);
    return result.datasources || [];
  },
  get: (id: string) => request<Datasource>(`/datasources/${id}`),
  getColumns: (id: string) => request<Column[]>(`/datasources/${id}/columns`),
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
  model_type: string;
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
  model_type: string;
  parameters?: {
    algorithm?: string;
    hyperparameters?: Record<string, unknown>;
    train_test_split?: number;
    random_seed?: number;
    max_iterations?: number;
    early_stop_rounds?: number;
  };
}

export interface Datasource {
  id: string;
  name: string;
  description?: string;
  type: string;
  collection_id?: string;
  row_count?: number;
  status: string;
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

