import { useState, MouseEvent, useEffect, useCallback } from 'react';
import Layout from '../components/Layout';
import TreeView, { TreeNode } from '../components/TreeView';
import DetailPanel, { DataNode } from '../components/DetailPanel';
import DataPreviewPanel from '../components/DataPreviewPanel';
import ContextMenu, { MenuItem, MenuIcons } from '../components/ContextMenu';
import FolderDialog from '../components/FolderDialog';
import ProjectDialog from '../components/ProjectDialog';
import BuildModelDialog from '../components/BuildModelDialog';
import BuildEditDialog from '../components/BuildEditDialog';
import ModelEditDialog from '../components/ModelEditDialog';
import ScoreModelDialog from '../components/ScoreModelDialog';
import RetrainDialog from '../components/RetrainDialog';
import ConfirmDialog from '../components/ConfirmDialog';
import DataSourcePanel from '../components/DataSourcePanel';
import SearchPanel from '../components/SearchPanel';
import { folderApi, projectApi, buildApi, modelApi, datasourceApi, collectionApi, Folder, Project, ModelBuild, Model, Collection, Datasource, FolderContentsCount, SearchResultItem } from '../lib/api';

type RightPanelTab = 'details' | 'data';

type SidebarTab = 'explorer' | 'datasource';

interface ContextMenuState {
  x: number;
  y: number;
  node: TreeNode | null;
}

export default function MainPage() {
  const [activeTab, setActiveTab] = useState<SidebarTab>('explorer');
  const [selectedNode, setSelectedNode] = useState<TreeNode | null>(null);
  const [selectedDataNode, setSelectedDataNode] = useState<DataNode | null>(null);
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null);
  const [refreshTrigger, setRefreshTrigger] = useState(0);
  const [dataRefreshTrigger, setDataRefreshTrigger] = useState(0);
  // For targeted node refresh (folder-specific refresh)
  const [refreshNode, setRefreshNode] = useState<{ id: string; type: 'folder' | 'project' } | null>(null);
  // Explorer filters
  const [hideBuilds, setHideBuilds] = useState(false);
  const [hideModels, setHideModels] = useState(false);

  // Dialog states
  const [folderDialog, setFolderDialog] = useState<{
    isOpen: boolean;
    parentId?: string;
    folder?: Folder;
  }>({ isOpen: false });

  const [projectDialog, setProjectDialog] = useState<{
    isOpen: boolean;
    folderId?: string;
    folderName?: string;
    project?: Project;
  }>({ isOpen: false });

  const [deleteDialog, setDeleteDialog] = useState<{
    isOpen: boolean;
    node: TreeNode | null;
    isLoading: boolean;
    folderContents?: FolderContentsCount;
    isLoadingContents?: boolean;
  }>({ isOpen: false, node: null, isLoading: false });

  const [buildDialog, setBuildDialog] = useState<{
    isOpen: boolean;
    projectId?: string;
    projectName?: string;
    folderId?: string;
    folderName?: string;
  }>({ isOpen: false });

  const [buildEditDialog, setBuildEditDialog] = useState<{
    isOpen: boolean;
    build?: ModelBuild;
  }>({ isOpen: false });

  const [modelEditDialog, setModelEditDialog] = useState<{
    isOpen: boolean;
    model?: Model;
  }>({ isOpen: false });

  const [scoreDialog, setScoreDialog] = useState<{
    isOpen: boolean;
    model?: Model;
  }>({ isOpen: false });

  const [retrainDialog, setRetrainDialog] = useState<{
    isOpen: boolean;
    model: Model | null;
  }>({ isOpen: false, model: null });

  // Right panel tab state (for datasource data preview)
  const [rightPanelTab, setRightPanelTab] = useState<RightPanelTab>('details');

  // Search panel state
  const [searchOpen, setSearchOpen] = useState(false);

  // Reset right panel tab when data node selection changes (for data preview tab)
  useEffect(() => {
    setRightPanelTab('details');
  }, [selectedDataNode?.id]);

  // ⌘K / Ctrl+K opens search
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault();
        setSearchOpen((prev) => !prev);
      }
    };
    window.addEventListener('keydown', handler);
    return () => window.removeEventListener('keydown', handler);
  }, []);

  // Refresh tree
  const refresh = () => setRefreshTrigger((prev) => prev + 1);

  // Handle node selection from Explorer tree
  const handleSelect = (node: TreeNode) => {
    setSelectedNode(node);
    // Don't clear data selection - preserve state when switching tabs
  };

  // Navigate to a search result by loading its full data and selecting it
  const handleSearchResult = useCallback(async (item: SearchResultItem) => {
    try {
      if (item.type === 'build') {
        const build = await buildApi.get(item.id);
        setActiveTab('explorer');
        setSelectedNode({ id: item.id, name: item.name, type: 'build', data: build });
      } else if (item.type === 'model') {
        const model = await modelApi.get(item.id);
        setActiveTab('explorer');
        setSelectedNode({ id: item.id, name: item.name, type: 'model', data: model });
      } else if (item.type === 'project') {
        const project = await projectApi.getProject(item.id);
        setActiveTab('explorer');
        setSelectedNode({ id: item.id, name: item.name, type: 'project', data: project });
      } else if (item.type === 'folder') {
        const folder = await folderApi.getFolder(item.id);
        setActiveTab('explorer');
        setSelectedNode({ id: item.id, name: item.name, type: 'folder', data: folder });
      }
    } catch {
      // Fallback: select with partial data
      setActiveTab('explorer');
      setSelectedNode({ id: item.id, name: item.name, type: item.type as TreeNode['type'], data: {} as Model });
    }
  }, []);

  // Handle data item selection from Data tab
  const handleDataSelect = (item: { type: 'collection' | 'datasource'; data: Collection | Datasource }) => {
    setSelectedDataNode({
      id: item.data.id,
      name: item.data.name,
      type: item.type,
      data: item.data,
    });
    // Don't clear explorer selection - preserve state when switching tabs
  };

  // Handle delete data node (datasource or collection)
  const handleDeleteDataNode = async () => {
    if (!selectedDataNode) return;

    let confirmMessage: string;
    let hasDatasources = false;
    
    if (selectedDataNode.type === 'collection') {
      const collection = selectedDataNode.data as Collection;
      hasDatasources = collection.datasource_count > 0;
      confirmMessage = hasDatasources
        ? `Warning: This collection contains ${collection.datasource_count} data source${collection.datasource_count > 1 ? 's' : ''}. Deleting this collection will remove all data sources in it. Are you sure you want to continue?`
        : `Delete collection "${selectedDataNode.name}"? This action cannot be undone.`;
    } else {
      confirmMessage = `Delete data source "${selectedDataNode.name}"? This will delete the data from storage.`;
    }

    if (!confirm(confirmMessage)) return;

    try {
      if (selectedDataNode.type === 'datasource') {
        await datasourceApi.delete(selectedDataNode.id);
      } else {
        await collectionApi.delete(selectedDataNode.id, hasDatasources);
      }
      setSelectedDataNode(null);
      // Trigger refresh of the DataSourcePanel
      setDataRefreshTrigger((prev) => prev + 1);
    } catch (error) {
      console.error('Failed to delete:', error);
      alert(error instanceof Error ? error.message : 'Failed to delete');
    }
  };

  // Handle context menu
  const handleContextMenu = (e: MouseEvent, node: TreeNode | null) => {
    e.preventDefault();
    setContextMenu({
      x: e.clientX,
      y: e.clientY,
      node,
    });
  };

  // Handle sidebar right-click (for creating root items)
  const handleSidebarContextMenu = (e: MouseEvent<HTMLDivElement>) => {
    // Only handle if clicking on empty space, not on a tree node
    if ((e.target as HTMLElement).closest('[data-tree-node]')) {
      return;
    }
    e.preventDefault();
    setContextMenu({
      x: e.clientX,
      y: e.clientY,
      node: null,
    });
  };

  // Close context menu
  const closeContextMenu = () => setContextMenu(null);

  // Get context menu items based on node type
  const getContextMenuItems = (): MenuItem[] => {
    const node = contextMenu?.node;

    if (!node) {
      // Root level - can create folders and projects
      return [
        {
          label: 'New Folder',
          icon: MenuIcons.folder,
          onClick: () => setFolderDialog({ isOpen: true }),
        },
        {
          label: 'New Project',
          icon: MenuIcons.project,
          onClick: () => setProjectDialog({ isOpen: true }),
        },
        { label: '', divider: true, onClick: () => {} },
        {
          label: 'Refresh',
          icon: MenuIcons.refresh,
          onClick: refresh,
        },
      ];
    }

    if (node.type === 'folder') {
      return [
        {
          label: 'Build Model',
          icon: MenuIcons.project,
          onClick: () => setBuildDialog({ isOpen: true, folderId: node.id, folderName: node.name }),
        },
        { label: '', divider: true, onClick: () => {} },
        {
          label: 'New Subfolder',
          icon: MenuIcons.folder,
          onClick: () => setFolderDialog({ isOpen: true, parentId: node.id }),
        },
        {
          label: 'New Project',
          icon: MenuIcons.project,
          onClick: () => setProjectDialog({ isOpen: true, folderId: node.id, folderName: node.name }),
        },
        { label: '', divider: true, onClick: () => {} },
        {
          label: 'Refresh',
          icon: MenuIcons.refresh,
          onClick: () => setRefreshNode({ id: node.id, type: 'folder' }),
        },
        {
          label: 'Edit Folder',
          icon: MenuIcons.edit,
          onClick: () => setFolderDialog({ isOpen: true, folder: node.data as Folder }),
        },
        {
          label: 'Delete Folder',
          icon: MenuIcons.delete,
          danger: true,
          onClick: async () => {
            setDeleteDialog({ isOpen: true, node, isLoading: false, isLoadingContents: true });
            try {
              const contents = await folderApi.getContentsCount(node.id);
              setDeleteDialog(prev => ({ ...prev, folderContents: contents, isLoadingContents: false }));
            } catch (error) {
              console.error('Failed to get folder contents:', error);
              setDeleteDialog(prev => ({ ...prev, isLoadingContents: false }));
            }
          },
        },
      ];
    }

    if (node.type === 'project') {
      return [
        {
          label: 'Build Model',
          icon: MenuIcons.project,
          onClick: () => setBuildDialog({ isOpen: true, projectId: node.id, projectName: node.name }),
        },
        { label: '', divider: true, onClick: () => {} },
        {
          label: 'Refresh',
          icon: MenuIcons.refresh,
          onClick: () => setRefreshNode({ id: node.id, type: 'project' }),
        },
        {
          label: 'Edit Project',
          icon: MenuIcons.edit,
          onClick: () => setProjectDialog({ isOpen: true, project: node.data as Project }),
        },
        {
          label: 'Delete Project',
          icon: MenuIcons.delete,
          danger: true,
          onClick: () => setDeleteDialog({ isOpen: true, node, isLoading: false }),
        },
      ];
    }

    if (node.type === 'build') {
      const build = node.data as ModelBuild;
      const items: MenuItem[] = [];
      
      if (build.status === 'pending') {
        items.push({
          label: 'Start Build',
          icon: MenuIcons.project,
          onClick: async () => {
            try {
              await buildApi.start(node.id);
              // Targeted refresh of parent
              if (build.folder_id) {
                setRefreshNode({ id: build.folder_id, type: 'folder' });
              } else if (build.project_id) {
                setRefreshNode({ id: build.project_id, type: 'project' });
              } else {
                refresh();
              }
            } catch (error) {
              console.error('Failed to start build:', error);
            }
          },
        });
      }
      
      if (build.status === 'running') {
        items.push({
          label: 'Cancel Build',
          icon: MenuIcons.delete,
          onClick: async () => {
            if (!confirm(`Cancel build "${build.name}"?`)) return;
            try {
              await buildApi.cancel(node.id);
              // Targeted refresh of parent
              if (build.folder_id) {
                setRefreshNode({ id: build.folder_id, type: 'folder' });
              } else if (build.project_id) {
                setRefreshNode({ id: build.project_id, type: 'project' });
              } else {
                refresh();
              }
            } catch (error) {
              console.error('Failed to cancel build:', error);
              alert(error instanceof Error ? error.message : 'Failed to cancel build');
            }
          },
        });
      }
      
      items.push(
        {
          label: 'Edit Build',
          icon: MenuIcons.edit,
          onClick: () => setBuildEditDialog({ isOpen: true, build }),
        },
        {
          label: 'Delete Build',
          icon: MenuIcons.delete,
          danger: true,
          onClick: () => setDeleteDialog({ isOpen: true, node, isLoading: false }),
        },
      );
      
      return items;
    }

    if (node.type === 'model') {
      return [
        {
          label: 'View Details',
          icon: MenuIcons.edit,
          onClick: () => setSelectedNode(node),
        },
        {
          label: 'Delete Model',
          icon: MenuIcons.delete,
          danger: true,
          onClick: () => setDeleteDialog({ isOpen: true, node, isLoading: false }),
        },
      ];
    }

    return [];
  };

  // Handle delete
  const handleDelete = async () => {
    if (!deleteDialog.node) return;

    setDeleteDialog((prev) => ({ ...prev, isLoading: true }));
    const nodeToDelete = deleteDialog.node;

    try {
      if (nodeToDelete.type === 'folder') {
        // Use force delete if folder has contents
        const hasContents = deleteDialog.folderContents && (
          deleteDialog.folderContents.subfolder_count > 0 ||
          deleteDialog.folderContents.project_count > 0 ||
          deleteDialog.folderContents.model_count > 0 ||
          deleteDialog.folderContents.build_count > 0
        );
        await folderApi.delete(nodeToDelete.id, hasContents);
      } else if (nodeToDelete.type === 'project') {
        await projectApi.delete(nodeToDelete.id);
      } else if (nodeToDelete.type === 'build') {
        await buildApi.delete(nodeToDelete.id);
      } else if (nodeToDelete.type === 'model') {
        await modelApi.delete(nodeToDelete.id);
      }

      // Clear selection if deleted node was selected
      if (selectedNode?.id === nodeToDelete.id) {
        setSelectedNode(null);
      }

      // Targeted refresh based on node type and parent
      if (nodeToDelete.type === 'folder') {
        const folder = nodeToDelete.data as Folder;
        if (folder.parent_id) {
          setRefreshNode({ id: folder.parent_id, type: 'folder' });
        } else {
          refresh(); // Root level folder, need full refresh
        }
      } else if (nodeToDelete.type === 'project') {
        const project = nodeToDelete.data as Project;
        if (project.folder_id) {
          setRefreshNode({ id: project.folder_id, type: 'folder' });
        } else {
          refresh(); // Root level project (shouldn't happen normally)
        }
      } else if (nodeToDelete.type === 'build') {
        const build = nodeToDelete.data as ModelBuild;
        if (build.folder_id) {
          setRefreshNode({ id: build.folder_id, type: 'folder' });
        } else if (build.project_id) {
          setRefreshNode({ id: build.project_id, type: 'project' });
        } else {
          refresh();
        }
      } else if (nodeToDelete.type === 'model') {
        const model = nodeToDelete.data as Model;
        if (model.folder_id) {
          setRefreshNode({ id: model.folder_id, type: 'folder' });
        } else if (model.project_id) {
          setRefreshNode({ id: model.project_id, type: 'project' });
        } else {
          refresh();
        }
      } else {
        refresh();
      }
      
      setDeleteDialog({ isOpen: false, node: null, isLoading: false });
    } catch (error) {
      console.error('Delete failed:', error);
      setDeleteDialog((prev) => ({ ...prev, isLoading: false }));
      
      // Show helpful error message
      const errorMsg = error instanceof Error ? error.message : 'Delete failed';
      if (errorMsg.includes('already running') || errorMsg.includes('running')) {
        alert('Cannot delete a running build. Please cancel the build first, then try deleting again.');
      } else {
        alert(errorMsg);
      }
    }
  };

  // Handle edit from detail panel
  const handleEdit = () => {
    if (!selectedNode) return;

    if (selectedNode.type === 'folder') {
      setFolderDialog({ isOpen: true, folder: selectedNode.data as Folder });
    } else if (selectedNode.type === 'project') {
      setProjectDialog({ isOpen: true, project: selectedNode.data as Project });
    } else if (selectedNode.type === 'build') {
      setBuildEditDialog({ isOpen: true, build: selectedNode.data as ModelBuild });
    } else if (selectedNode.type === 'model') {
      setModelEditDialog({ isOpen: true, model: selectedNode.data as Model });
    }
  };

  // Handle delete from detail panel
  const handleDeleteFromPanel = () => {
    if (!selectedNode) return;
    setDeleteDialog({ isOpen: true, node: selectedNode, isLoading: false });
  };

  // Handle start build
  const handleStartBuild = async () => {
    if (!selectedNode || selectedNode.type !== 'build') return;
    
    try {
      await buildApi.start(selectedNode.id);
      
      // Refresh the selected node data
      const updatedBuild = await buildApi.get(selectedNode.id);
      setSelectedNode({
        ...selectedNode,
        data: updatedBuild,
      });
      
      // Targeted refresh of parent folder or project
      if (updatedBuild.folder_id) {
        setRefreshNode({ id: updatedBuild.folder_id, type: 'folder' });
      } else if (updatedBuild.project_id) {
        setRefreshNode({ id: updatedBuild.project_id, type: 'project' });
      } else {
        refresh();
      }
    } catch (error) {
      console.error('Failed to start build:', error);
      alert('Failed to start build. Please try again.');
    }
  };

  // Handle cancel build
  const handleCancelBuild = async () => {
    if (!selectedNode || selectedNode.type !== 'build') return;
    
    const build = selectedNode.data as ModelBuild;
    if (!confirm(`Cancel build "${build.name}"? This will stop the build process.`)) return;
    
    try {
      await buildApi.cancel(selectedNode.id);
      
      // Refresh the selected node data
      const updatedBuild = await buildApi.get(selectedNode.id);
      setSelectedNode({
        ...selectedNode,
        data: updatedBuild,
      });
      
      // Targeted refresh of parent folder or project
      if (updatedBuild.folder_id) {
        setRefreshNode({ id: updatedBuild.folder_id, type: 'folder' });
      } else if (updatedBuild.project_id) {
        setRefreshNode({ id: updatedBuild.project_id, type: 'project' });
      } else {
        refresh();
      }
    } catch (error) {
      console.error('Failed to cancel build:', error);
      alert(error instanceof Error ? error.message : 'Failed to cancel build. Please try again.');
    }
  };

  // Handle navigation to a datasource (from model details)
  const handleNavigateToDatasource = (datasource: Datasource) => {
    // Switch to data tab and select the datasource
    setActiveTab('datasource');
    setSelectedDataNode({
      id: datasource.id,
      name: datasource.name,
      type: 'datasource',
      data: datasource,
    });
  };

  // Handle navigation to a build (from model details)
  const handleNavigateToBuild = (build: ModelBuild) => {
    // Switch to explorer tab and select the build
    setActiveTab('explorer');
    setSelectedNode({
      id: build.id,
      name: build.name,
      type: 'build',
      data: build,
    });
  };

  // Explorer tab content
  const explorerContent = (
    <div
      className="flex flex-col h-full"
      onContextMenu={handleSidebarContextMenu}
    >
      {/* Explorer header */}
      <div className="px-4 py-3 border-b border-slate-200 flex items-center justify-between">
        <h2 className="text-sm font-semibold text-slate-700">Explorer</h2>
        <div className="flex items-center space-x-1">
          {/* Search button */}
          <button
            onClick={() => setSearchOpen(true)}
            className="p-1.5 rounded text-slate-500 hover:text-slate-700 hover:bg-slate-100 transition-colors"
            title="Search (⌘K)"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-4.35-4.35M17 11A6 6 0 115 11a6 6 0 0112 0z" />
            </svg>
          </button>
          <button
            onClick={() => setFolderDialog({ 
              isOpen: true, 
              parentId: selectedNode?.type === 'folder' ? selectedNode.id : undefined 
            })}
            disabled={selectedNode !== null && selectedNode.type !== 'folder'}
            className={`p-1.5 rounded transition-colors ${
              selectedNode !== null && selectedNode.type !== 'folder'
                ? 'text-slate-300 cursor-not-allowed'
                : 'text-slate-500 hover:text-slate-700 hover:bg-slate-100'
            }`}
            title={
              selectedNode !== null && selectedNode.type !== 'folder'
                ? 'Cannot create folder under this item'
                : selectedNode?.type === 'folder'
                ? 'New Subfolder'
                : 'New Folder'
            }
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 13h6m-3-3v6m-9 1V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2z" />
            </svg>
          </button>
          <button
            onClick={() => {
              const folderId = selectedNode?.type === 'folder' ? selectedNode.id : undefined;
              const folderName = selectedNode?.type === 'folder' ? selectedNode.name : undefined;
              setProjectDialog({ isOpen: true, folderId, folderName });
            }}
            disabled={selectedNode !== null && selectedNode.type !== 'folder'}
            className={`p-1.5 rounded transition-colors ${
              selectedNode !== null && selectedNode.type !== 'folder'
                ? 'text-slate-300 cursor-not-allowed'
                : 'text-slate-500 hover:text-slate-700 hover:bg-slate-100'
            }`}
            title={
              selectedNode !== null && selectedNode.type !== 'folder'
                ? 'Cannot create project under this item'
                : selectedNode?.type === 'folder'
                ? `New Project in "${selectedNode.name}"`
                : 'New Project'
            }
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 13h6m-3-3v6m5 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
          </button>
          {/* Separator */}
          <div className="w-px h-4 bg-slate-200 mx-1" />
          
          {/* Filter toggles */}
          <button
            onClick={() => setHideBuilds(!hideBuilds)}
            className={`p-1.5 rounded transition-colors ${
              hideBuilds
                ? 'text-slate-300 bg-slate-100'
                : 'text-purple-500 hover:text-purple-600 hover:bg-purple-50'
            }`}
            title={hideBuilds ? 'Show Builds' : 'Hide Builds'}
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
            </svg>
          </button>
          <button
            onClick={() => setHideModels(!hideModels)}
            className={`p-1.5 rounded transition-colors ${
              hideModels
                ? 'text-slate-300 bg-slate-100'
                : 'text-emerald-500 hover:text-emerald-600 hover:bg-emerald-50'
            }`}
            title={hideModels ? 'Show Models' : 'Hide Models'}
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
            </svg>
          </button>
          
          {/* Separator */}
          <div className="w-px h-4 bg-slate-200 mx-1" />
          
          <button
            onClick={refresh}
            className="p-1.5 text-slate-500 hover:text-slate-700 hover:bg-slate-100 rounded transition-colors"
            title="Refresh"
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
            </svg>
          </button>
        </div>
      </div>

      {/* Tree view */}
      <div className="flex-1 overflow-auto">
        <TreeView
          onSelect={handleSelect}
          selectedId={selectedNode?.id}
          onContextMenu={handleContextMenu}
          refreshTrigger={refreshTrigger}
          refreshNodeId={refreshNode?.id}
          refreshNodeType={refreshNode?.type}
          onNodeRefreshed={() => setRefreshNode(null)}
          hideBuilds={hideBuilds}
          hideModels={hideModels}
        />
      </div>
    </div>
  );

  // Sidebar content with vertical tabs
  const sidebar = (
    <div className="flex h-full">
      {/* Vertical tab bar */}
      <div className="flex flex-col w-12 bg-slate-100 border-r border-slate-200 shrink-0">
        <button
          onClick={() => setActiveTab('explorer')}
          className={`p-3 transition-colors relative group ${
            activeTab === 'explorer'
              ? 'text-blue-600 bg-white'
              : 'text-slate-500 hover:text-slate-700 hover:bg-slate-50'
          }`}
          title="Explorer"
        >
          <svg className="w-5 h-5 mx-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
          </svg>
          {activeTab === 'explorer' && (
            <div className="absolute left-0 top-0 bottom-0 w-0.5 bg-blue-600" />
          )}
        </button>
        <button
          onClick={() => setActiveTab('datasource')}
          className={`p-3 transition-colors relative group ${
            activeTab === 'datasource'
              ? 'text-blue-600 bg-white'
              : 'text-slate-500 hover:text-slate-700 hover:bg-slate-50'
          }`}
          title="Data"
        >
          <svg className="w-5 h-5 mx-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
          </svg>
          {activeTab === 'datasource' && (
            <div className="absolute left-0 top-0 bottom-0 w-0.5 bg-blue-600" />
          )}
        </button>
      </div>

      {/* Tab content - render both but hide inactive to preserve state */}
      <div className="flex-1 flex flex-col min-w-0">
        <div className={`flex-1 overflow-hidden ${activeTab === 'explorer' ? '' : 'hidden'}`}>
          {explorerContent}
        </div>
        <div className={`flex-1 overflow-hidden ${activeTab === 'datasource' ? '' : 'hidden'}`}>
          <DataSourcePanel 
            onSelect={handleDataSelect} 
            refreshTrigger={dataRefreshTrigger}
            externalSelectedDatasource={
              selectedDataNode?.type === 'datasource' 
                ? { id: selectedDataNode.id, collection_id: (selectedDataNode.data as Datasource).collection_id }
                : null
            }
          />
        </div>
      </div>
    </div>
  );

  // Check if we should show data preview tab (only for datasources in data tab)
  const showDataPreviewTab = activeTab === 'datasource' && selectedDataNode?.type === 'datasource';

  // Determine which node to show based on active sidebar tab
  const currentNode = activeTab === 'explorer' ? selectedNode : null;
  const currentDataNode = activeTab === 'datasource' ? selectedDataNode : null;

  // Right panel content with optional tabs
  const rightPanelContent = (
    <div className="h-full flex flex-col">
      {/* Tab bar - only show for datasources in data tab */}
      {showDataPreviewTab && (
        <div className="flex border-b border-slate-200 bg-white shrink-0">
          <button
            onClick={() => setRightPanelTab('details')}
            className={`px-6 py-2.5 text-sm font-medium transition-colors relative ${
              rightPanelTab === 'details'
                ? 'text-blue-600'
                : 'text-slate-500 hover:text-slate-700 hover:bg-slate-50'
            }`}
          >
            <div className="flex items-center space-x-2">
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
              </svg>
              <span>Details</span>
            </div>
            {rightPanelTab === 'details' && (
              <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600" />
            )}
          </button>
          <button
            onClick={() => setRightPanelTab('data')}
            className={`px-6 py-2.5 text-sm font-medium transition-colors relative ${
              rightPanelTab === 'data'
                ? 'text-blue-600'
                : 'text-slate-500 hover:text-slate-700 hover:bg-slate-50'
            }`}
          >
            <div className="flex items-center space-x-2">
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 10h18M3 14h18m-9-4v8m-7 0h14a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z" />
              </svg>
              <span>Data</span>
            </div>
            {rightPanelTab === 'data' && (
              <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600" />
            )}
          </button>
        </div>
      )}

      {/* Tab content */}
      <div className="flex-1 overflow-auto p-6">
        {rightPanelTab === 'details' || !showDataPreviewTab ? (
          <DetailPanel
            node={currentNode}
            dataNode={currentDataNode}
            onEdit={handleEdit}
            onDelete={handleDeleteFromPanel}
            onBuildModel={
              currentNode?.type === 'project'
                ? () => setBuildDialog({ isOpen: true, projectId: currentNode.id, projectName: currentNode.name })
                : currentNode?.type === 'folder'
                ? () => setBuildDialog({ isOpen: true, folderId: currentNode.id, folderName: currentNode.name })
                : undefined
            }
            onStartBuild={currentNode?.type === 'build' ? handleStartBuild : undefined}
            onCancelBuild={currentNode?.type === 'build' ? handleCancelBuild : undefined}
            onDeleteDataNode={currentDataNode ? handleDeleteDataNode : undefined}
            onScoreModel={currentNode?.type === 'model' ? () => setScoreDialog({ isOpen: true, model: currentNode.data as Model }) : undefined}
            onRetrain={currentNode?.type === 'model' ? () => setRetrainDialog({ isOpen: true, model: currentNode.data as Model }) : undefined}
            onRefreshModel={
              currentNode?.type === 'model'
                ? () => {
                    refresh();
                    modelApi.get(currentNode.id).then((m) => setSelectedNode((prev) => (prev?.id === currentNode.id ? { ...prev, data: m } : prev)));
                  }
                : undefined
            }
            onNavigateToDatasource={handleNavigateToDatasource}
            onNavigateToBuild={handleNavigateToBuild}
          />
        ) : (
          <DataPreviewPanel datasource={currentDataNode?.data as Datasource} />
        )}
      </div>
    </div>
  );

  return (
    <Layout sidebar={sidebar}>
      {rightPanelContent}

      {/* Search Panel */}
      <SearchPanel
        isOpen={searchOpen}
        onClose={() => setSearchOpen(false)}
        onSelectResult={handleSearchResult}
        scopeFolderId={selectedNode?.type === 'folder' ? selectedNode.id : undefined}
        scopeFolderName={selectedNode?.type === 'folder' ? selectedNode.name : undefined}
      />

      {/* Context Menu */}
      {contextMenu && (
        <ContextMenu
          x={contextMenu.x}
          y={contextMenu.y}
          items={getContextMenuItems()}
          onClose={closeContextMenu}
        />
      )}

      {/* Folder Dialog */}
      <FolderDialog
        isOpen={folderDialog.isOpen}
        onClose={() => setFolderDialog({ isOpen: false })}
        onSuccess={() => {
          // Targeted refresh: only refresh the parent folder if creating subfolder
          if (folderDialog.parentId) {
            setRefreshNode({ id: folderDialog.parentId, type: 'folder' });
          } else {
            refresh(); // Full refresh for root-level folders
          }
        }}
        parentId={folderDialog.parentId}
        folder={folderDialog.folder}
      />

      {/* Project Dialog */}
      <ProjectDialog
        isOpen={projectDialog.isOpen}
        onClose={() => setProjectDialog({ isOpen: false })}
        onSuccess={() => {
          // Targeted refresh: only refresh the parent folder
          if (projectDialog.folderId) {
            setRefreshNode({ id: projectDialog.folderId, type: 'folder' });
          } else {
            refresh(); // Fallback to full refresh
          }
        }}
        folderId={projectDialog.folderId}
        folderName={projectDialog.folderName}
        project={projectDialog.project}
      />

      {/* Delete Confirmation Dialog */}
      <ConfirmDialog
        isOpen={deleteDialog.isOpen}
        onClose={() => setDeleteDialog({ isOpen: false, node: null, isLoading: false, folderContents: undefined, isLoadingContents: false })}
        onConfirm={handleDelete}
        title={`Delete ${deleteDialog.node?.type || 'item'}?`}
        message={(() => {
          const baseMessage = `Are you sure you want to delete "${deleteDialog.node?.name}"? This action cannot be undone.`;
          
          if (deleteDialog.node?.type !== 'folder') {
            return baseMessage;
          }
          
          if (deleteDialog.isLoadingContents) {
            return 'Checking folder contents...';
          }
          
          const contents = deleteDialog.folderContents;
          if (!contents) {
            return baseMessage;
          }
          
          const hasContents = contents.subfolder_count > 0 || contents.project_count > 0 || 
                             contents.model_count > 0 || contents.build_count > 0;
          
          if (!hasContents) {
            return baseMessage;
          }
          
          const parts: string[] = [];
          if (contents.subfolder_count > 0) {
            parts.push(`${contents.subfolder_count} subfolder${contents.subfolder_count > 1 ? 's' : ''}`);
          }
          if (contents.project_count > 0) {
            parts.push(`${contents.project_count} project${contents.project_count > 1 ? 's' : ''}`);
          }
          if (contents.model_count > 0) {
            parts.push(`${contents.model_count} model${contents.model_count > 1 ? 's' : ''}`);
          }
          if (contents.build_count > 0) {
            parts.push(`${contents.build_count} build${contents.build_count > 1 ? 's' : ''}`);
          }
          
          return `Warning: This folder contains ${parts.join(', ')}. Deleting this folder will remove all contents. Are you sure you want to continue?`;
        })()}
        confirmText="Delete"
        isLoading={deleteDialog.isLoading || deleteDialog.isLoadingContents}
      />

      {/* Build Model Dialog */}
      <BuildModelDialog
        isOpen={buildDialog.isOpen}
        onClose={() => setBuildDialog({ isOpen: false })}
        onSuccess={() => {
          // Targeted refresh: only refresh the parent folder or project
          if (buildDialog.folderId) {
            setRefreshNode({ id: buildDialog.folderId, type: 'folder' });
          } else if (buildDialog.projectId) {
            setRefreshNode({ id: buildDialog.projectId, type: 'project' });
          } else {
            refresh(); // Fallback to full refresh
          }
        }}
        projectId={buildDialog.projectId}
        projectName={buildDialog.projectName}
        folderId={buildDialog.folderId}
        folderName={buildDialog.folderName}
      />

      {/* Build Edit Dialog */}
      <BuildEditDialog
        isOpen={buildEditDialog.isOpen}
        onClose={() => setBuildEditDialog({ isOpen: false })}
        onSuccess={() => {
          // Targeted refresh based on parent
          const build = buildEditDialog.build;
          if (build?.folder_id) {
            setRefreshNode({ id: build.folder_id, type: 'folder' });
          } else if (build?.project_id) {
            setRefreshNode({ id: build.project_id, type: 'project' });
          } else {
            refresh();
          }
          // Refresh selected node if it's the edited build
          if (selectedNode?.type === 'build' && build?.id === selectedNode.id) {
            buildApi.get(selectedNode.id).then((updatedBuild) => {
              setSelectedNode({ ...selectedNode, name: updatedBuild.name, data: updatedBuild });
            });
          }
        }}
        build={buildEditDialog.build}
      />

      {/* Model Edit Dialog */}
      <ModelEditDialog
        isOpen={modelEditDialog.isOpen}
        onClose={() => setModelEditDialog({ isOpen: false })}
        onSuccess={() => {
          // Targeted refresh based on parent
          const model = modelEditDialog.model;
          if (model?.folder_id) {
            setRefreshNode({ id: model.folder_id, type: 'folder' });
          } else if (model?.project_id) {
            setRefreshNode({ id: model.project_id, type: 'project' });
          } else {
            refresh();
          }
          // Refresh selected node if it's the edited model
          if (selectedNode?.type === 'model' && model?.id === selectedNode.id) {
            modelApi.get(selectedNode.id).then((updatedModel) => {
              setSelectedNode({ ...selectedNode, name: updatedModel.name, data: updatedModel });
            });
          }
        }}
        model={modelEditDialog.model}
      />

      {/* Score Model Dialog */}
      <ScoreModelDialog
        isOpen={scoreDialog.isOpen}
        onClose={() => setScoreDialog({ isOpen: false })}
        onSuccess={() => {
          setDataRefreshTrigger((prev) => prev + 1);
        }}
        model={scoreDialog.model}
      />

      <RetrainDialog
        isOpen={retrainDialog.isOpen}
        onClose={() => setRetrainDialog({ isOpen: false, model: null })}
        onSuccess={(buildId) => {
          refresh();
          if (buildId) {
            setActiveTab('explorer');
            buildApi.get(buildId).then((build) => {
              setSelectedNode({ id: build.id, name: build.name, type: 'build', data: build });
            }).catch(() => {});
          }
        }}
        model={retrainDialog.model}
      />
    </Layout>
  );
}

