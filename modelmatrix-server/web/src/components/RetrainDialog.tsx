import { useState, useEffect } from 'react';
import Dialog from './Dialog';
import { modelApi, datasourceApi, collectionApi, Model, Datasource, Collection, RetrainRequest } from '../lib/api';

interface RetrainDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: (buildId?: string) => void;
  model: Model | null;
}

export default function RetrainDialog({
  isOpen,
  onClose,
  onSuccess,
  model,
}: RetrainDialogProps) {
  const [name, setName] = useState('');
  const [datasourceId, setDatasourceId] = useState('');
  const [collectionId, setCollectionId] = useState('');
  const [collections, setCollections] = useState<Collection[]>([]);
  const [datasources, setDatasources] = useState<Datasource[]>([]);
  const [filteredDatasources, setFilteredDatasources] = useState<Datasource[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (isOpen) {
      loadCollections();
      loadDatasources();
      setName('');
      setDatasourceId('');
      setCollectionId('');
      setError('');
    }
  }, [isOpen]);

  useEffect(() => {
    if (collectionId) {
      setFilteredDatasources(datasources.filter((ds) => ds.collection_id === collectionId));
      setDatasourceId('');
    } else {
      setFilteredDatasources(datasources);
      setDatasourceId('');
    }
  }, [collectionId, datasources]);

  const loadCollections = async () => {
    try {
      const data = await collectionApi.list();
      setCollections(data);
    } catch (err) {
      console.error('Failed to load collections:', err);
    }
  };

  const loadDatasources = async () => {
    try {
      const data = await datasourceApi.list();
      setDatasources(data);
    } catch (err) {
      console.error('Failed to load datasources:', err);
    }
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!model) return;
    setIsLoading(true);
    setError('');
    try {
      const body: RetrainRequest = {};
      if (name.trim()) body.name = name.trim();
      if (datasourceId) body.datasource_id = datasourceId;
      const build = await modelApi.retrain(model.id, Object.keys(body).length > 0 ? body : undefined);
      onSuccess(build?.id);
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start retrain');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Dialog isOpen={isOpen} onClose={onClose} title="Retrain Model">
      {model && (
        <form onSubmit={handleSubmit} className="space-y-4">
          <p className="text-sm text-slate-600">
            Start a new training run using the same algorithm and target as <strong>{model.name}</strong>. The new build will update this model when it completes.
          </p>
          {error && (
            <div className="bg-red-50 border border-red-200 text-red-700 px-3 py-2 rounded text-sm">
              {error}
            </div>
          )}
          <div>
            <label className="block text-xs font-medium text-slate-600 mb-1">Build name (optional)</label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. Retrain 2024-03"
              className="w-full px-3 py-2 text-sm border border-slate-300 rounded-md focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
            />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-xs font-medium text-slate-600 mb-1">Collection (optional)</label>
              <select
                value={collectionId}
                onChange={(e) => setCollectionId(e.target.value)}
                className="w-full px-3 py-2 text-sm border border-slate-300 rounded-md focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
              >
                <option value="">Same as current</option>
                {collections.map((col) => (
                  <option key={col.id} value={col.id}>
                    {col.name}
                  </option>
                ))}
              </select>
            </div>
            <div>
              <label className="block text-xs font-medium text-slate-600 mb-1">Data source (optional)</label>
              <select
                value={datasourceId}
                onChange={(e) => setDatasourceId(e.target.value)}
                className="w-full px-3 py-2 text-sm border border-slate-300 rounded-md focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
              >
                <option value="">Same as current</option>
                {filteredDatasources.map((ds) => (
                  <option key={ds.id} value={ds.id}>
                    {ds.name}
                  </option>
                ))}
              </select>
            </div>
          </div>
          <div className="flex justify-end gap-2 pt-2 border-t border-slate-100">
            <button
              type="button"
              onClick={onClose}
              className="px-4 py-2 text-sm font-medium text-slate-600 hover:bg-slate-100 rounded-md transition-colors"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isLoading}
              className="px-4 py-2 text-sm font-medium text-white bg-violet-600 rounded-md hover:bg-violet-700 disabled:opacity-50 transition-colors"
            >
              {isLoading ? 'Starting…' : 'Start retrain'}
            </button>
          </div>
        </form>
      )}
    </Dialog>
  );
}
