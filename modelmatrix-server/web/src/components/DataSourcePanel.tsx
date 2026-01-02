import { useState, useEffect, useCallback, MouseEvent, useRef, ChangeEvent } from 'react';

// Types
interface Collection {
  id: string;
  name: string;
  description?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

interface Datasource {
  id: string;
  name: string;
  description?: string;
  collection_id: string;
  source_type: string;
  file_path?: string;
  row_count?: number;
  column_count?: number;
  status: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

interface DataSourcePanelProps {
  onSelect?: (item: { type: 'collection' | 'datasource'; data: Collection | Datasource }) => void;
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
  const data = await response.json();
  if (!response.ok) {
    throw new Error(data.msg || data.error || 'Request failed');
  }
  return data.data ?? data;
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
  delete: (id: string) => apiRequest<void>(`/collections/${id}`, { method: 'DELETE' }),
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
    const formData = new FormData();
    formData.append('file', file);
    formData.append('name', name);
    formData.append('collection_id', collectionId);
    formData.append('type', 'csv');
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
              {ds.row_count && (
                <span className="ml-auto text-xs text-slate-400">
                  {ds.row_count.toLocaleString()} rows
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

function UploadFileDialog({
  isOpen,
  onClose,
  onSuccess,
  collection,
}: {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  collection: Collection | null;
}) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [file, setFile] = useState<File | null>(null);
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleFileChange = (e: ChangeEvent<HTMLInputElement>) => {
    const selectedFile = e.target.files?.[0];
    if (selectedFile) {
      setFile(selectedFile);
      if (!name) {
        // Auto-fill name from filename
        setName(selectedFile.name.replace(/\.[^/.]+$/, ''));
      }
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!file || !collection) return;

    setError('');
    setIsLoading(true);
    try {
      await datasourceApi.upload(collection.id, file, name, description);
      onSuccess();
      onClose();
      setName('');
      setDescription('');
      setFile(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to upload file');
    } finally {
      setIsLoading(false);
    }
  };

  if (!isOpen || !collection) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white rounded-xl shadow-xl w-full max-w-md p-6">
        <h2 className="text-lg font-semibold text-slate-900 mb-4">
          Upload Data File to "{collection.name}"
        </h2>
        <form onSubmit={handleSubmit}>
          {error && (
            <div className="mb-4 p-3 rounded-lg bg-red-50 border border-red-200 text-red-600 text-sm">
              {error}
            </div>
          )}
          <div className="mb-4">
            <label className="block text-sm font-medium text-slate-700 mb-1">File</label>
            <div
              className="border-2 border-dashed border-slate-300 rounded-lg p-4 text-center cursor-pointer hover:border-blue-400 transition-colors"
              onClick={() => fileInputRef.current?.click()}
            >
              {file ? (
                <div className="text-sm text-slate-700">
                  <span className="font-medium">{file.name}</span>
                  <span className="text-slate-400 ml-2">
                    ({(file.size / 1024 / 1024).toFixed(2)} MB)
                  </span>
                </div>
              ) : (
                <div className="text-sm text-slate-500">
                  Click to select a CSV file
                </div>
              )}
              <input
                ref={fileInputRef}
                type="file"
                accept=".csv"
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
              onClick={onClose}
              className="px-4 py-2 text-sm text-slate-700 hover:bg-slate-100 rounded-lg transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isLoading || !file}
              className="px-4 py-2 text-sm bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50 transition-colors"
            >
              {isLoading ? 'Uploading...' : 'Upload'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

// Main DataSourcePanel Component
export default function DataSourcePanel({ onSelect }: DataSourcePanelProps) {
  const [collections, setCollections] = useState<Collection[]>([]);
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());
  const [datasourcesByCollection, setDatasourcesByCollection] = useState<Record<string, Datasource[]>>({});
  const [selectedCollectionId, setSelectedCollectionId] = useState<string | null>(null);
  const [selectedDatasourceId, setSelectedDatasourceId] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null);

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

  useEffect(() => {
    loadCollections();
  }, [loadCollections]);

  // Load datasources for a collection
  const loadDatasources = async (collectionId: string) => {
    try {
      const datasources = await datasourceApi.listByCollection(collectionId);
      setDatasourcesByCollection((prev) => ({ ...prev, [collectionId]: datasources }));
    } catch (error) {
      console.error('Failed to load datasources:', error);
    }
  };

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
    if (!confirm(`Delete collection "${collection.name}"? This will also delete all data sources in it.`)) return;
    try {
      await collectionApi.delete(collection.id);
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
            onClick={() => setCreateCollectionOpen(true)}
            className="p-1.5 text-slate-500 hover:text-slate-700 hover:bg-slate-100 rounded transition-colors"
            title="New Collection"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 4v16m8-8H4" />
            </svg>
          </button>
          <button
            onClick={loadCollections}
            className="p-1.5 text-slate-500 hover:text-slate-700 hover:bg-slate-100 rounded transition-colors"
            title="Refresh"
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
        onSuccess={() => {
          if (uploadCollection) {
            loadDatasources(uploadCollection.id);
          }
        }}
        collection={uploadCollection}
      />
    </div>
  );
}

