import { useState, useEffect, FormEvent } from 'react';
import Dialog from './Dialog';
import { modelApi, collectionApi, datasourceApi, Collection, Datasource, Model, ScoreRequest } from '../lib/api';

interface ScoreModelDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  model?: Model;
}

export default function ScoreModelDialog({ isOpen, onClose, onSuccess, model }: ScoreModelDialogProps) {
  const [collections, setCollections] = useState<Collection[]>([]);
  const [datasources, setDatasources] = useState<Datasource[]>([]);
  const [outputCollections, setOutputCollections] = useState<Collection[]>([]);
  
  const [selectedCollectionId, setSelectedCollectionId] = useState('');
  const [selectedDatasourceId, setSelectedDatasourceId] = useState('');
  const [outputCollectionId, setOutputCollectionId] = useState('');
  const [outputTableName, setOutputTableName] = useState('');
  
  const [isLoading, setIsLoading] = useState(false);
  const [isLoadingCollections, setIsLoadingCollections] = useState(false);
  const [isLoadingDatasources, setIsLoadingDatasources] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Load collections on mount
  useEffect(() => {
    if (isOpen) {
      loadCollections();
    }
  }, [isOpen]);

  // Load datasources when input collection changes
  useEffect(() => {
    if (selectedCollectionId) {
      loadDatasources(selectedCollectionId);
    } else {
      setDatasources([]);
      setSelectedDatasourceId('');
    }
  }, [selectedCollectionId]);

  // Reset form when dialog opens with a model
  useEffect(() => {
    if (model && isOpen) {
      setSelectedCollectionId('');
      setSelectedDatasourceId('');
      setOutputCollectionId('');
      setOutputTableName('');
      setError(null);
    }
  }, [model, isOpen]);

  const loadCollections = async () => {
    setIsLoadingCollections(true);
    try {
      const cols = await collectionApi.list();
      setCollections(cols);
      setOutputCollections(cols);
    } catch (err) {
      console.error('Failed to load collections:', err);
    } finally {
      setIsLoadingCollections(false);
    }
  };

  const loadDatasources = async (collectionId: string) => {
    setIsLoadingDatasources(true);
    try {
      const ds = await datasourceApi.list(collectionId);
      setDatasources(ds);
    } catch (err) {
      console.error('Failed to load datasources:', err);
    } finally {
      setIsLoadingDatasources(false);
    }
  };

  if (!isOpen || !model) return null;

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    
    if (!selectedDatasourceId) {
      setError('Please select an input datasource');
      return;
    }
    if (!outputCollectionId) {
      setError('Please select an output collection');
      return;
    }

    setIsLoading(true);
    setError(null);

    try {
      const req: ScoreRequest = {
        datasource_id: selectedDatasourceId,
        output_collection_id: outputCollectionId,
        output_table_name: outputTableName || undefined,
      };
      
      await modelApi.score(model.id, req);
      onSuccess();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start scoring');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Dialog isOpen={isOpen} onClose={onClose} title={`Score Data with ${model.name}`}>
      <form onSubmit={handleSubmit}>
        <div className="px-6 py-4 space-y-4">
          {error && (
            <div className="p-3 bg-red-50 border border-red-200 rounded-md text-sm text-red-600">
              {error}
            </div>
          )}

          {/* Model Info */}
          <div className="bg-slate-50 p-3 rounded-md">
            <div className="text-sm text-slate-600 space-y-1">
              <div><span className="font-medium">Algorithm:</span> {model.algorithm}</div>
              <div><span className="font-medium">Type:</span> {model.model_type}</div>
              <div><span className="font-medium">Target:</span> {model.target_column}</div>
            </div>
          </div>

          {/* Input Section */}
          <div className="border-t border-slate-200 pt-4">
            <h4 className="text-sm font-medium text-slate-700 mb-3">Input Data</h4>
            
            {/* Input Collection */}
            <div className="mb-3">
              <label htmlFor="inputCollection" className="block text-sm font-medium text-slate-700 mb-1">
                Collection
              </label>
              <select
                id="inputCollection"
                value={selectedCollectionId}
                onChange={(e) => setSelectedCollectionId(e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                disabled={isLoadingCollections}
              >
                <option value="">Select a collection...</option>
                {collections.map((col) => (
                  <option key={col.id} value={col.id}>
                    {col.name}
                  </option>
                ))}
              </select>
            </div>

            {/* Input Datasource */}
            <div>
              <label htmlFor="inputDatasource" className="block text-sm font-medium text-slate-700 mb-1">
                Datasource
              </label>
              <select
                id="inputDatasource"
                value={selectedDatasourceId}
                onChange={(e) => setSelectedDatasourceId(e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                disabled={!selectedCollectionId || isLoadingDatasources}
              >
                <option value="">
                  {!selectedCollectionId
                    ? 'Select a collection first...'
                    : isLoadingDatasources
                    ? 'Loading...'
                    : 'Select a datasource...'}
                </option>
                {datasources.map((ds) => (
                  <option key={ds.id} value={ds.id}>
                    {ds.name} ({ds.column_count || '?'} columns)
                  </option>
                ))}
              </select>
            </div>
          </div>

          {/* Output Section */}
          <div className="border-t border-slate-200 pt-4">
            <h4 className="text-sm font-medium text-slate-700 mb-3">Output</h4>
            
            {/* Output Collection */}
            <div className="mb-3">
              <label htmlFor="outputCollection" className="block text-sm font-medium text-slate-700 mb-1">
                Output Collection
              </label>
              <select
                id="outputCollection"
                value={outputCollectionId}
                onChange={(e) => setOutputCollectionId(e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
              >
                <option value="">Select output collection...</option>
                {outputCollections.map((col) => (
                  <option key={col.id} value={col.id}>
                    {col.name}
                  </option>
                ))}
              </select>
            </div>

            {/* Output Table Name */}
            <div>
              <label htmlFor="outputTableName" className="block text-sm font-medium text-slate-700 mb-1">
                Output Table Name
                <span className="text-slate-400 font-normal ml-1">(optional)</span>
              </label>
              <input
                type="text"
                id="outputTableName"
                value={outputTableName}
                onChange={(e) => setOutputTableName(e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                placeholder={`scored_${model.name}_${new Date().toISOString().slice(0, 10).replace(/-/g, '')}`}
              />
              <p className="mt-1 text-xs text-slate-500">
                Leave empty to auto-generate a name with timestamp
              </p>
            </div>
          </div>
        </div>

        <div className="px-6 py-4 border-t border-slate-200 flex justify-end space-x-3">
          <button
            type="button"
            onClick={onClose}
            className="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-md hover:bg-slate-50 transition-colors"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={isLoading || !selectedDatasourceId || !outputCollectionId}
            className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {isLoading ? 'Starting...' : 'Start Scoring'}
          </button>
        </div>
      </form>
    </Dialog>
  );
}
