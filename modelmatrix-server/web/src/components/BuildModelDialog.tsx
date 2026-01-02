import { useState, useEffect } from 'react';
import Dialog from './Dialog';
import { buildApi, datasourceApi, projectApi, folderApi, Datasource, Column } from '../lib/api';

interface BuildModelDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  // Either projectId OR folderId should be provided
  projectId?: string;
  projectName?: string;
  folderId?: string;
  folderName?: string;
}

const MODEL_TYPES = [
  { value: 'classification', label: 'Classification', defaultAlgorithm: 'random_forest' },
  { value: 'regression', label: 'Regression', defaultAlgorithm: 'gradient_boosting' },
  { value: 'clustering', label: 'Clustering', defaultAlgorithm: 'kmeans' },
];

const ALGORITHMS = {
  classification: [
    { value: 'random_forest', label: 'Random Forest' },
    { value: 'logistic_regression', label: 'Logistic Regression' },
    { value: 'xgboost', label: 'XGBoost' },
    { value: 'svm', label: 'Support Vector Machine' },
  ],
  regression: [
    { value: 'gradient_boosting', label: 'Gradient Boosting' },
    { value: 'random_forest', label: 'Random Forest' },
    { value: 'linear_regression', label: 'Linear Regression' },
    { value: 'xgboost', label: 'XGBoost' },
  ],
  clustering: [
    { value: 'kmeans', label: 'K-Means' },
    { value: 'dbscan', label: 'DBSCAN' },
    { value: 'hierarchical', label: 'Hierarchical' },
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
  // Determine context (project or folder)
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

  // Data sources and columns
  const [datasources, setDatasources] = useState<Datasource[]>([]);
  const [columns, setColumns] = useState<Column[]>([]);
  const [loadingDatasources, setLoadingDatasources] = useState(false);
  const [loadingColumns, setLoadingColumns] = useState(false);

  // Load datasources when dialog opens
  useEffect(() => {
    if (isOpen) {
      loadDatasources();
    }
  }, [isOpen]);

  // Load columns when datasource changes
  useEffect(() => {
    if (datasourceId) {
      loadColumns(datasourceId);
    } else {
      setColumns([]);
    }
  }, [datasourceId]);

  // Update algorithm when model type changes
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
      // Create the build
      const build = await buildApi.create({
        name,
        description,
        datasource_id: datasourceId,
        model_type: modelType,
        parameters: {
          algorithm,
          train_test_split: trainTestSplit,
        },
      });

      // Associate build with project or folder
      if (projectId) {
        await projectApi.addBuild(projectId, build.id);
      } else if (folderId) {
        await folderApi.addBuild(folderId, build.id);
      }

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
    onClose();
  };

  return (
    <Dialog isOpen={isOpen} onClose={handleClose} title="Build Model">
      <form onSubmit={handleSubmit} className="space-y-5">
        {/* Context info */}
        <div className="bg-slate-50 rounded-lg p-3 border border-slate-200">
          <p className="text-sm text-slate-600">
            Building model in {contextType}: <span className="font-medium text-slate-800">{contextName}</span>
          </p>
        </div>

        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-lg text-sm">
            {error}
          </div>
        )}

        {/* Build name */}
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1.5">
            Build Name <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g., Sales Predictor v1"
            className="w-full px-3.5 py-2.5 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-all"
            required
          />
        </div>

        {/* Description */}
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1.5">
            Description
          </label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Describe the purpose of this model build..."
            rows={2}
            className="w-full px-3.5 py-2.5 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-all resize-none"
          />
        </div>

        {/* Datasource selection */}
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1.5">
            Data Source <span className="text-red-500">*</span>
          </label>
          <select
            value={datasourceId}
            onChange={(e) => setDatasourceId(e.target.value)}
            className="w-full px-3.5 py-2.5 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-all"
            required
          >
            <option value="">
              {loadingDatasources ? 'Loading...' : 'Select a data source'}
            </option>
            {datasources.map((ds) => (
              <option key={ds.id} value={ds.id}>
                {ds.name} ({ds.type}) - {ds.row_count?.toLocaleString() || '?'} rows
              </option>
            ))}
          </select>
        </div>

        {/* Show columns preview if datasource selected */}
        {datasourceId && (
          <div className="bg-slate-50 rounded-lg p-3 border border-slate-200">
            <p className="text-sm font-medium text-slate-700 mb-2">Data Columns</p>
            {loadingColumns ? (
              <p className="text-sm text-slate-500">Loading columns...</p>
            ) : columns.length > 0 ? (
              <div className="max-h-32 overflow-y-auto">
                <div className="flex flex-wrap gap-1.5">
                  {columns.map((col) => (
                    <span
                      key={col.id}
                      className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                        col.role === 'target'
                          ? 'bg-green-100 text-green-800'
                          : col.role === 'exclude'
                          ? 'bg-gray-100 text-gray-500 line-through'
                          : 'bg-blue-100 text-blue-800'
                      }`}
                    >
                      {col.name}
                      <span className="ml-1 text-xs opacity-60">({col.data_type})</span>
                    </span>
                  ))}
                </div>
              </div>
            ) : (
              <p className="text-sm text-slate-500">No columns found</p>
            )}
          </div>
        )}

        {/* Model type */}
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1.5">
            Model Type <span className="text-red-500">*</span>
          </label>
          <div className="grid grid-cols-3 gap-2">
            {MODEL_TYPES.map((type) => (
              <button
                key={type.value}
                type="button"
                onClick={() => setModelType(type.value as typeof modelType)}
                className={`px-3 py-2 text-sm font-medium rounded-lg border transition-all ${
                  modelType === type.value
                    ? 'border-blue-500 bg-blue-50 text-blue-700'
                    : 'border-slate-300 bg-white text-slate-700 hover:bg-slate-50'
                }`}
              >
                {type.label}
              </button>
            ))}
          </div>
        </div>

        {/* Algorithm */}
        <div>
          <label className="block text-sm font-medium text-slate-700 mb-1.5">
            Algorithm <span className="text-red-500">*</span>
          </label>
          <select
            value={algorithm}
            onChange={(e) => setAlgorithm(e.target.value)}
            className="w-full px-3.5 py-2.5 border border-slate-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-blue-500 transition-all"
            required
          >
            {ALGORITHMS[modelType].map((alg) => (
              <option key={alg.value} value={alg.value}>
                {alg.label}
              </option>
            ))}
          </select>
        </div>

        {/* Training parameters */}
        <div className="bg-slate-50 rounded-lg p-4 border border-slate-200">
          <p className="text-sm font-medium text-slate-700 mb-3">Training Parameters</p>
          
          <div className="space-y-3">
            <div>
              <label className="block text-sm text-slate-600 mb-1">
                Train/Test Split: {Math.round(trainTestSplit * 100)}% / {Math.round((1 - trainTestSplit) * 100)}%
              </label>
              <input
                type="range"
                min="0.5"
                max="0.95"
                step="0.05"
                value={trainTestSplit}
                onChange={(e) => setTrainTestSplit(parseFloat(e.target.value))}
                className="w-full accent-blue-600"
              />
              <div className="flex justify-between text-xs text-slate-500">
                <span>50% Train</span>
                <span>95% Train</span>
              </div>
            </div>
          </div>
        </div>

        {/* Actions */}
        <div className="flex justify-end space-x-3 pt-2">
          <button
            type="button"
            onClick={handleClose}
            className="px-4 py-2.5 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 transition-colors"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={isLoading || !name || !datasourceId}
            className="px-4 py-2.5 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isLoading ? 'Creating...' : 'Create Build'}
          </button>
        </div>
      </form>
    </Dialog>
  );
}

