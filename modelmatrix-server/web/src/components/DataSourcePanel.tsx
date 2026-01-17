import { useState, useEffect, useCallback, MouseEvent, useRef, ChangeEvent } from 'react';

// Types
interface Collection {
  id: string;
  name: string;
  description?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  datasource_count: number;
}

interface Datasource {
  id: string;
  name: string;
  description?: string;
  collection_id: string;
  collection_name?: string;
  type: string;
  file_path?: string;
  column_count: number;
  created_by: string;
  created_at: string;
  updated_at: string;
}

interface DataSourcePanelProps {
  onSelect?: (item: { type: 'collection' | 'datasource'; data: Collection | Datasource }) => void;
  refreshTrigger?: number;
  externalSelectedDatasource?: { id: string; collection_id: string } | null;
}

// API functions (simplified - should be moved to api.ts)
const API_BASE = '/api';
const getToken = () => localStorage.getItem('token');

async function apiRequest<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...options.headers,
  };
  const token = getToken();
  if (token) {
    (headers as Record<string, string>)['Authorization'] = `Bearer ${token}`;
  }
  const response = await fetch(`${API_BASE}${endpoint}`, { ...options, headers });
  
  // Handle empty responses (e.g., 204 No Content from DELETE)
  const contentType = response.headers.get('content-type');
  const hasJsonContent = contentType && contentType.includes('application/json');
  const text = await response.text();
  
  let data: Record<string, unknown> | null = null;
  if (text && hasJsonContent) {
    try {
      data = JSON.parse(text);
    } catch {
      data = null;
    }
  }
  
  if (!response.ok) {
    const errorMessage = data?.msg || data?.error || data?.message || 'Request failed';
    throw new Error(String(errorMessage));
  }
  
  if (!data) {
    return {} as T;
  }
  
  return (data.data ?? data) as T;
}

interface CollectionListResponse {
  collections: Collection[];
  total: number;
}

const collectionApi = {
  list: async (): Promise<Collection[]> => {
    const response = await apiRequest<CollectionListResponse>('/collections');
    return response.collections || [];
  },
  create: (data: { name: string; description?: string }) =>
    apiRequest<Collection>('/collections', { method: 'POST', body: JSON.stringify(data) }),
  delete: (id: string, force?: boolean) => 
    apiRequest<void>(`/collections/${id}${force ? '?force=true' : ''}`, { method: 'DELETE' }),
};

interface DatasourceListResponse {
  datasources: Datasource[];
  total: number;
}

const datasourceApi = {
  listByCollection: async (collectionId: string): Promise<Datasource[]> => {
    const response = await apiRequest<DatasourceListResponse>(`/datasources?collection_id=${collectionId}`);
    return response.datasources || [];
  },
  upload: async (collectionId: string, file: File, name: string, description?: string) => {
    // Determine type from file extension
    const fileExt = file.name.toLowerCase().slice(file.name.lastIndexOf('.'));
    const fileType = fileExt === '.parquet' ? 'parquet' : 'csv';
    
    const formData = new FormData();
    formData.append('file', file);
    formData.append('name', name);
    formData.append('collection_id', collectionId);
    formData.append('type', fileType);
    if (description) formData.append('description', description);
    
    const token = getToken();
    const response = await fetch(`${API_BASE}/datasources`, {
      method: 'POST',
      headers: token ? { 'Authorization': `Bearer ${token}` } : {},
      body: formData,
    });
    const data = await response.json();
    if (!response.ok) {
      throw new Error(data.msg || data.error || 'Upload failed');
    }
    return data.data as Datasource;
  },
  delete: (id: string) => apiRequest<void>(`/datasources/${id}`, { method: 'DELETE' }),
};

// Icons
const CollectionIcon = () => (
  <svg className="w-5 h-5 text-indigo-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
  </svg>
);

const FileIcon = () => (
  <svg className="w-5 h-5 text-emerald-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
  </svg>
);

const ChevronIcon = ({ isExpanded }: { isExpanded: boolean }) => (
  <svg
    className={`w-4 h-4 text-slate-400 transition-transform ${isExpanded ? 'rotate-90' : ''}`}
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
  >
    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
  </svg>
);

// Collection Item Component
interface CollectionItemProps {
  collection: Collection;
  isSelected: boolean;
  isExpanded: boolean;
  datasources: Datasource[];
  onSelect: () => void;
  onToggle: () => void;
  onContextMenu: (e: MouseEvent) => void;
  onDatasourceSelect: (ds: Datasource) => void;
  onDatasourceContextMenu: (e: MouseEvent, ds: Datasource) => void;
  selectedDatasourceId?: string;
}

function CollectionItem({
  collection,
  isSelected,
  isExpanded,
  datasources,
  onSelect,
  onToggle,
  onContextMenu,
  onDatasourceSelect,
  onDatasourceContextMenu,
  selectedDatasourceId,
}: CollectionItemProps) {
  return (
    <div>
      <div
        data-tree-node="true"
        className={`flex items-center py-1.5 px-2 cursor-pointer transition-colors ${
          isSelected ? 'bg-blue-100 text-blue-900' : 'hover:bg-slate-100 text-slate-700'
        }`}
        style={{ paddingLeft: 12 }}
        onClick={() => {
          onSelect();
          onToggle();
        }}
        onContextMenu={(e) => {
          e.stopPropagation();
          e.preventDefault();
          onContextMenu(e);
        }}
      >
        <span className="mr-1">
          <ChevronIcon isExpanded={isExpanded} />
        </span>
        <span className="mr-2">
          <CollectionIcon />
        </span>
        <span className="truncate text-sm font-medium">{collection.name}</span>
      </div>

      {isExpanded && (
        <div>
          {datasources.map((ds) => (
            <div
              key={ds.id}
              data-tree-node="true"
              className={`flex items-center py-1.5 px-2 cursor-pointer transition-colors ${
                selectedDatasourceId === ds.id
                  ? 'bg-blue-100 text-blue-900'
                  : 'hover:bg-slate-100 text-slate-700'
              }`}
              style={{ paddingLeft: 52 }}
              onClick={() => onDatasourceSelect(ds)}
              onContextMenu={(e) => {
                e.stopPropagation();
                e.preventDefault();
                onDatasourceContextMenu(e, ds);
              }}
            >
              <span className="mr-2">
                <FileIcon />
              </span>
              <span className="truncate text-sm">{ds.name}</span>
              {ds.column_count > 0 && (
                <span className="ml-auto text-xs text-slate-400">
                  {ds.column_count} cols
                </span>
              )}
            </div>
          ))}
          {datasources.length === 0 && (
            <div className="py-2 px-4 text-xs text-slate-400" style={{ paddingLeft: 52 }}>
              No data files. Right-click to upload.
            </div>
          )}
        </div>
      )}
    </div>
  );
}

// Context Menu Component
interface ContextMenuState {
  x: number;
  y: number;
  type: 'root' | 'collection' | 'datasource';
  collection?: Collection;
  datasource?: Datasource;
}

// Dialog Components
function CreateCollectionDialog({
  isOpen,
  onClose,
  onSuccess,
}: {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
}) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setIsLoading(true);
    try {
      await collectionApi.create({ name, description });
      onSuccess();
      onClose();
      setName('');
      setDescription('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create collection');
    } finally {
      setIsLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white rounded-xl shadow-xl w-full max-w-md p-6">
        <h2 className="text-lg font-semibold text-slate-900 mb-4">Create Collection</h2>
        <form onSubmit={handleSubmit}>
          {error && (
            <div className="mb-4 p-3 rounded-lg bg-red-50 border border-red-200 text-red-600 text-sm">
              {error}
            </div>
          )}
          <div className="mb-4">
            <label className="block text-sm font-medium text-slate-700 mb-1">Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              placeholder="Collection name"
              required
              autoFocus
            />
          </div>
          <div className="mb-4">
            <label className="block text-sm font-medium text-slate-700 mb-1">Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              placeholder="Optional description"
              rows={3}
            />
          </div>
          <div className="flex justify-end space-x-3">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm text-slate-700 hover:bg-slate-100 rounded-lg transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isLoading}
              className="px-4 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
            >
              {isLoading ? 'Creating...' : 'Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// Column role options
const COLUMN_ROLES = [
  { value: 'input', label: 'Input', description: 'Feature used for prediction' },
  { value: 'target', label: 'Target', description: 'Variable to predict' },
  { value: 'ignore', label: 'Ignore', description: 'Excluded from model' },
] as const;

interface ColumnWithRole {
  id: string;
  name: string;
  data_type: string;
  role: string;
}

function UploadFileDialog({
  isOpen,
  onClose,
  onSuccess,
  collection,
  collections,
}: {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: (collectionId: string) => void;
  collection: Collection | null;
  collections?: Collection[];
}) {
  const [step, setStep] = useState<'upload' | 'configure'>('upload');
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [file, setFile] = useState<File | null>(null);
  const [selectedCollectionId, setSelectedCollectionId] = useState<string>(collection?.id || '');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [isDragging, setIsDragging] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);
  
  // Column configuration state
  const [datasourceId, setDatasourceId] = useState<string | null>(null);
  const [columns, setColumns] = useState<ColumnWithRole[]>([]);
  const [isSavingRoles, setIsSavingRoles] = useState(false);

  // Reset selected collection when dialog opens
  useEffect(() => {
    if (isOpen) {
      setStep('upload');
      setSelectedCollectionId(collection?.id || '');
      setName('');
      setDescription('');
      setFile(null);
      setError('');
      setDatasourceId(null);
      setColumns([]);
    }
  }, [isOpen, collection?.id]);

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const selectedFile = e.target.files?.[0];
    if (selectedFile) {
      handleFileSelected(selectedFile);
    }
  };

  const handleFileSelected = (selectedFile: File) => {
    // Validate file type
    const validTypes = ['.csv', '.parquet'];
    const fileExt = selectedFile.name.toLowerCase().slice(selectedFile.name.lastIndexOf('.'));
    if (!validTypes.includes(fileExt)) {
      setError('Please select a CSV or Parquet file');
      return;
    }
    
    setFile(selectedFile);
    setError('');
    if (!name) {
      // Auto-fill name from filename
      setName(selectedFile.name.replace(/\.[^/.]+$/, ''));
    }
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(true);
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    setIsDragging(false);

    const droppedFile = e.dataTransfer.files?.[0];
    if (droppedFile) {
      handleFileSelected(droppedFile);
    }
  };

  const handleUpload = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!file || !selectedCollectionId) return;

    setError('');
    setIsLoading(true);
    try {
      const result = await datasourceApi.upload(selectedCollectionId, file, name, description);
      setDatasourceId(result.id);
      
      // Fetch columns for the new datasource
      const token = localStorage.getItem('token');
      const response = await fetch(`/api/datasources/${result.id}/columns`, {
        headers: token ? { 'Authorization': `Bearer ${token}` } : {},
      });
      const data = await response.json();
      const columnsData = data.data || data || [];
      
      setColumns(columnsData.map((col: { id: string; name: string; data_type: string; role?: string }) => ({
        id: col.id,
        name: col.name,
        data_type: col.data_type,
        role: col.role || 'input',
      })));
      
      setStep('configure');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to upload file');
    } finally {
      setIsLoading(false);
    }
  };

  const handleColumnRoleChange = (columnId: string, role: string) => {
    setColumns((prev) =>
      prev.map((col) =>
        col.id === columnId ? { ...col, role } : col
      )
    );
  };

  const handleSaveRoles = async () => {
    if (!datasourceId) return;

    // Validate: exactly one target column
    const targetCount = columns.filter((c) => c.role === 'target').length;
    if (targetCount === 0) {
      setError('Please select exactly one target column');
      return;
    }
    if (targetCount > 1) {
      setError('Only one target column is allowed');
      return;
    }

    setError('');
    setIsSavingRoles(true);
    try {
      const token = localStorage.getItem('token');
      const response = await fetch(`/api/datasources/${datasourceId}/columns/roles`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
        },
        body: JSON.stringify({
          columns: columns.map((col) => ({
            column_id: col.id,
            role: col.role,
          })),
        }),
      });

      if (!response.ok) {
        const data = await response.json();
        throw new Error(data.msg || data.error || 'Failed to save column roles');
      }

      onSuccess(selectedCollectionId);
      handleClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save column roles');
    } finally {
      setIsSavingRoles(false);
    }
  };

  const handleSkipConfiguration = () => {
    onSuccess(selectedCollectionId);
    handleClose();
  };

  const handleClose = () => {
    setStep('upload');
    setName('');
    setDescription('');
    setFile(null);
    setError('');
    setSelectedCollectionId('');
    setDatasourceId(null);
    setColumns([]);
    onClose();
  };

  if (!isOpen) return null;

  // Step 2: Configure column roles
  if (step === 'configure') {
    return (
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
        <div className="bg-white rounded-xl shadow-xl w-full max-w-2xl p-6 max-h-[90vh] flex flex-col">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h2 className="text-lg font-semibold text-slate-900">Configure Columns</h2>
              <p className="text-sm text-slate-500">Set the role for each column in your dataset</p>
            </div>
            <div className="flex items-center space-x-2 text-xs">
              <span className="px-2 py-1 bg-blue-100 text-blue-700 rounded">Input</span>
              <span className="px-2 py-1 bg-emerald-100 text-emerald-700 rounded">Target</span>
              <span className="px-2 py-1 bg-slate-100 text-slate-500 rounded">Ignore</span>
            </div>
          </div>

          {error && (
            <div className="mb-4 p-3 rounded-lg bg-red-50 border border-red-200 text-red-600 text-sm">
              {error}
            </div>
          )}

          <div className="flex-1 overflow-auto border border-slate-200 rounded-lg">
            <table className="w-full text-sm">
              <thead className="bg-slate-50 sticky top-0">
                <tr>
                  <th className="text-left px-4 py-2 font-medium text-slate-600">Column</th>
                  <th className="text-left px-4 py-2 font-medium text-slate-600">Type</th>
                  <th className="text-left px-4 py-2 font-medium text-slate-600">Role</th>
                </tr>
              </thead>
              <tbody>
                {columns.map((col, index) => (
                  <tr key={col.id} className={index % 2 === 0 ? 'bg-white' : 'bg-slate-50/50'}>
                    <td className="px-4 py-2">
                      <code className="text-xs bg-slate-100 px-1.5 py-0.5 rounded">{col.name}</code>
                    </td>
                    <td className="px-4 py-2 text-slate-500 text-xs">{col.data_type}</td>
                    <td className="px-4 py-2">
                      <div className="flex space-x-1">
                        {COLUMN_ROLES.map((role) => (
                          <button
                            key={role.value}
                            type="button"
                            onClick={() => handleColumnRoleChange(col.id, role.value)}
                            className={`px-2 py-1 text-xs rounded transition-colors ${
                              col.role === role.value
                                ? role.value === 'target'
                                  ? 'bg-emerald-500 text-white'
                                  : role.value === 'ignore'
                                  ? 'bg-slate-400 text-white'
                                  : 'bg-blue-500 text-white'
                                : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
                            }`}
                            title={role.description}
                          >
                            {role.label}
                          </button>
                        ))}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="flex items-center justify-between mt-4 pt-4 border-t border-slate-200">
            <div className="text-xs text-slate-500">
              {columns.filter((c) => c.role === 'input').length} inputs, 
              {' '}{columns.filter((c) => c.role === 'target').length} target,
              {' '}{columns.filter((c) => c.role === 'ignore').length} ignored
            </div>
            <div className="flex space-x-3">
              <button
                type="button"
                onClick={handleSkipConfiguration}
                className="px-4 py-2 text-sm text-slate-600 hover:bg-slate-100 rounded-lg transition-colors"
              >
                Skip (use defaults)
              </button>
              <button
                type="button"
                onClick={handleSaveRoles}
                disabled={isSavingRoles}
                className="px-4 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors flex items-center"
              >
                {isSavingRoles ? (
                  <>
                    <svg className="animate-spin h-4 w-4 mr-2" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                    </svg>
                    Saving...
                  </>
                ) : (
                  'Save & Continue'
                )}
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  }

  // Step 1: Upload file
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white rounded-xl shadow-xl w-full max-w-md p-6">
        <h2 className="text-lg font-semibold text-slate-900 mb-4">
          {collection ? `Upload to "${collection.name}"` : 'Upload Data File'}
        </h2>
        <form onSubmit={handleUpload}>
          {error && (
            <div className="mb-4 p-3 rounded-lg bg-red-50 border border-red-200 text-red-600 text-sm">
              {error}
            </div>
          )}
          
          {/* Collection selector (shown when no collection is pre-selected) */}
          {!collection && collections && collections.length > 0 && (
            <div className="mb-4">
              <label className="block text-sm font-medium text-slate-700 mb-1">Collection</label>
              <select
                value={selectedCollectionId}
                onChange={(e) => setSelectedCollectionId(e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                required
              >
                <option value="">Select a collection...</option>
                {collections.map((c) => (
                  <option key={c.id} value={c.id}>{c.name}</option>
                ))}
              </select>
            </div>
          )}

          {!collection && (!collections || collections.length === 0) && (
            <div className="mb-4 p-3 rounded-lg bg-amber-50 border border-amber-200 text-amber-700 text-sm">
              No collections found. Please create a collection first.
            </div>
          )}

          <div className="mb-4">
            <label className="block text-sm font-medium text-slate-700 mb-1">File</label>
            <div
              className={`border-2 border-dashed rounded-lg p-6 text-center cursor-pointer transition-colors ${
                isDragging 
                  ? 'border-blue-500 bg-blue-50' 
                  : file 
                    ? 'border-emerald-400 bg-emerald-50' 
                    : 'border-slate-300 hover:border-blue-400'
              }`}
              onClick={() => fileInputRef.current?.click()}
              onDragOver={handleDragOver}
              onDragLeave={handleDragLeave}
              onDrop={handleDrop}
            >
              {file ? (
                <div className="text-sm">
                  <div className="flex items-center justify-center text-emerald-600 mb-1">
                    <svg className="w-8 h-8" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                    </svg>
                  </div>
                  <span className="font-medium text-slate-700">{file.name}</span>
                  <div className="text-slate-400 text-xs mt-1">
                    {(file.size / 1024 / 1024).toFixed(2)} MB
                  </div>
                  <button
                    type="button"
                    onClick={(e) => {
                      e.stopPropagation();
                      setFile(null);
                      setName('');
                    }}
                    className="mt-2 text-xs text-slate-500 hover:text-red-500 underline"
                  >
                    Remove
                  </button>
                </div>
              ) : (
                <div className="text-sm text-slate-500">
                  <svg className="w-10 h-10 mx-auto mb-2 text-slate-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                  </svg>
                  <p className="font-medium">Drop your file here</p>
                  <p className="text-xs text-slate-400 mt-1">or click to browse</p>
                  <p className="text-xs text-slate-400 mt-2">Supports CSV and Parquet files</p>
                </div>
              )}
              <input
                ref={fileInputRef}
                type="file"
                accept=".csv,.parquet"
                onChange={handleFileChange}
                className="hidden"
              />
            </div>
          </div>
          
          <div className="mb-4">
            <label className="block text-sm font-medium text-slate-700 mb-1">Name</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              placeholder="Data source name"
              required
            />
          </div>
          
          <div className="mb-4">
            <label className="block text-sm font-medium text-slate-700 mb-1">Description</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              className="w-full px-3 py-2 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              placeholder="Optional description"
              rows={2}
            />
          </div>
          
          <div className="flex justify-end space-x-3">
            <button
              type="button"
              onClick={handleClose}
              className="px-4 py-2 text-sm text-slate-700 hover:bg-slate-100 rounded-lg transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isLoading || !file || !selectedCollectionId}
              className="px-4 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors flex items-center"
            >
              {isLoading ? (
                <>
                  <svg className="animate-spin h-4 w-4 mr-2" fill="none" viewBox="0 0 24 24">
                    <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                    <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                  </svg>
                  Uploading...
                </>
              ) : (
                <>
                  <svg className="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
                  </svg>
                  Upload & Configure
                </>
              )}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// Main DataSourcePanel Component
export default function DataSourcePanel({ onSelect, refreshTrigger, externalSelectedDatasource }: DataSourcePanelProps) {
  const [collections, setCollections] = useState<Collection[]>([]);
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  const [datasourcesByCollection, setDatasourcesByCollection] = useState<Record<string, Datasource[]>>({});
  const [selectedCollectionId, setSelectedCollectionId] = useState<string | null>(null);
  const [selectedDatasourceId, setSelectedDatasourceId] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null);
  const [pendingExternalSelection, setPendingExternalSelection] = useState<{ id: string; collection_id: string } | null>(null);

  // Handle external selection (e.g., when navigating from model details)
  useEffect(() => {
    if (externalSelectedDatasource && externalSelectedDatasource.id !== selectedDatasourceId) {
      const { id, collection_id } = externalSelectedDatasource;
      
      // Update selection immediately
      setSelectedDatasourceId(id);
      setSelectedCollectionId(null);
      
      // Expand the collection
      setExpandedIds(prev => new Set([...prev, collection_id]));
      
      // If collection's datasources aren't loaded yet, load them
      if (!datasourcesByCollection[collection_id]) {
        setPendingExternalSelection(externalSelectedDatasource);
      }
    }
  }, [externalSelectedDatasource, selectedDatasourceId, datasourcesByCollection]);

  // Dialog states
  const [createCollectionOpen, setCreateCollectionOpen] = useState(false);
  const [uploadFileOpen, setUploadFileOpen] = useState(false);
  const [uploadCollection, setUploadCollection] = useState<Collection | null>(null);

  // Load collections
  const loadCollections = useCallback(async () => {
    setIsLoading(true);
    try {
      const data = await collectionApi.list();
      setCollections(data);
    } catch (error) {
      console.error('Failed to load collections:', error);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Load datasources for a collection
  const loadDatasources = useCallback(async (collectionId: string) => {
    try {
      const datasources = await datasourceApi.listByCollection(collectionId);
      setDatasourcesByCollection((prev) => ({ ...prev, [collectionId]: datasources }));
    } catch (error) {
      console.error('Failed to load datasources:', error);
    }
  }, []);

  useEffect(() => {
    loadCollections();
  }, [loadCollections, refreshTrigger]);

  // Load datasources for newly expanded collections (including from external selection)
  useEffect(() => {
    expandedIds.forEach((collectionId) => {
      if (!datasourcesByCollection[collectionId]) {
        loadDatasources(collectionId);
      }
    });
  }, [expandedIds, datasourcesByCollection, loadDatasources]);

  // Clear pending selection once the datasources are loaded
  useEffect(() => {
    if (pendingExternalSelection) {
      const { collection_id } = pendingExternalSelection;
      if (datasourcesByCollection[collection_id]) {
        setPendingExternalSelection(null);
      }
    }
  }, [pendingExternalSelection, datasourcesByCollection]);

  // Reload all expanded collections' datasources when refreshTrigger changes
  useEffect(() => {
    if (refreshTrigger !== undefined && refreshTrigger > 0) {
      expandedIds.forEach((collectionId) => {
        loadDatasources(collectionId);
      });
    }
  }, [refreshTrigger, expandedIds, loadDatasources]);

  // Refresh all: reload collections and all expanded collections' datasources
  const handleRefreshAll = useCallback(async () => {
    await loadCollections();
    // After collections are loaded, refresh all expanded ones
    expandedIds.forEach((collectionId) => {
      loadDatasources(collectionId);
    });
  }, [loadCollections, expandedIds, loadDatasources]);

  // Refresh a single collection's datasources
  const handleRefreshCollection = useCallback((collectionId: string) => {
    loadDatasources(collectionId);
  }, [loadDatasources]);

  // Toggle collection expansion
  const handleToggle = (collection: Collection) => {
    const newExpanded = new Set(expandedIds);
    if (newExpanded.has(collection.id)) {
      newExpanded.delete(collection.id);
    } else {
      newExpanded.add(collection.id);
      // Load datasources if not already loaded
      if (!datasourcesByCollection[collection.id]) {
        loadDatasources(collection.id);
      }
    }
    setExpandedIds(newExpanded);
  };

  // Handle collection select
  const handleCollectionSelect = (collection: Collection) => {
    setSelectedCollectionId(collection.id);
    setSelectedDatasourceId(null);
    onSelect?.({ type: 'collection', data: collection });
  };

  // Handle datasource select
  const handleDatasourceSelect = (datasource: Datasource) => {
    setSelectedCollectionId(null);
    setSelectedDatasourceId(datasource.id);
    onSelect?.({ type: 'datasource', data: datasource });
  };

  // Context menu handlers
  const handleRootContextMenu = (e: MouseEvent<HTMLDivElement>) => {
    if ((e.target as HTMLElement).closest('[data-tree-node]')) return;
    e.preventDefault();
    setContextMenu({ x: e.clientX, y: e.clientY, type: 'root' });
  };

  const handleCollectionContextMenu = (e: MouseEvent, collection: Collection) => {
    setContextMenu({ x: e.clientX, y: e.clientY, type: 'collection', collection });
  };

  const handleDatasourceContextMenu = (e: MouseEvent, datasource: Datasource) => {
    setContextMenu({ x: e.clientX, y: e.clientY, type: 'datasource', datasource });
  };

  // Context menu actions
  const handleDeleteCollection = async (collection: Collection) => {
    const hasDatasources = collection.datasource_count > 0;
    const message = hasDatasources
      ? `Warning: This collection contains ${collection.datasource_count} data source${collection.datasource_count > 1 ? 's' : ''}. Deleting this collection will remove all data sources in it. Are you sure you want to continue?`
      : `Delete collection "${collection.name}"? This action cannot be undone.`;
    
    if (!confirm(message)) return;
    try {
      await collectionApi.delete(collection.id, hasDatasources);
      loadCollections();
    } catch (error) {
      console.error('Failed to delete collection:', error);
      alert(error instanceof Error ? error.message : 'Failed to delete collection');
    }
  };

  const handleDeleteDatasource = async (datasource: Datasource) => {
    if (!confirm(`Delete data source "${datasource.name}"?`)) return;
    try {
      await datasourceApi.delete(datasource.id);
      loadDatasources(datasource.collection_id);
    } catch (error) {
      console.error('Failed to delete datasource:', error);
      alert(error instanceof Error ? error.message : 'Failed to delete datasource');
    }
  };

  // Render context menu
  const renderContextMenu = () => {
    if (!contextMenu) return null;

    const items: { label: string; onClick: () => void; danger?: boolean }[] = [];

    if (contextMenu.type === 'root') {
      items.push({ label: 'New Collection', onClick: () => setCreateCollectionOpen(true) });
    } else if (contextMenu.type === 'collection' && contextMenu.collection) {
      items.push({
        label: 'Upload Data File',
        onClick: () => {
          setUploadCollection(contextMenu.collection!);
          setUploadFileOpen(true);
        },
      });
      items.push({
        label: 'Refresh',
        onClick: () => handleRefreshCollection(contextMenu.collection!.id),
      });
      items.push({
        label: 'Delete Collection',
        onClick: () => handleDeleteCollection(contextMenu.collection!),
        danger: true,
      });
    } else if (contextMenu.type === 'datasource' && contextMenu.datasource) {
      items.push({
        label: 'Delete Data Source',
        onClick: () => handleDeleteDatasource(contextMenu.datasource!),
        danger: true,
      });
    }

    return (
      <div
        className="fixed z-50 min-w-48 py-1 bg-white rounded-lg shadow-lg border border-slate-200"
        style={{ left: contextMenu.x, top: contextMenu.y }}
      >
        {items.map((item, index) => (
          <button
            key={index}
            className={`w-full px-4 py-2 text-left text-sm transition-colors ${
              item.danger ? 'text-red-600 hover:bg-red-50' : 'text-slate-700 hover:bg-slate-100'
            }`}
            onClick={() => {
              item.onClick();
              setContextMenu(null);
            }}
          >
            {item.label}
          </button>
        ))}
      </div>
    );
  };

  // Close context menu on click outside
  useEffect(() => {
    const handleClick = () => setContextMenu(null);
    document.addEventListener('click', handleClick);
    return () => document.removeEventListener('click', handleClick);
  }, []);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full text-slate-500">
        <svg className="animate-spin h-6 w-6 mr-2" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
        Loading...
      </div>
    );
  }

  return (
    <div className="flex flex-col h-full" onContextMenu={handleRootContextMenu}>
      {/* Header */}
      <div className="px-4 py-3 border-b border-slate-200 flex items-center justify-between">
        <h2 className="text-sm font-semibold text-slate-700">Data Sources</h2>
        <div className="flex items-center space-x-1">
          <button
            onClick={() => {
              setUploadCollection(null);
              setUploadFileOpen(true);
            }}
            className="p-1.5 text-slate-500 hover:text-slate-700 hover:bg-slate-100 rounded transition-colors"
            title="Upload Data File"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
            </svg>
          </button>
          <button
            onClick={() => setCreateCollectionOpen(true)}
            className="p-1.5 text-slate-500 hover:text-slate-700 hover:bg-slate-100 rounded transition-colors"
            title="New Collection"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
          </button>
          <button
            onClick={handleRefreshAll}
            className="p-1.5 text-slate-500 hover:text-slate-700 hover:bg-slate-100 rounded transition-colors"
            title="Refresh All"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          </button>
        </div>
      </div>

      {/* Collection list */}
      <div className="flex-1 overflow-auto py-2">
        {collections.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-full text-slate-500 p-4">
            <CollectionIcon />
            <p className="text-sm mt-2">No collections</p>
            <p className="text-xs text-slate-400 mt-1">Right-click to create</p>
          </div>
        ) : (
          collections.map((collection) => (
            <CollectionItem
              key={collection.id}
              collection={collection}
              isSelected={selectedCollectionId === collection.id}
              isExpanded={expandedIds.has(collection.id)}
              datasources={datasourcesByCollection[collection.id] || []}
              onSelect={() => handleCollectionSelect(collection)}
              onToggle={() => handleToggle(collection)}
              onContextMenu={(e) => handleCollectionContextMenu(e, collection)}
              onDatasourceSelect={handleDatasourceSelect}
              onDatasourceContextMenu={handleDatasourceContextMenu}
              selectedDatasourceId={selectedDatasourceId || undefined}
            />
          ))
        )}
      </div>

      {/* Context Menu */}
      {renderContextMenu()}

      {/* Dialogs */}
      <CreateCollectionDialog
        isOpen={createCollectionOpen}
        onClose={() => setCreateCollectionOpen(false)}
        onSuccess={loadCollections}
      />
      <UploadFileDialog
        isOpen={uploadFileOpen}
        onClose={() => {
          setUploadFileOpen(false);
          setUploadCollection(null);
        }}
        onSuccess={(collectionId: string) => {
          // Refresh the datasources for the collection that was uploaded to
          loadDatasources(collectionId);
          // Expand the collection to show the new datasource
          setExpandedIds((prev) => new Set([...prev, collectionId]));
        }}
        collection={uploadCollection}
        collections={collections}
      />
    </div>
  );
}

