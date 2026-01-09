import { useState, useEffect } from 'react';
import { TreeNode } from './TreeView';
import { Folder, Project, ModelBuild, Model, ModelVariable, Collection, Datasource, Column, datasourceApi, modelApi } from '../lib/api';

// Extended node type that can include data items
export interface DataNode {
  id: string;
  name: string;
  type: 'collection' | 'datasource';
  data: Collection | Datasource;
}

interface DetailPanelProps {
  node: TreeNode | null;
  dataNode?: DataNode | null;
  onEdit: () => void;
  onDelete: () => void;
  onBuildModel?: () => void;
  onStartBuild?: () => void;
  onCancelBuild?: () => void;
  onDeleteDataNode?: () => void;
}

export default function DetailPanel({ node, dataNode, onEdit, onDelete, onBuildModel, onStartBuild, onCancelBuild, onDeleteDataNode }: DetailPanelProps) {
  // Determine which node to display (dataNode takes priority if present)
  const displayNode = dataNode || node;
  
  if (!displayNode) {
    return (
      <div className="h-full flex items-center justify-center text-slate-500">
        <div className="text-center">
          <svg className="w-16 h-16 mx-auto mb-4 text-slate-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          <p className="text-lg font-medium">Select an item</p>
          <p className="text-sm text-slate-400 mt-1">Choose an item from the explorer or data panel</p>
        </div>
      </div>
    );
  }

  // If it's a data node (collection or datasource), show details with actions
  if (dataNode) {
    return (
      <div className="bg-white rounded-lg shadow-sm border border-slate-200">
        {/* Header */}
        <div className="px-6 py-4 border-b border-slate-200 flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <NodeTypeIcon type={dataNode.type} />
            <div>
              <h2 className="text-xl font-semibold text-slate-900">{dataNode.name}</h2>
              <p className="text-sm text-slate-500 capitalize">{dataNode.type}</p>
            </div>
          </div>
          <div className="flex items-center space-x-2">
            {onDeleteDataNode && (
              <button
                onClick={onDeleteDataNode}
                className="px-3 py-1.5 text-sm font-medium text-red-600 bg-red-50 hover:bg-red-100 rounded-md transition-colors flex items-center space-x-1"
              >
                <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                </svg>
                <span>Delete</span>
              </button>
            )}
          </div>
        </div>

        {/* Content */}
        <div className="p-6">
          {dataNode.type === 'collection' && <CollectionDetails collection={dataNode.data as Collection} />}
          {dataNode.type === 'datasource' && <DatasourceDetails datasource={dataNode.data as Datasource} />}
        </div>
      </div>
    );
  }

  // At this point, node must exist (displayNode is set and dataNode is not)
  if (!node) return null;

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
          {node.type === 'build' && onStartBuild && (node.data as ModelBuild).status === 'pending' && (
            <button
              onClick={onStartBuild}
              className="px-3 py-1.5 text-sm font-medium text-white bg-green-600 hover:bg-green-700 rounded-md transition-colors flex items-center space-x-1"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span>Start Build</span>
            </button>
          )}
          {node.type === 'build' && onCancelBuild && (node.data as ModelBuild).status === 'running' && (
            <button
              onClick={onCancelBuild}
              className="px-3 py-1.5 text-sm font-medium text-white bg-orange-600 hover:bg-orange-700 rounded-md transition-colors flex items-center space-x-1"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
              <span>Cancel Build</span>
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
    case 'collection':
      return (
        <div className={`${iconClass} bg-indigo-100`}>
          <svg className="w-full h-full text-indigo-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
        </div>
      );
    case 'datasource':
      return (
        <div className={`${iconClass} bg-emerald-100`}>
          <svg className="w-full h-full text-emerald-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4" />
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
  const [variables, setVariables] = useState<ModelVariable[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [showVariables, setShowVariables] = useState(false);

  // Load variables when model changes or when showVariables is toggled on
  useEffect(() => {
    if (!showVariables) return;
    
    const loadVariables = async () => {
      setIsLoading(true);
      try {
        const detail = await modelApi.getDetail(model.id);
        // Sort by importance (descending), nulls last
        const sorted = [...detail.variables].sort((a, b) => {
          // Target variable always last
          if (a.role === 'target') return 1;
          if (b.role === 'target') return -1;
          // Sort by importance descending
          const impA = a.importance ?? -1;
          const impB = b.importance ?? -1;
          return impB - impA;
        });
        setVariables(sorted);
      } catch (err) {
        console.error('Failed to load model variables:', err);
      } finally {
        setIsLoading(false);
      }
    };
    
    loadVariables();
  }, [model.id, showVariables]);

  // Reset when model changes
  useEffect(() => {
    setVariables([]);
    setShowVariables(false);
  }, [model.id]);

  return (
    <div className="space-y-6">
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

      {/* Model Variables Section */}
      <div className="border-t border-slate-200 pt-4">
        <button
          onClick={() => setShowVariables(!showVariables)}
          className="flex items-center justify-between w-full text-left"
        >
          <h3 className="text-sm font-semibold text-slate-700">Model Variables</h3>
          <svg
            className={`w-5 h-5 text-slate-400 transition-transform ${showVariables ? 'rotate-180' : ''}`}
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </button>

        {showVariables && (
          <div className="mt-3">
            {isLoading ? (
              <div className="flex items-center justify-center py-4">
                <svg className="animate-spin h-5 w-5 text-blue-600" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
                <span className="ml-2 text-sm text-slate-500">Loading variables...</span>
              </div>
            ) : variables.length > 0 ? (
              <div className="overflow-x-auto">
                <table className="min-w-full text-sm">
                  <thead>
                    <tr className="border-b border-slate-200">
                      <th className="text-left py-2 pr-4 font-medium text-slate-600">Variable</th>
                      <th className="text-left py-2 pr-4 font-medium text-slate-600">Role</th>
                      <th className="text-right py-2 font-medium text-slate-600">Importance</th>
                    </tr>
                  </thead>
                  <tbody>
                    {variables.map((variable) => (
                      <tr key={variable.id} className="border-b border-slate-100 last:border-0">
                        <td className="py-2 pr-4">
                          <code className="text-xs bg-slate-100 px-1.5 py-0.5 rounded">{variable.name}</code>
                        </td>
                        <td className="py-2 pr-4">
                          <span className={`inline-flex items-center px-2 py-0.5 rounded text-xs font-medium ${
                            variable.role === 'target' 
                              ? 'bg-emerald-100 text-emerald-700' 
                              : 'bg-blue-100 text-blue-700'
                          }`}>
                            {variable.role}
                          </span>
                        </td>
                        <td className="py-2 text-right">
                          {variable.role === 'target' ? (
                            <span className="text-slate-400">—</span>
                          ) : variable.importance !== undefined && variable.importance !== null ? (
                            <div className="flex items-center justify-end space-x-2">
                              <div className="w-16 bg-slate-200 rounded-full h-1.5">
                                <div
                                  className={`h-1.5 rounded-full ${
                                    variable.importance === 0 
                                      ? 'bg-slate-300' 
                                      : variable.importance > 0.1 
                                        ? 'bg-blue-500' 
                                        : 'bg-blue-300'
                                  }`}
                                  style={{ width: `${Math.min(variable.importance * 100, 100)}%` }}
                                />
                              </div>
                              <span className={`font-mono text-xs ${
                                variable.importance === 0 ? 'text-slate-400' : 'text-slate-600'
                              }`}>
                                {variable.importance.toFixed(4)}
                              </span>
                              {variable.importance === 0 && (
                                <span className="text-xs text-orange-500" title="This feature was not used by the model">
                                  unused
                                </span>
                              )}
                            </div>
                          ) : (
                            <span className="text-slate-400">—</span>
                          )}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <p className="text-sm text-slate-500 py-2">No variables available</p>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

function CollectionDetails({ collection }: { collection: Collection }) {
  return (
    <dl>
      <InfoRow label="Description" value={collection.description} />
      <InfoRow label="ID" value={<code className="text-xs bg-slate-100 px-2 py-1 rounded">{collection.id}</code>} />
      <InfoRow label="Created By" value={collection.created_by} />
      <InfoRow label="Created At" value={new Date(collection.created_at).toLocaleString()} />
      <InfoRow label="Updated At" value={new Date(collection.updated_at).toLocaleString()} />
    </dl>
  );
}

// Column role options
const COLUMN_ROLES = [
  { value: 'input', label: 'Input', color: 'bg-blue-500' },
  { value: 'target', label: 'Target', color: 'bg-emerald-500' },
  { value: 'exclude', label: 'Exclude', color: 'bg-slate-400' },
] as const;

function DatasourceDetails({ datasource }: { datasource: Datasource }) {
  const [columns, setColumns] = useState<Column[]>([]);
  const [isLoadingColumns, setIsLoadingColumns] = useState(false);
  const [showColumns, setShowColumns] = useState(false);
  const [editingRoles, setEditingRoles] = useState(false);
  const [columnRoles, setColumnRoles] = useState<Record<string, string>>({});
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState('');

  // Load columns automatically when datasource changes
  useEffect(() => {
    // Reset state when datasource changes
    setColumns([]);
    setColumnRoles({});
    setShowColumns(false);
    setEditingRoles(false);
    setError('');
    
    // Load columns for target column display
    const loadColumnsForDatasource = async () => {
      setIsLoadingColumns(true);
      try {
        const cols = await datasourceApi.getColumns(datasource.id);
        setColumns(cols);
        setColumnRoles(Object.fromEntries(cols.map((c) => [c.id, c.role])));
      } catch (err) {
        console.error('Failed to load columns:', err);
      } finally {
        setIsLoadingColumns(false);
      }
    };
    
    loadColumnsForDatasource();
  }, [datasource.id]);

  // Find the target column
  const targetColumn = columns.find((c) => c.role === 'target');

  const loadColumns = async () => {
    setIsLoadingColumns(true);
    try {
      const cols = await datasourceApi.getColumns(datasource.id);
      setColumns(cols);
      setColumnRoles(Object.fromEntries(cols.map((c) => [c.id, c.role])));
    } catch (err) {
      console.error('Failed to load columns:', err);
    } finally {
      setIsLoadingColumns(false);
    }
  };

  const handleRoleChange = (columnId: string, role: string) => {
    setColumnRoles((prev) => ({ ...prev, [columnId]: role }));
  };

  const handleSaveRoles = async () => {
    // Validate: exactly one target
    const targetCount = Object.values(columnRoles).filter((r) => r === 'target').length;
    if (targetCount === 0) {
      setError('Please select exactly one target column');
      return;
    }
    if (targetCount > 1) {
      setError('Only one target column is allowed');
      return;
    }

    setError('');
    setIsSaving(true);
    try {
      await datasourceApi.updateColumnRoles(
        datasource.id,
        Object.entries(columnRoles).map(([column_id, role]) => ({ column_id, role }))
      );
      setEditingRoles(false);
      loadColumns(); // Refresh columns
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save');
    } finally {
      setIsSaving(false);
    }
  };

  const handleCancelEdit = () => {
    setColumnRoles(Object.fromEntries(columns.map((c) => [c.id, c.role])));
    setEditingRoles(false);
    setError('');
  };

  return (
    <div className="space-y-4">
      <dl>
        <InfoRow label="Description" value={datasource.description} />
        <InfoRow label="Type" value={
          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-800">
            {datasource.type || 'Unknown'}
          </span>
        } />
        {datasource.file_path && (
          <InfoRow label="File Path" value={<code className="text-xs bg-slate-100 px-2 py-1 rounded break-all">{datasource.file_path}</code>} />
        )}
        <InfoRow label="Columns" value={datasource.column_count?.toLocaleString() || '-'} />
        <InfoRow 
          label="Target Column" 
          value={
            isLoadingColumns ? (
              <span className="text-slate-400 text-sm">Loading...</span>
            ) : targetColumn ? (
              <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-100 text-emerald-800">
                {targetColumn.name}
              </span>
            ) : (
              <span className="text-slate-400 text-sm italic">Not set</span>
            )
          } 
        />
        <InfoRow label="ID" value={<code className="text-xs bg-slate-100 px-2 py-1 rounded">{datasource.id}</code>} />
        {datasource.collection_name && (
          <InfoRow label="Collection" value={datasource.collection_name} />
        )}
        {datasource.collection_id && (
          <InfoRow label="Collection ID" value={<code className="text-xs bg-slate-100 px-2 py-1 rounded">{datasource.collection_id}</code>} />
        )}
        <InfoRow label="Created By" value={datasource.created_by} />
        <InfoRow label="Created At" value={new Date(datasource.created_at).toLocaleString()} />
        <InfoRow label="Updated At" value={new Date(datasource.updated_at).toLocaleString()} />
      </dl>

      {/* Column roles section */}
      <div className="border-t border-slate-200 pt-4">
        <div className="flex items-center justify-between mb-3">
          <button
            onClick={() => setShowColumns(!showColumns)}
            className="flex items-center text-sm font-medium text-slate-700 hover:text-slate-900"
          >
            <svg
              className={`w-4 h-4 mr-1 transition-transform ${showColumns ? 'rotate-90' : ''}`}
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
            </svg>
            Column Roles ({columns.length || datasource.column_count || 0})
          </button>
          {showColumns && !editingRoles && columns.length > 0 && (
            <button
              onClick={() => setEditingRoles(true)}
              className="text-xs text-blue-600 hover:text-blue-700"
            >
              Edit Roles
            </button>
          )}
        </div>

        {showColumns && (
          <div>
            {isLoadingColumns ? (
              <div className="flex items-center text-sm text-slate-500 py-2">
                <svg className="animate-spin h-4 w-4 mr-2" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                </svg>
                Loading columns...
              </div>
            ) : columns.length === 0 ? (
              <div className="text-sm text-slate-500 py-2">No columns found</div>
            ) : (
              <>
                {error && (
                  <div className="mb-2 p-2 rounded bg-red-50 border border-red-200 text-red-600 text-xs">
                    {error}
                  </div>
                )}
                <div className="border border-slate-200 rounded-lg overflow-hidden">
                  <table className="w-full text-xs">
                    <thead className="bg-slate-50 sticky top-0">
                      <tr>
                        <th className="text-left px-3 py-2 font-medium text-slate-600">Column</th>
                        <th className="text-left px-3 py-2 font-medium text-slate-600">Type</th>
                        <th className="text-left px-3 py-2 font-medium text-slate-600">Role</th>
                      </tr>
                    </thead>
                    <tbody>
                      {columns.map((col, index) => (
                        <tr key={col.id} className={index % 2 === 0 ? 'bg-white' : 'bg-slate-50/50'}>
                          <td className="px-3 py-1.5">
                            <code className="text-xs bg-slate-100 px-1 py-0.5 rounded">{col.name}</code>
                          </td>
                          <td className="px-3 py-1.5 text-slate-500">{col.data_type}</td>
                          <td className="px-3 py-1.5">
                            {editingRoles ? (
                              <div className="flex space-x-1">
                                {COLUMN_ROLES.map((role) => (
                                  <button
                                    key={role.value}
                                    type="button"
                                    onClick={() => handleRoleChange(col.id, role.value)}
                                    className={`px-1.5 py-0.5 text-xs rounded transition-colors ${
                                      columnRoles[col.id] === role.value
                                        ? `${role.color} text-white`
                                        : 'bg-slate-100 text-slate-600 hover:bg-slate-200'
                                    }`}
                                  >
                                    {role.label}
                                  </button>
                                ))}
                              </div>
                            ) : (
                              <span
                                className={`inline-flex px-1.5 py-0.5 rounded text-white text-xs ${
                                  col.role === 'target'
                                    ? 'bg-emerald-500'
                                    : col.role === 'exclude'
                                    ? 'bg-slate-400'
                                    : 'bg-blue-500'
                                }`}
                              >
                                {col.role}
                              </span>
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
                {editingRoles && (
                  <div className="flex justify-end space-x-2 mt-2">
                    <button
                      onClick={handleCancelEdit}
                      className="px-3 py-1.5 text-xs text-slate-600 hover:bg-slate-100 rounded transition-colors"
                    >
                      Cancel
                    </button>
                    <button
                      onClick={handleSaveRoles}
                      disabled={isSaving}
                      className="px-3 py-1.5 text-xs bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50 transition-colors flex items-center"
                    >
                      {isSaving ? (
                        <>
                          <svg className="animate-spin h-3 w-3 mr-1" fill="none" viewBox="0 0 24 24">
                            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
                          </svg>
                          Saving...
                        </>
                      ) : (
                        'Save Roles'
                      )}
                    </button>
                  </div>
                )}
              </>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

