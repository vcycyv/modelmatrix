import { useState, FormEvent } from 'react';
import Dialog, { FormField, Input, Textarea, Button } from './Dialog';
import { projectApi, Project } from '../lib/api';

interface ProjectDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onSuccess: () => void;
  folderId?: string;
  folderName?: string;
  project?: Project; // If provided, we're editing
}

export default function ProjectDialog({ isOpen, onClose, onSuccess, folderId, folderName, project }: ProjectDialogProps) {
  const [name, setName] = useState(project?.name || '');
  const [description, setDescription] = useState(project?.description || '');
  const [error, setError] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const isEditing = !!project;

  const handleSubmit = async (e: FormEvent) => {
    e.preventDefault();
    setError('');
    setIsLoading(true);

    try {
      if (isEditing) {
        await projectApi.update(project.id, { name, description });
      } else {
        await projectApi.create({ name, description, folder_id: folderId });
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
    setName(project?.name || '');
    setDescription(project?.description || '');
    setError('');
    onClose();
  };

  return (
    <Dialog isOpen={isOpen} onClose={handleClose} title={isEditing ? 'Edit Project' : 'Create Project'}>
      <form onSubmit={handleSubmit}>
        {error && (
          <div className="mb-4 p-3 rounded-lg bg-red-50 border border-red-200 text-red-600 text-sm">
            {error}
          </div>
        )}

        {!isEditing && folderId && (
          <div className="mb-4 p-3 rounded-lg bg-blue-50 border border-blue-200 text-blue-700 text-sm">
            Creating project in folder: <strong>{folderName || folderId}</strong>
          </div>
        )}

        <FormField label="Name">
          <Input
            value={name}
            onChange={setName}
            placeholder="Enter project name"
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
            {isEditing ? 'Save Changes' : 'Create Project'}
          </Button>
        </div>
      </form>
    </Dialog>
  );
}

