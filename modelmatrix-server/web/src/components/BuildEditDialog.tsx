import { useState, useEffect, FormEvent } from 'react';
import { buildApi, ModelBuild } from '../lib/api';

interface BuildEditDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  build?: ModelBuild;
}

export default function BuildEditDialog({ isOpen, onClose, onSuccess, build }: BuildEditDialogProps) {
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // Populate form when editing
  useEffect(() => {
    if (build) {
      setName(build.name || '');
      setDescription(build.description || '');
    } else {
      setName('');
      setDescription('');
    }
    setError(null);
  }, [build, isOpen]);

  if (!isOpen) return null;

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!build) return;

    setIsLoading(true);
    setError(null);

    try {
      await buildApi.update(build.id, {
        name: name || undefined,
        description: description || undefined,
      });
      onSuccess();
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update build');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />

      {/* Dialog */}
      <div className="relative bg-white rounded-lg shadow-xl w-full max-w-md mx-4">
        {/* Header */}
        <div className="px-6 py-4 border-b border-slate-200">
          <h3 className="text-lg font-semibold text-slate-900">
            Edit Build
          </h3>
          <p className="text-sm text-slate-500 mt-1">
            Update the build name and description
          </p>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit}>
          <div className="px-6 py-4 space-y-4">
            {error && (
              <div className="p-3 bg-red-50 border border-red-200 rounded-md text-sm text-red-600">
                {error}
              </div>
            )}

            <div>
              <label htmlFor="name" className="block text-sm font-medium text-slate-700 mb-1">
                Name
              </label>
              <input
                type="text"
                id="name"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                placeholder="Build name"
              />
            </div>

            <div>
              <label htmlFor="description" className="block text-sm font-medium text-slate-700 mb-1">
                Description
              </label>
              <textarea
                id="description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={3}
                className="w-full px-3 py-2 border border-slate-300 rounded-md shadow-sm focus:ring-2 focus:ring-blue-500 focus:border-blue-500"
                placeholder="Optional description"
              />
            </div>

            {/* Read-only info */}
            <div className="bg-slate-50 rounded-md p-3 space-y-2">
              <div className="flex justify-between text-sm">
                <span className="text-slate-500">Status:</span>
                <span className="font-medium text-slate-700">{build?.status}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-slate-500">Model Type:</span>
                <span className="font-medium text-slate-700">{build?.model_type || '-'}</span>
              </div>
              <div className="flex justify-between text-sm">
                <span className="text-slate-500">Created:</span>
                <span className="font-medium text-slate-700">
                  {build?.created_at ? new Date(build.created_at).toLocaleDateString() : '-'}
                </span>
              </div>
            </div>
          </div>

          {/* Footer */}
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
              disabled={isLoading}
              className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-md hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {isLoading ? 'Saving...' : 'Save Changes'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

