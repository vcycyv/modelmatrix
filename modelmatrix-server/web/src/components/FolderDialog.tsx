import { useState, FormEvent } from 'react';
import Dialog, { FormField, Input, Textarea, Button } from './Dialog';
import { folderApi, Folder } from '../lib/api';

interface FolderDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  parentId?: string;
  folder?: Folder; // If provided, we're editing
}

export default function FolderDialog({ isOpen, onClose, onSuccess, parentId, folder }: FolderDialogProps) {
  const [name, setName] = useState(folder?.name || '');
  const [description, setDescription] = useState(folder?.description || '');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const isEditing = !!folder;

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setIsLoading(true);

    try {
      if (isEditing) {
        await folderApi.update(folder.id, { name, description });
      } else {
        await folderApi.create({ name, description, parent_id: parentId });
      }
      onSuccess();
      onClose();
      // Reset form
      setName('');
      setDescription('');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Operation failed');
    } finally {
      setIsLoading(false);
    }
  };

  const handleClose = () => {
    setName(folder?.name || '');
    setDescription(folder?.description || '');
    setError('');
    onClose();
  };

  return (
    <Dialog isOpen={isOpen} onClose={handleClose} title={isEditing ? 'Edit Folder' : 'Create Folder'}>
      <form onSubmit={handleSubmit}>
        {error && (
          <div className="mb-4 p-3 rounded-lg bg-red-50 border border-red-200 text-red-600 text-sm">
            {error}
          </div>
        )}

        <FormField label="Name">
          <Input
            value={name}
            onChange={setName}
            placeholder="Enter folder name"
            required
            autoFocus
          />
        </FormField>

        <FormField label="Description">
          <Textarea
            value={description}
            onChange={setDescription}
            placeholder="Enter description (optional)"
          />
        </FormField>

        <div className="flex justify-end space-x-3 mt-6">
          <Button variant="secondary" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" loading={isLoading}>
            {isEditing ? 'Save Changes' : 'Create Folder'}
          </Button>
        </div>
      </form>
    </Dialog>
  );
}

