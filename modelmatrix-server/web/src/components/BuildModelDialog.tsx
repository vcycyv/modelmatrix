import { useState, useEffect } from 'react';
import Dialog from './Dialog';
import { buildApi, datasourceApi, Datasource, Column } from '../lib/api';

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
  { value: 'classification', label: 'Classification', defaultAlgorithm: 'random_forest' },
  { value: 'regression', label: 'Regression', defaultAlgorithm: 'random_forest' },
  { value: 'clustering', label: 'Clustering', defaultAlgorithm: 'decision_tree' },
];

// Backend only supports: decision_tree, random_forest, xgboost
const ALGORITHMS = {
  classification: [
    { value: 'decision_tree', label: 'Decision Tree' },
    { value: 'random_forest', label: 'Random Forest' },
    { value: 'xgboost', label: 'XGBoost' },
  ],
  regression: [
    { value: 'decision_tree', label: 'Decision Tree' },
    { value: 'random_forest', label: 'Random Forest' },
    { value: 'xgboost', label: 'XGBoost' },
  ],
  clustering: [
    { value: 'decision_tree', label: 'Decision Tree' },
    { value: 'random_forest', label: 'Random Forest' },
    { value: 'xgboost', label: 'XGBoost' },
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
  const [datasourceId, setDatasourceId] = useState('');
  const [modelType, setModelType] = useState<'classification' | 'regression' | 'clustering'>('classification');
  const [algorithm, setAlgorithm] = useState('random_forest');
  const [trainTestSplit, setTrainTestSplit] = useState(0.8);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');
  const [showColumns, setShowColumns] = useState(false);

  const [datasources, setDatasources] = useState<Datasource[]>([]);
  const [columns, setColumns] = useState<Column[]>([]);
  const [loadingDatasources, setLoadingDatasources] = useState(false);
  const [loadingColumns, setLoadingColumns] = useState(false);

  useEffect(() => {
    if (isOpen) {
      loadDatasources();
    }
  }, [isOpen]);

  useEffect(() => {
    if (datasourceId) {
      loadColumns(datasourceId);
    } else {
      setColumns([]);
    }
  }, [datasourceId]);

  useEffect(() => {
    const typeConfig = MODEL_TYPES.find((t) => t.value === modelType);
    if (typeConfig) {
      setAlgorithm(typeConfig.defaultAlgorithm);
    }
  }, [modelType]);

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

        {/* Row 1: Name + Data Source */}
        <div className="grid grid-cols-2 gap-3">
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
              >
                <option value="">{loadingDatasources ? 'Loading...' : 'Select...'}</option>
                {datasources.map((ds) => (
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
          <label className="block text-xs font-medium text-slate-600 mb-1">
            Model Type <span className="text-red-500">*</span>
          </label>
          <div className="flex rounded-md border border-slate-300 overflow-hidden">
            {MODEL_TYPES.map((type) => (
              <button
                key={type.value}
                type="button"
                onClick={() => setModelType(type.value as typeof modelType)}
                className={`flex-1 px-3 py-2 text-sm font-medium transition-colors ${
                  modelType === type.value
                    ? 'bg-blue-600 text-white'
                    : 'bg-white text-slate-600 hover:bg-slate-50'
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
