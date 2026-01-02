import { TreeNode } from './TreeView';
import { Folder, Project, ModelBuild, Model } from '../lib/api';

interface DetailPanelProps {
  node: TreeNode | null;
  onEdit: () => void;
  onDelete: () => void;
  onBuildModel?: () => void;
}

export default function DetailPanel({ node, onEdit, onDelete, onBuildModel }: DetailPanelProps) {
  if (!node) {
    return (
      <div className="h-full flex items-center justify-center text-slate-500">
        <div className="text-center">
          <svg className="w-16 h-16 mx-auto mb-4 text-slate-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <p className="text-lg font-medium">Select an item</p>
          <p className="text-sm text-slate-400 mt-1">Choose a folder, project, build, or model from the tree</p>
        </div>
      </div>
    );
  }

  return (
    <div className="bg-white rounded-lg shadow-sm border border-slate-200">
      {/* Header */}
      <div className="px-6 py-4 border-b border-slate-200 flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <NodeTypeIcon type={node.type} />
          <div>
            <h2 className="text-xl font-semibold text-slate-900">{node.name}</h2>
            <p className="text-sm text-slate-500 capitalize">{node.type}</p>
          </div>
        </div>
        <div className="flex items-center space-x-2">
          {(node.type === 'project' || node.type === 'folder') && onBuildModel && (
            <button
              onClick={onBuildModel}
              className="px-3 py-1.5 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md transition-colors flex items-center space-x-1"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
              </svg>
              <span>Build Model</span>
            </button>
          )}
          <button
            onClick={onEdit}
            className="px-3 py-1.5 text-sm font-medium text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-md transition-colors"
          >
            Edit
          </button>
          <button
            onClick={onDelete}
            className="px-3 py-1.5 text-sm font-medium text-red-600 bg-red-50 hover:bg-red-100 rounded-md transition-colors"
          >
            Delete
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="p-6">
        {node.type === 'folder' && <FolderDetails folder={node.data as Folder} />}
        {node.type === 'project' && <ProjectDetails project={node.data as Project} />}
        {node.type === 'build' && <BuildDetails build={node.data as ModelBuild} />}
        {node.type === 'model' && <ModelDetails model={node.data as Model} />}
      </div>
    </div>
  );
}

function NodeTypeIcon({ type }: { type: string }) {
  const iconClass = "w-10 h-10 p-2 rounded-lg";
  
  switch (type) {
    case 'folder':
      return (
        <div className={`${iconClass} bg-amber-100`}>
          <svg className="w-full h-full text-amber-600" fill="currentColor" viewBox="0 0 20 20">
            <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
          </svg>
        </div>
      );
    case 'project':
      return (
        <div className={`${iconClass} bg-blue-100`}>
          <svg className="w-full h-full text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
        </div>
      );
    case 'build':
      return (
        <div className={`${iconClass} bg-purple-100`}>
          <svg className="w-full h-full text-purple-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
          </svg>
        </div>
      );
    case 'model':
      return (
        <div className={`${iconClass} bg-green-100`}>
          <svg className="w-full h-full text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
          </svg>
        </div>
      );
    default:
      return null;
  }
}

function InfoRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="py-3 border-b border-slate-100 last:border-0">
      <dt className="text-sm font-medium text-slate-500">{label}</dt>
      <dd className="mt-1 text-sm text-slate-900">{value || '-'}</dd>
    </div>
  );
}

function StatusBadge({ status }: { status: string }) {
  const colors: Record<string, string> = {
    pending: 'bg-yellow-100 text-yellow-800',
    running: 'bg-blue-100 text-blue-800',
    completed: 'bg-green-100 text-green-800',
    failed: 'bg-red-100 text-red-800',
    cancelled: 'bg-slate-100 text-slate-800',
    draft: 'bg-slate-100 text-slate-800',
    active: 'bg-green-100 text-green-800',
    inactive: 'bg-yellow-100 text-yellow-800',
    archived: 'bg-slate-100 text-slate-800',
  };

  return (
    <span className={`inline-flex px-2 py-1 text-xs font-medium rounded-full ${colors[status] || 'bg-slate-100 text-slate-800'}`}>
      {status}
    </span>
  );
}

function FolderDetails({ folder }: { folder: Folder }) {
  return (
    <dl>
      <InfoRow label="Description" value={folder.description} />
      <InfoRow label="Path" value={<code className="text-xs bg-slate-100 px-2 py-1 rounded">{folder.path}</code>} />
      <InfoRow label="Depth" value={folder.depth} />
      <InfoRow label="Created By" value={folder.created_by} />
      <InfoRow label="Created At" value={new Date(folder.created_at).toLocaleString()} />
      <InfoRow label="Updated At" value={new Date(folder.updated_at).toLocaleString()} />
    </dl>
  );
}

function ProjectDetails({ project }: { project: Project }) {
  return (
    <dl>
      <InfoRow label="Description" value={project.description} />
      <InfoRow label="Created By" value={project.created_by} />
      <InfoRow label="Created At" value={new Date(project.created_at).toLocaleString()} />
      <InfoRow label="Updated At" value={new Date(project.updated_at).toLocaleString()} />
    </dl>
  );
}

function BuildDetails({ build }: { build: ModelBuild }) {
  return (
    <dl>
      <InfoRow label="Status" value={<StatusBadge status={build.status} />} />
      <InfoRow label="Description" value={build.description} />
      <InfoRow label="Model Type" value={build.model_type} />
      <InfoRow label="Datasource ID" value={<code className="text-xs bg-slate-100 px-2 py-1 rounded">{build.datasource_id}</code>} />
      {build.error_message && (
        <InfoRow label="Error" value={<span className="text-red-600">{build.error_message}</span>} />
      )}
      {build.metrics && Object.keys(build.metrics).length > 0 && (
        <InfoRow 
          label="Metrics" 
          value={
            <div className="space-y-1">
              {Object.entries(build.metrics).map(([key, value]) => (
                <div key={key} className="flex justify-between text-sm">
                  <span className="text-slate-500">{key}:</span>
                  <span className="font-mono">{typeof value === 'number' ? value.toFixed(4) : value}</span>
                </div>
              ))}
            </div>
          } 
        />
      )}
      <InfoRow label="Started At" value={build.started_at ? new Date(build.started_at).toLocaleString() : '-'} />
      <InfoRow label="Completed At" value={build.completed_at ? new Date(build.completed_at).toLocaleString() : '-'} />
      <InfoRow label="Created By" value={build.created_by} />
      <InfoRow label="Created At" value={new Date(build.created_at).toLocaleString()} />
    </dl>
  );
}

function ModelDetails({ model }: { model: Model }) {
  return (
    <dl>
      <InfoRow label="Status" value={<StatusBadge status={model.status} />} />
      <InfoRow label="Description" value={model.description} />
      <InfoRow label="Algorithm" value={model.algorithm} />
      <InfoRow label="Model Type" value={model.model_type} />
      <InfoRow label="Target Column" value={model.target_column} />
      <InfoRow label="Version" value={model.version} />
      {model.metrics && Object.keys(model.metrics).length > 0 && (
        <InfoRow 
          label="Metrics" 
          value={
            <div className="space-y-1">
              {Object.entries(model.metrics).map(([key, value]) => (
                <div key={key} className="flex justify-between text-sm">
                  <span className="text-slate-500">{key}:</span>
                  <span className="font-mono">{typeof value === 'number' ? value.toFixed(4) : value}</span>
                </div>
              ))}
            </div>
          } 
        />
      )}
      <InfoRow label="Build ID" value={<code className="text-xs bg-slate-100 px-2 py-1 rounded">{model.build_id}</code>} />
      <InfoRow label="Datasource ID" value={<code className="text-xs bg-slate-100 px-2 py-1 rounded">{model.datasource_id}</code>} />
      <InfoRow label="Created By" value={model.created_by} />
      <InfoRow label="Created At" value={new Date(model.created_at).toLocaleString()} />
    </dl>
  );
}

