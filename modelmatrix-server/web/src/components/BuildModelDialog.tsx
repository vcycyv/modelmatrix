import { useState, useEffect } from 'react';
import Dialog from './Dialog';
import { buildApi, datasourceApi, collectionApi, Datasource, Column, Collection } from '../lib/api';

interface BuildModelDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  projectId?: string;
  projectName?: string;
  folderId?: string;
  folderName?: string;
}

const MODEL_TYPES = [
  { value: 'classification', label: 'Classification', defaultAlgorithm: 'random_forest', enabled: true },
  { value: 'regression', label: 'Regression', defaultAlgorithm: 'linear_regression', enabled: true },
  { value: 'clustering', label: 'Clustering', defaultAlgorithm: 'kmeans', enabled: true },
];

// Algorithms available per model type
// Classification: tree-based classifiers
// Regression: linear models + tree-based regressors
// Clustering: requires separate implementation
const ALGORITHMS: Record<string, { value: string; label: string }[]> = {
  classification: [
    { value: 'decision_tree', label: 'Decision Tree' },
    { value: 'random_forest', label: 'Random Forest' },
    { value: 'xgboost', label: 'XGBoost' },
  ],
  regression: [
    { value: 'linear_regression', label: 'Linear Regression' },
    { value: 'polynomial_regression', label: 'Polynomial Regression' },
    { value: 'decision_tree', label: 'Decision Tree' },
    { value: 'random_forest', label: 'Random Forest' },
    { value: 'xgboost', label: 'XGBoost' },
  ],
  clustering: [
    { value: 'kmeans', label: 'K-Means' },
  ],
};

export default function BuildModelDialog({
  isOpen,
  onClose,
  onSuccess,
  projectId,
  projectName,
  folderId,
  folderName,
}: BuildModelDialogProps) {
  const isProjectContext = !!projectId;
  const contextName = isProjectContext ? projectName : folderName;
  const contextType = isProjectContext ? 'project' : 'folder';

  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [collectionId, setCollectionId] = useState('');
  const [datasourceId, setDatasourceId] = useState('');
  const [modelType, setModelType] = useState<'classification' | 'regression' | 'clustering'>('classification');
  const [algorithm, setAlgorithm] = useState('random_forest');
  const [trainTestSplit, setTrainTestSplit] = useState(0.8);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [showColumns, setShowColumns] = useState(false);

  const [collections, setCollections] = useState<Collection[]>([]);
  const [datasources, setDatasources] = useState<Datasource[]>([]);
  const [filteredDatasources, setFilteredDatasources] = useState<Datasource[]>([]);
  const [columns, setColumns] = useState<Column[]>([]);
  const [loadingCollections, setLoadingCollections] = useState(false);
  const [loadingDatasources, setLoadingDatasources] = useState(false);
  const [loadingColumns, setLoadingColumns] = useState(false);

  // Load collections when dialog opens
  useEffect(() => {
    if (isOpen) {
      loadCollections();
      loadDatasources();
    }
  }, [isOpen]);

  // Filter datasources when collection changes
  useEffect(() => {
    if (collectionId) {
      setFilteredDatasources(datasources.filter(ds => ds.collection_id === collectionId));
      setDatasourceId(''); // Reset datasource selection
      setColumns([]);
    } else {
      setFilteredDatasources([]);
      setDatasourceId('');
      setColumns([]);
    }
  }, [collectionId, datasources]);

  // Load columns when datasource changes
  useEffect(() => {
    if (datasourceId) {
      loadColumns(datasourceId);
    } else {
      setColumns([]);
    }
  }, [datasourceId]);

  // Auto-detect model type based on target column data type
  useEffect(() => {
    if (columns.length === 0) return;

    const targetColumn = columns.find(col => col.role === 'target');
    
    if (!targetColumn) {
      // No target column → Clustering
      setModelType('clustering');
    } else {
      // Determine model type based on target column data type
      const dataType = targetColumn.data_type?.toLowerCase() || '';
      
      if (dataType.includes('int') || dataType.includes('float') || dataType === 'float64' || dataType === 'int64') {
        // Numeric target → Regression (could also be classification for discrete integers)
        // Check if it looks like a categorical integer (e.g., 0/1 for binary classification)
        // For now, default numeric to regression; user can change if needed
        setModelType('regression');
      } else if (dataType === 'boolean' || dataType === 'bool') {
        // Boolean → Classification
        setModelType('classification');
      } else {
        // String/object/categorical → Classification
        setModelType('classification');
      }
    }
  }, [columns]);

  useEffect(() => {
    const typeConfig = MODEL_TYPES.find((t) => t.value === modelType);
    if (typeConfig) {
      setAlgorithm(typeConfig.defaultAlgorithm);
    }
  }, [modelType]);

  const loadCollections = async () => {
    setLoadingCollections(true);
    try {
      const data = await collectionApi.list();
      setCollections(data);
    } catch (err) {
      console.error('Failed to load collections:', err);
    } finally {
      setLoadingCollections(false);
    }
  };

  const loadDatasources = async () => {
    setLoadingDatasources(true);
    try {
      const data = await datasourceApi.list();
      setDatasources(data);
    } catch (err) {
      console.error('Failed to load datasources:', err);
    } finally {
      setLoadingDatasources(false);
    }
  };

  const loadColumns = async (dsId: string) => {
    setLoadingColumns(true);
    try {
      const data = await datasourceApi.getColumns(dsId);
      setColumns(data);
    } catch (err) {
      console.error('Failed to load columns:', err);
      setColumns([]);
    } finally {
      setLoadingColumns(false);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError('');

    try {
      // Create build with project_id or folder_id directly (no need for separate association call)
      await buildApi.create({
        name,
        description,
        datasource_id: datasourceId,
        project_id: projectId,
        folder_id: folderId,
        model_type: modelType,
        algorithm,
        parameters: {
          train_test_split: trainTestSplit,
        },
      });

      onSuccess();
      handleClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create build');
    } finally {
      setIsLoading(false);
    }
  };

  const handleClose = () => {
    setName('');
    setDescription('');
    setCollectionId('');
    setDatasourceId('');
    setModelType('classification');
    setAlgorithm('random_forest');
    setTrainTestSplit(0.8);
    setError('');
    setColumns([]);
    setShowColumns(false);
    onClose();
  };

  return (
    <Dialog isOpen={isOpen} onClose={handleClose} title="Build Model">
      <form onSubmit={handleSubmit} className="space-y-4">
        {/* Context badge */}
        <div className="inline-flex items-center px-2.5 py-1 bg-slate-100 rounded-full text-xs text-slate-600">
          <span className="capitalize">{contextType}:</span>
          <span className="ml-1 font-medium text-slate-800">{contextName}</span>
        </div>

        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 px-3 py-2 rounded text-sm">
            {error}
          </div>
        )}

        {/* Row 1: Name */}
        <div>
          <label className="block text-xs font-medium text-slate-600 mb-1">
            Build Name <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g., forest_v1"
            className="w-full px-3 py-2 text-sm border border-slate-300 rounded-md focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
            required
          />
        </div>

        {/* Row 2: Collection + Data Source */}
        <div className="grid grid-cols-2 gap-3">
          <div>
            <label className="block text-xs font-medium text-slate-600 mb-1">
              Collection <span className="text-red-500">*</span>
            </label>
            <select
              value={collectionId}
              onChange={(e) => setCollectionId(e.target.value)}
              className="w-full px-3 py-2 text-sm border border-slate-300 rounded-md focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
              required
            >
              <option value="">{loadingCollections ? 'Loading...' : 'Select collection...'}</option>
              {collections.map((col) => (
                <option key={col.id} value={col.id}>
                  {col.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs font-medium text-slate-600 mb-1">
              Data Source <span className="text-red-500">*</span>
            </label>
            <div className="flex gap-1">
              <select
                value={datasourceId}
                onChange={(e) => setDatasourceId(e.target.value)}
                className="flex-1 px-3 py-2 text-sm border border-slate-300 rounded-md focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
                required
                disabled={!collectionId}
              >
                <option value="">
                  {!collectionId ? 'Select collection first' : loadingDatasources ? 'Loading...' : 'Select datasource...'}
                </option>
                {filteredDatasources.map((ds) => (
                  <option key={ds.id} value={ds.id}>
                    {ds.name}
                  </option>
                ))}
              </select>
              {datasourceId && (
                <button
                  type="button"
                  onClick={() => setShowColumns(!showColumns)}
                  className="px-2 py-1 text-xs text-blue-600 hover:bg-blue-50 rounded border border-slate-300"
                  title="View columns"
                >
                  {loadingColumns ? '...' : `${columns.length} cols`}
                </button>
              )}
            </div>
          </div>
        </div>

        {/* Collapsible columns preview */}
        {showColumns && columns.length > 0 && (
          <div className="bg-slate-50 rounded p-2 border border-slate-200 max-h-24 overflow-y-auto">
            <div className="flex flex-wrap gap-1">
              {columns.map((col) => (
                <span
                  key={col.id}
                  className={`px-1.5 py-0.5 rounded text-xs ${
                    col.role === 'target'
                      ? 'bg-green-100 text-green-700'
                      : col.role === 'exclude'
                      ? 'bg-gray-100 text-gray-400 line-through'
                      : 'bg-blue-50 text-blue-700'
                  }`}
                >
                  {col.name}
                </span>
              ))}
            </div>
          </div>
        )}

        {/* Model Type - full width */}
        <div>
          <div className="flex items-center justify-between mb-1">
            <label className="text-xs font-medium text-slate-600">
              Model Type <span className="text-red-500">*</span>
            </label>
            {columns.length > 0 && (
              <span className="text-xs text-slate-400">
                {(() => {
                  const targetCol = columns.find(c => c.role === 'target');
                  if (!targetCol) return 'No target → Clustering';
                  return `Target: ${targetCol.name} (${targetCol.data_type})`;
                })()}
              </span>
            )}
          </div>
          <div className="flex rounded-md border border-slate-300 overflow-hidden">
            {MODEL_TYPES.map((type) => (
              <button
                key={type.value}
                type="button"
                onClick={() => type.enabled && setModelType(type.value as typeof modelType)}
                disabled={!type.enabled}
                title={!type.enabled ? 'Coming soon' : undefined}
                className={`flex-1 px-3 py-2 text-sm font-medium transition-colors ${
                  modelType === type.value
                    ? 'bg-blue-600 text-white'
                    : type.enabled
                    ? 'bg-white text-slate-600 hover:bg-slate-50'
                    : 'bg-slate-100 text-slate-400 cursor-not-allowed'
                }`}
              >
                {type.label}
              </button>
            ))}
          </div>
        </div>

        {/* Algorithm */}
        <div>
          <label className="block text-xs font-medium text-slate-600 mb-1">
            Algorithm <span className="text-red-500">*</span>
          </label>
          <select
            value={algorithm}
            onChange={(e) => setAlgorithm(e.target.value)}
            className="w-full px-3 py-2 text-sm border border-slate-300 rounded-md focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
            required
          >
            {ALGORITHMS[modelType].map((alg) => (
              <option key={alg.value} value={alg.value}>
                {alg.label}
              </option>
            ))}
          </select>
        </div>

        {/* Row 3: Train/Test Split (compact) */}
        <div>
          <div className="flex items-center justify-between mb-1">
            <label className="text-xs font-medium text-slate-600">Train/Test Split</label>
            <span className="text-xs text-slate-500 font-mono">
              {Math.round(trainTestSplit * 100)}% / {Math.round((1 - trainTestSplit) * 100)}%
            </span>
          </div>
          <input
            type="range"
            min="0.5"
            max="0.95"
            step="0.05"
            value={trainTestSplit}
            onChange={(e) => setTrainTestSplit(parseFloat(e.target.value))}
            className="w-full h-1.5 accent-blue-600"
          />
        </div>

        {/* Description (optional, collapsed by default) */}
        <div>
          <label className="block text-xs font-medium text-slate-600 mb-1">
            Description <span className="text-slate-400">(optional)</span>
          </label>
          <input
            type="text"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Brief description..."
            className="w-full px-3 py-2 text-sm border border-slate-300 rounded-md focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
          />
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-2 pt-2 border-t border-slate-100">
          <button
            type="button"
            onClick={handleClose}
            className="px-4 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 rounded-md transition-colors"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={isLoading || !name || !datasourceId}
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isLoading ? 'Creating...' : 'Create Build'}
          </button>
        </div>
      </form>
    </Dialog>
  );
}
