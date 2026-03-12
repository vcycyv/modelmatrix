import { useState, useEffect } from 'react';
import { TreeNode } from './TreeView';
import { Folder, Project, ModelBuild, Model, ModelVariable, ModelFile, Collection, Datasource, Column, datasourceApi, modelApi, buildApi, versionApi, FileContentResponse, ModelVersion } from '../lib/api';
import PerformanceMonitorPanel from './PerformanceMonitorPanel';

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
  onScoreModel?: () => void;
  onRetrain?: () => void;
  onRefreshModel?: () => void;
  onNavigateToDatasource?: (datasource: Datasource) => void;
  onNavigateToBuild?: (build: ModelBuild) => void;
}

export default function DetailPanel({ node, dataNode, onEdit, onDelete, onBuildModel, onStartBuild, onCancelBuild, onDeleteDataNode, onScoreModel, onRetrain, onRefreshModel, onNavigateToDatasource, onNavigateToBuild }: DetailPanelProps) {
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
          {node.type === 'model' && onRetrain && (
            <button
              onClick={onRetrain}
              className="px-3 py-1.5 text-sm font-medium text-white bg-violet-600 hover:bg-violet-700 rounded-md transition-colors flex items-center space-x-1"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
              </svg>
              <span>Retrain</span>
            </button>
          )}
          {node.type === 'model' && onScoreModel && (
            <button
              onClick={onScoreModel}
              className="px-3 py-1.5 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-700 rounded-md transition-colors flex items-center space-x-1"
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
              </svg>
              <span>Score Data</span>
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
        {node.type === 'model' && <ModelDetails model={node.data as Model} onNavigateToDatasource={onNavigateToDatasource} onNavigateToBuild={onNavigateToBuild} onRefreshModel={onRefreshModel} />}
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

type ModelDetailTab = 'details' | 'variables' | 'files' | 'versions' | 'performance';

function ModelDetails({ model, onNavigateToDatasource, onNavigateToBuild, onRefreshModel }: { model: Model; onNavigateToDatasource?: (datasource: Datasource) => void; onNavigateToBuild?: (build: ModelBuild) => void; onRefreshModel?: () => void }) {
  const [activeTab, setActiveTab] = useState<ModelDetailTab>('details');
  const [variables, setVariables] = useState<ModelVariable[]>([]);
  const [files, setFiles] = useState<ModelFile[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [datasource, setDatasource] = useState<Datasource | null>(null);
  const [isLoadingDatasource, setIsLoadingDatasource] = useState(false);
  const [build, setBuild] = useState<ModelBuild | null>(null);
  const [isLoadingBuild, setIsLoadingBuild] = useState(false);
  
  // File viewer state
  const [viewingFile, setViewingFile] = useState<ModelFile | null>(null);
  const [fileContent, setFileContent] = useState<FileContentResponse | null>(null);
  const [isLoadingContent, setIsLoadingContent] = useState(false);
  const [contentError, setContentError] = useState<string | null>(null);

  // Versions tab state
  const [versions, setVersions] = useState<ModelVersion[]>([]);
  const [versionsTotal, setVersionsTotal] = useState(0);
  const [versionsPage, setVersionsPage] = useState(1);
  const [isLoadingVersions, setIsLoadingVersions] = useState(false);
  const [versionActionLoading, setVersionActionLoading] = useState<string | null>(null);

  // Load files when model changes
  useEffect(() => {
    const loadFiles = async () => {
      try {
        const detail = await modelApi.getDetail(model.id);
        setFiles(detail.files || []);
      } catch (err) {
        console.error('Failed to load model files:', err);
        setFiles([]);
      }
    };
    loadFiles();
  }, [model.id]);

  // Load datasource info when model changes
  useEffect(() => {
    const loadDatasource = async () => {
      if (!model.datasource_id) return;
      setIsLoadingDatasource(true);
      try {
        const ds = await datasourceApi.get(model.datasource_id);
        setDatasource(ds);
      } catch (err) {
        console.error('Failed to load datasource:', err);
        setDatasource(null);
      } finally {
        setIsLoadingDatasource(false);
      }
    };
    loadDatasource();
  }, [model.id, model.datasource_id]);

  // Load build info when model changes
  useEffect(() => {
    const loadBuild = async () => {
      if (!model.build_id) return;
      setIsLoadingBuild(true);
      try {
        const b = await buildApi.get(model.build_id);
        setBuild(b);
      } catch (err) {
        console.error('Failed to load build:', err);
        setBuild(null);
      } finally {
        setIsLoadingBuild(false);
      }
    };
    loadBuild();
  }, [model.id, model.build_id]);

  // Load variables when switching to variables tab
  useEffect(() => {
    if (activeTab !== 'variables') return;
    if (variables.length > 0) return; // Already loaded
    
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
  }, [model.id, activeTab, variables.length]);

  // Load versions when switching to versions tab
  useEffect(() => {
    if (activeTab !== 'versions') return;
    const loadVersions = async () => {
      setIsLoadingVersions(true);
      try {
        const res = await versionApi.list(model.id, { page: versionsPage, page_size: 10 });
        setVersions(res.versions || []);
        setVersionsTotal(res.total ?? 0);
      } catch (err) {
        console.error('Failed to load versions:', err);
        setVersions([]);
        setVersionsTotal(0);
      } finally {
        setIsLoadingVersions(false);
      }
    };
    loadVersions();
  }, [model.id, activeTab, versionsPage]);

  // Reset when model changes
  useEffect(() => {
    setVariables([]);
    setActiveTab('details');
    setFiles([]);
    setViewingFile(null);
    setFileContent(null);
    setContentError(null);
    setVersions([]);
    setVersionsTotal(0);
    setVersionsPage(1);
  }, [model.id]);

  // Function to check if a file is viewable (text-based)
  const isTextFile = (file: ModelFile) => {
    const textTypes = ['training_code', 'metadata', 'feature_names'];
    if (textTypes.includes(file.file_type)) return true;
    
    const textExtensions = ['.py', '.txt', '.json', '.yaml', '.yml', '.md', '.csv', '.log', '.xml', '.html', '.css', '.js', '.ts', '.sql', '.sh', '.r'];
    const ext = file.file_name.toLowerCase().substring(file.file_name.lastIndexOf('.'));
    return textExtensions.includes(ext);
  };

  // Function to load file content
  const handleViewFile = async (file: ModelFile) => {
    setViewingFile(file);
    setIsLoadingContent(true);
    setContentError(null);
    setFileContent(null);
    
    try {
      const content = await modelApi.getFileContent(model.id, file.id);
      setFileContent(content);
    } catch (err) {
      console.error('Failed to load file content:', err);
      setContentError(err instanceof Error ? err.message : 'Failed to load file content');
    } finally {
      setIsLoadingContent(false);
    }
  };

  // Function to close file viewer
  const handleCloseViewer = () => {
    setViewingFile(null);
    setFileContent(null);
    setContentError(null);
  };

  // Get syntax highlighting language from file name
  const getLanguage = (fileName: string): string => {
    const ext = fileName.toLowerCase().substring(fileName.lastIndexOf('.'));
    const langMap: Record<string, string> = {
      '.py': 'python',
      '.js': 'javascript',
      '.ts': 'typescript',
      '.json': 'json',
      '.yaml': 'yaml',
      '.yml': 'yaml',
      '.md': 'markdown',
      '.sql': 'sql',
      '.html': 'html',
      '.css': 'css',
      '.sh': 'bash',
      '.r': 'r',
    };
    return langMap[ext] || 'text';
  };

  // Find training code file
  const trainingCodeFile = files.find(f => f.file_type === 'training_code');
  const modelFile = files.find(f => f.file_type === 'model');

  const tabs: { id: ModelDetailTab; label: string; icon: React.ReactNode }[] = [
    {
      id: 'details',
      label: 'Details',
      icon: (
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      ),
    },
    {
      id: 'variables',
      label: 'Variables',
      icon: (
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
        </svg>
      ),
    },
    {
      id: 'files',
      label: 'Files',
      icon: (
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
        </svg>
      ),
    },
    {
      id: 'versions',
      label: 'Versions',
      icon: (
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
        </svg>
      ),
    },
    {
      id: 'performance',
      label: 'Performance',
      icon: (
        <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
        </svg>
      ),
    },
  ];

  return (
    <div className="space-y-4">
      {/* Tab Navigation */}
      <div className="border-b border-slate-200">
        <nav className="-mb-px flex space-x-4" aria-label="Tabs">
          {tabs.map((tab) => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={`group inline-flex items-center py-3 px-1 border-b-2 font-medium text-sm transition-colors ${
                activeTab === tab.id
                  ? 'border-blue-500 text-blue-600'
                  : 'border-transparent text-slate-500 hover:text-slate-700 hover:border-slate-300'
              }`}
            >
              <span className={`mr-2 ${activeTab === tab.id ? 'text-blue-500' : 'text-slate-400 group-hover:text-slate-500'}`}>
                {tab.icon}
              </span>
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="pt-2">
        {/* Details Tab */}
        {activeTab === 'details' && (
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
            <InfoRow label="Build" value={
              isLoadingBuild ? (
                <span className="text-slate-400 text-sm">Loading...</span>
              ) : build ? (
                <button
                  onClick={() => onNavigateToBuild?.(build)}
                  className="inline-flex items-center text-blue-600 hover:text-blue-800 hover:underline text-sm font-medium"
                  title={`View build: ${build.name}`}
                >
                  <svg className="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                  </svg>
                  {build.name}
                </button>
              ) : (
                <code className="text-xs bg-slate-100 px-2 py-1 rounded">{model.build_id}</code>
              )
            } />
            <InfoRow label="Datasource" value={
              isLoadingDatasource ? (
                <span className="text-slate-400 text-sm">Loading...</span>
              ) : datasource ? (
                <button
                  onClick={() => onNavigateToDatasource?.(datasource)}
                  className="inline-flex items-center text-blue-600 hover:text-blue-800 hover:underline text-sm font-medium"
                  title={`View datasource: ${datasource.name}`}
                >
                  <svg className="w-4 h-4 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
                  </svg>
                  {datasource.name}
                </button>
              ) : (
                <code className="text-xs bg-slate-100 px-2 py-1 rounded">{model.datasource_id}</code>
              )
            } />
            <InfoRow label="Created By" value={model.created_by} />
            <InfoRow label="Created At" value={new Date(model.created_at).toLocaleString()} />
          </dl>
        )}

        {/* Variables Tab */}
        {activeTab === 'variables' && (
          <div>
            {isLoading ? (
              <div className="flex items-center justify-center py-8">
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
              <div className="text-center py-8">
                <svg className="w-12 h-12 mx-auto text-slate-300 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
                </svg>
                <p className="text-sm text-slate-500">No variables available</p>
              </div>
            )}
          </div>
        )}

        {/* Files Tab */}
        {activeTab === 'files' && (
          <div>
            {(modelFile || trainingCodeFile) ? (
              <div className="space-y-3">
                {modelFile && (
                  <div className="flex items-center justify-between p-3 bg-slate-50 rounded-lg border border-slate-200">
                    <div className="flex items-center space-x-3">
                      <div className="p-2 bg-blue-100 rounded-lg">
                        <svg className="w-5 h-5 text-blue-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                        </svg>
                      </div>
                      <div>
                        <span className="text-sm font-medium text-slate-700">Model File</span>
                        <p className="text-xs text-slate-500 font-mono truncate max-w-xs" title={modelFile.file_name}>
                          {modelFile.file_name}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center space-x-2">
                      <span className="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-medium bg-blue-100 text-blue-700">
                        .pkl
                      </span>
                      <span className="text-xs text-slate-400">Binary</span>
                    </div>
                  </div>
                )}
                {trainingCodeFile && (
                  <div className="flex items-center justify-between p-3 bg-slate-50 rounded-lg border border-slate-200">
                    <div className="flex items-center space-x-3">
                      <div className="p-2 bg-yellow-100 rounded-lg">
                        <svg className="w-5 h-5 text-yellow-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
                        </svg>
                      </div>
                      <div>
                        <span className="text-sm font-medium text-slate-700">Training Code</span>
                        <p className="text-xs text-slate-500 font-mono truncate max-w-xs" title={trainingCodeFile.file_name}>
                          {trainingCodeFile.file_name}
                        </p>
                      </div>
                    </div>
                    <div className="flex items-center space-x-2">
                      <span className="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-medium bg-yellow-100 text-yellow-700">
                        .py
                      </span>
                      {isTextFile(trainingCodeFile) && (
                        <button
                          onClick={() => handleViewFile(trainingCodeFile)}
                          className="inline-flex items-center px-2.5 py-1 rounded-md text-xs font-medium bg-slate-100 text-slate-700 hover:bg-slate-200 transition-colors"
                        >
                          <svg className="w-3.5 h-3.5 mr-1" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                          </svg>
                          View
                        </button>
                      )}
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <div className="text-center py-8">
                <svg className="w-12 h-12 mx-auto text-slate-300 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                </svg>
                <p className="text-sm text-slate-500">No files available</p>
              </div>
            )}
          </div>
        )}

        {/* Versions Tab */}
        {activeTab === 'versions' && (
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-sm text-slate-600">Snapshot history (newest first). Create a version to save current state; restore to roll back.</span>
              <button
                type="button"
                onClick={async () => {
                  setVersionActionLoading('create');
                  try {
                    await versionApi.create(model.id);
                    const res = await versionApi.list(model.id, { page: 1, page_size: 10 });
                    setVersions(res.versions || []);
                    setVersionsTotal(res.total ?? 0);
                    setVersionsPage(1);
                    onRefreshModel?.();
                  } catch (err) {
                    console.error('Failed to create version:', err);
                    alert(err instanceof Error ? err.message : 'Failed to create version');
                  } finally {
                    setVersionActionLoading(null);
                  }
                }}
                disabled={!!versionActionLoading}
                className="px-3 py-1.5 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-md disabled:opacity-50 flex items-center space-x-1"
              >
                {versionActionLoading === 'create' ? (
                  <>
                    <svg className="animate-spin h-4 w-4" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                    </svg>
                    <span>Creating...</span>
                  </>
                ) : (
                  <>
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                    </svg>
                    <span>Create version</span>
                  </>
                )}
              </button>
            </div>
            {isLoadingVersions ? (
              <div className="flex items-center justify-center py-8">
                <svg className="animate-spin h-5 w-5 text-blue-600" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                </svg>
                <span className="ml-2 text-sm text-slate-500">Loading versions...</span>
              </div>
            ) : versions.length === 0 ? (
              <div className="text-center py-8 border border-dashed border-slate-200 rounded-lg">
                <svg className="w-12 h-12 mx-auto text-slate-300 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <p className="text-sm text-slate-500">No versions yet</p>
                <p className="text-xs text-slate-400 mt-1">Create a version to snapshot the current model state.</p>
              </div>
            ) : (
              <div className="border border-slate-200 rounded-lg overflow-hidden">
                <table className="min-w-full text-sm">
                  <thead className="bg-slate-50">
                    <tr>
                      <th className="text-left py-2 px-3 font-medium text-slate-600">#</th>
                      <th className="text-left py-2 px-3 font-medium text-slate-600">Name</th>
                      <th className="text-left py-2 px-3 font-medium text-slate-600">Created</th>
                      <th className="text-left py-2 px-3 font-medium text-slate-600">By</th>
                      <th className="text-right py-2 px-3 font-medium text-slate-600">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {versions.map((v) => (
                      <tr key={v.id} className="border-t border-slate-100 hover:bg-slate-50/50">
                        <td className="py-2 px-3 font-mono text-slate-600">{v.version_number}</td>
                        <td className="py-2 px-3">{v.name || '-'}</td>
                        <td className="py-2 px-3 text-slate-500">{new Date(v.created_at).toLocaleString()}</td>
                        <td className="py-2 px-3 text-slate-500">{v.created_by}</td>
                        <td className="py-2 px-3 text-right">
                          <button
                            type="button"
                            onClick={async () => {
                              if (!confirm(`Restore model to version ${v.version_number}? Current state will be replaced.`)) return;
                              setVersionActionLoading(v.id);
                              try {
                                await versionApi.restore(model.id, v.id);
                                onRefreshModel?.();
                                const res = await versionApi.list(model.id, { page: versionsPage, page_size: 10 });
                                setVersions(res.versions || []);
                                setVersionsTotal(res.total ?? 0);
                              } catch (err) {
                                console.error('Failed to restore:', err);
                                alert(err instanceof Error ? err.message : 'Failed to restore version');
                              } finally {
                                setVersionActionLoading(null);
                              }
                            }}
                            disabled={!!versionActionLoading}
                            className="px-2 py-1 text-xs font-medium text-amber-700 bg-amber-50 hover:bg-amber-100 rounded disabled:opacity-50"
                          >
                            {versionActionLoading === v.id ? 'Restoring...' : 'Restore'}
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {versionsTotal > 10 && (
                  <div className="px-3 py-2 bg-slate-50 border-t border-slate-200 flex items-center justify-between text-xs text-slate-500">
                    <span>Showing {versions.length} of {versionsTotal}</span>
                    <div className="space-x-2">
                      <button
                        type="button"
                        disabled={versionsPage <= 1}
                        onClick={() => setVersionsPage((p) => Math.max(1, p - 1))}
                        className="text-blue-600 hover:underline disabled:opacity-50"
                      >
                        Previous
                      </button>
                      <button
                        type="button"
                        disabled={versionsPage * 10 >= versionsTotal}
                        onClick={() => setVersionsPage((p) => p + 1)}
                        className="text-blue-600 hover:underline disabled:opacity-50"
                      >
                        Next
                      </button>
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        )}

        {/* Performance Tab */}
        {activeTab === 'performance' && (
          <PerformanceMonitorPanel 
            modelId={model.id} 
            modelType={model.model_type} 
            modelMetrics={model.metrics}
          />
        )}
      </div>

      {/* File Viewer Modal */}
      {viewingFile && (
        <div className="fixed inset-0 z-50 overflow-y-auto">
          <div className="flex min-h-screen items-center justify-center px-4 pt-4 pb-20 text-center sm:block sm:p-0">
            {/* Background overlay */}
            <div 
              className="fixed inset-0 bg-slate-900/60 transition-opacity"
              onClick={handleCloseViewer}
            />

            {/* Modal panel */}
            <div className="inline-block align-bottom bg-white rounded-xl text-left overflow-hidden shadow-2xl transform transition-all sm:my-8 sm:align-middle sm:max-w-4xl sm:w-full">
              {/* Header */}
              <div className="bg-slate-800 px-6 py-4 flex items-center justify-between">
                <div className="flex items-center space-x-3">
                  <div className="p-2 bg-slate-700 rounded-lg">
                    <svg className="w-5 h-5 text-yellow-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4" />
                    </svg>
                  </div>
                  <div>
                    <h3 className="text-lg font-semibold text-white">{viewingFile.file_name}</h3>
                    <p className="text-sm text-slate-400">
                      {viewingFile.file_type === 'training_code' ? 'Training Code' : viewingFile.file_type}
                      {fileContent && ` • ${(fileContent.size / 1024).toFixed(1)} KB`}
                    </p>
                  </div>
                </div>
                <button
                  onClick={handleCloseViewer}
                  className="p-2 text-slate-400 hover:text-white hover:bg-slate-700 rounded-lg transition-colors"
                >
                  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
                  </svg>
                </button>
              </div>

              {/* Content */}
              <div className="bg-slate-900 max-h-[70vh] overflow-auto">
                {isLoadingContent ? (
                  <div className="flex items-center justify-center py-16">
                    <svg className="animate-spin h-8 w-8 text-blue-500" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
                    </svg>
                    <span className="ml-3 text-slate-400">Loading file content...</span>
                  </div>
                ) : contentError ? (
                  <div className="flex flex-col items-center justify-center py-16">
                    <svg className="w-12 h-12 text-red-500 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z" />
                    </svg>
                    <p className="text-red-400 text-sm">{contentError}</p>
                  </div>
                ) : fileContent && !fileContent.is_text ? (
                  <div className="flex flex-col items-center justify-center py-16">
                    <svg className="w-12 h-12 text-slate-500 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
                    </svg>
                    <p className="text-slate-400 text-sm">This is a binary file and cannot be displayed as text.</p>
                  </div>
                ) : fileContent ? (
                  <div className="relative">
                    {/* Language badge */}
                    <div className="absolute top-3 right-3 px-2 py-1 bg-slate-700 rounded text-xs text-slate-300 font-mono">
                      {getLanguage(viewingFile.file_name)}
                    </div>
                    {/* Code content */}
                    <pre className="p-6 text-sm font-mono text-slate-100 overflow-x-auto leading-relaxed">
                      <code>{fileContent.content}</code>
                    </pre>
                  </div>
                ) : null}
              </div>

              {/* Footer */}
              <div className="bg-slate-50 px-6 py-3 flex justify-end">
                <button
                  onClick={handleCloseViewer}
                  className="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 transition-colors"
                >
                  Close
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
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

