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

// Hyperparameter definitions per algorithm
interface HyperparamDef {
  key: string;
  label: string;
  type: 'number' | 'select';
  default: number | string;
  min?: number;
  max?: number;
  step?: number;
  options?: { value: string; label: string }[];
  tooltip?: string;
}

const HYPERPARAMETERS: Record<string, HyperparamDef[]> = {
  random_forest: [
    { key: 'n_estimators', label: 'Number of Trees', type: 'number', default: 100, min: 10, max: 500, step: 10, tooltip: 'Number of trees in the forest' },
    { key: 'max_depth', label: 'Max Depth', type: 'number', default: 10, min: 1, max: 50, step: 1, tooltip: 'Maximum depth of each tree' },
    { key: 'min_samples_split', label: 'Min Samples Split', type: 'number', default: 2, min: 2, max: 20, step: 1, tooltip: 'Minimum samples to split a node' },
    { key: 'min_samples_leaf', label: 'Min Samples Leaf', type: 'number', default: 1, min: 1, max: 10, step: 1, tooltip: 'Minimum samples in a leaf node' },
  ],
  decision_tree: [
    { key: 'max_depth', label: 'Max Depth', type: 'number', default: 10, min: 1, max: 50, step: 1, tooltip: 'Maximum depth of the tree' },
    { key: 'min_samples_split', label: 'Min Samples Split', type: 'number', default: 2, min: 2, max: 20, step: 1, tooltip: 'Minimum samples to split a node' },
    { key: 'min_samples_leaf', label: 'Min Samples Leaf', type: 'number', default: 1, min: 1, max: 10, step: 1, tooltip: 'Minimum samples in a leaf node' },
    { key: 'criterion', label: 'Split Criterion', type: 'select', default: 'gini', options: [
      { value: 'gini', label: 'Gini Impurity' },
      { value: 'entropy', label: 'Entropy' },
    ], tooltip: 'Function to measure split quality' },
  ],
  xgboost: [
    { key: 'n_estimators', label: 'Number of Trees', type: 'number', default: 100, min: 10, max: 500, step: 10, tooltip: 'Number of boosting rounds' },
    { key: 'max_depth', label: 'Max Depth', type: 'number', default: 6, min: 1, max: 20, step: 1, tooltip: 'Maximum tree depth' },
    { key: 'learning_rate', label: 'Learning Rate', type: 'number', default: 0.1, min: 0.01, max: 1, step: 0.01, tooltip: 'Boosting learning rate (eta)' },
    { key: 'min_child_weight', label: 'Min Child Weight', type: 'number', default: 1, min: 1, max: 10, step: 1, tooltip: 'Minimum sum of instance weight in child' },
  ],
  linear_regression: [
    { key: 'regularization', label: 'Regularization', type: 'select', default: 'none', options: [
      { value: 'none', label: 'None (OLS)' },
      { value: 'ridge', label: 'Ridge (L2)' },
      { value: 'lasso', label: 'Lasso (L1)' },
    ], tooltip: 'Type of regularization' },
    { key: 'alpha', label: 'Alpha (Regularization Strength)', type: 'number', default: 1.0, min: 0.01, max: 10, step: 0.1, tooltip: 'Regularization strength (only for Ridge/Lasso)' },
  ],
  polynomial_regression: [
    { key: 'degree', label: 'Polynomial Degree', type: 'number', default: 2, min: 2, max: 5, step: 1, tooltip: 'Degree of polynomial features' },
    { key: 'regularization', label: 'Regularization', type: 'select', default: 'ridge', options: [
      { value: 'none', label: 'None' },
      { value: 'ridge', label: 'Ridge (L2)' },
    ], tooltip: 'Ridge helps prevent overfitting' },
    { key: 'alpha', label: 'Alpha', type: 'number', default: 1.0, min: 0.01, max: 10, step: 0.1, tooltip: 'Regularization strength' },
  ],
  kmeans: [
    { key: 'n_clusters', label: 'Number of Clusters', type: 'number', default: 3, min: 2, max: 20, step: 1, tooltip: 'Number of clusters to form' },
    { key: 'max_iter', label: 'Max Iterations', type: 'number', default: 300, min: 100, max: 1000, step: 50, tooltip: 'Maximum iterations for a single run' },
    { key: 'n_init', label: 'Number of Initializations', type: 'number', default: 10, min: 1, max: 30, step: 1, tooltip: 'Number of times to run with different centroid seeds' },
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
  const [hyperparameters, setHyperparameters] = useState<Record<string, number | string>>({});
  const [showHyperparams, setShowHyperparams] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [showColumns, setShowColumns] = useState(false);

  // Initialize hyperparameters when algorithm changes
  useEffect(() => {
    const params = HYPERPARAMETERS[algorithm] || [];
    const defaults: Record<string, number | string> = {};
    params.forEach(p => {
      defaults[p.key] = p.default;
    });
    setHyperparameters(defaults);
  }, [algorithm]);

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

  // Auto-detect model type based on target column data type (+ cardinality for numeric columns)
  useEffect(() => {
    if (columns.length === 0) return;

    const targetColumn = columns.find(col => col.role === 'target');

    if (!targetColumn) {
      setModelType('clustering');
      return;
    }

    const dataType = targetColumn.data_type?.toLowerCase() || '';

    if (dataType === 'boolean' || dataType === 'bool') {
      setModelType('classification');
      return;
    }

    if (!dataType.includes('int') && !dataType.includes('float') && dataType !== 'float64' && dataType !== 'int64') {
      // Non-numeric → Classification
      setModelType('classification');
      return;
    }

    // Numeric target: peek at unique values from the data preview to decide
    // ≤ 20 distinct values → likely discrete labels → Classification; otherwise Regression
    if (datasourceId) {
      datasourceApi.getPreview(datasourceId, 500).then((preview) => {
        const targetName = targetColumn.name;
        const uniqueVals = new Set(
          preview.rows
            .map((row) => row[targetName])
            .filter((v) => v !== null && v !== undefined && v !== '')
        );
        if (uniqueVals.size <= 20) {
          setModelType('classification');
        } else {
          setModelType('regression');
        }
      }).catch(() => {
        // Preview unavailable; fall back to regression for numeric
        setModelType('regression');
      });
    } else {
      setModelType('regression');
    }
  }, [columns, datasourceId]);

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
          hyperparameters: hyperparameters,
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
    setHyperparameters({});
    setShowHyperparams(false);
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
          <div className="min-w-0">
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
          <div className="min-w-0">
            <label className="block text-xs font-medium text-slate-600 mb-1">
              Data Source <span className="text-red-500">*</span>
            </label>
            <div className="flex gap-1 min-w-0">
              <select
                value={datasourceId}
                onChange={(e) => setDatasourceId(e.target.value)}
                className="min-w-0 flex-1 px-3 py-2 text-sm border border-slate-300 rounded-md focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
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
                  className="flex-shrink-0 px-2 py-1 text-xs text-blue-600 hover:bg-blue-50 rounded border border-slate-300 whitespace-nowrap"
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
          <div className="flex items-center justify-between mb-1">
            <label className="text-xs font-medium text-slate-600">
              Algorithm <span className="text-red-500">*</span>
            </label>
            {HYPERPARAMETERS[algorithm]?.length > 0 && (
              <button
                type="button"
                onClick={() => setShowHyperparams(!showHyperparams)}
                className="text-xs text-blue-600 hover:text-blue-700"
              >
                {showHyperparams ? 'Hide' : 'Show'} Hyperparameters
              </button>
            )}
          </div>
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

        {/* Hyperparameters (collapsible) */}
        {showHyperparams && HYPERPARAMETERS[algorithm]?.length > 0 && (
          <div className="bg-slate-50 rounded-lg p-3 border border-slate-200 space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-xs font-medium text-slate-700">Hyperparameters</span>
              <button
                type="button"
                onClick={() => {
                  const defaults: Record<string, number | string> = {};
                  HYPERPARAMETERS[algorithm].forEach(p => { defaults[p.key] = p.default; });
                  setHyperparameters(defaults);
                }}
                className="text-xs text-slate-500 hover:text-slate-700"
              >
                Reset to defaults
              </button>
            </div>
            <div className="grid grid-cols-2 gap-3">
              {HYPERPARAMETERS[algorithm].map((param) => (
                <div key={param.key}>
                  <label className="block text-xs text-slate-500 mb-1" title={param.tooltip}>
                    {param.label}
                    {param.tooltip && (
                      <span className="ml-1 text-slate-400 cursor-help" title={param.tooltip}>ⓘ</span>
                    )}
                  </label>
                  {param.type === 'number' ? (
                    <div className="flex items-center gap-2">
                      <input
                        type="number"
                        value={hyperparameters[param.key] ?? param.default}
                        onChange={(e) => setHyperparameters(prev => ({
                          ...prev,
                          [param.key]: param.step && param.step < 1 ? parseFloat(e.target.value) : parseInt(e.target.value, 10)
                        }))}
                        min={param.min}
                        max={param.max}
                        step={param.step}
                        className="flex-1 px-2 py-1.5 text-xs border border-slate-300 rounded focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
                      />
                      <span className="text-xs text-slate-400 w-16 text-right">
                        {param.min}–{param.max}
                      </span>
                    </div>
                  ) : (
                    <select
                      value={hyperparameters[param.key] ?? param.default}
                      onChange={(e) => setHyperparameters(prev => ({
                        ...prev,
                        [param.key]: e.target.value
                      }))}
                      className="w-full px-2 py-1.5 text-xs border border-slate-300 rounded focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
                    >
                      {param.options?.map((opt) => (
                        <option key={opt.value} value={opt.value}>{opt.label}</option>
                      ))}
                    </select>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

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
