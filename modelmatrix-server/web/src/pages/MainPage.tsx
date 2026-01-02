import { useState, MouseEvent } from 'react';
import Layout from '../components/Layout';
import TreeView, { TreeNode } from '../components/TreeView';
import DetailPanel from '../components/DetailPanel';
import ContextMenu, { MenuItem, MenuIcons } from '../components/ContextMenu';
import FolderDialog from '../components/FolderDialog';
import ProjectDialog from '../components/ProjectDialog';
import BuildModelDialog from '../components/BuildModelDialog';
import ConfirmDialog from '../components/ConfirmDialog';
import DataSourcePanel from '../components/DataSourcePanel';
import { folderApi, projectApi, Folder, Project } from '../lib/api';

type SidebarTab = 'explorer' | 'datasource';

interface ContextMenuState {
  x: number;
  y: number;
  node: TreeNode | null;
}

export default function MainPage() {
  const [activeTab, setActiveTab] = useState<SidebarTab>('explorer');
  const [selectedNode, setSelectedNode] = useState<TreeNode | null>(null);
  const [contextMenu, setContextMenu] = useState<ContextMenuState | null>(null);
  const [refreshTrigger, setRefreshTrigger] = useState(0);

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
  }>({ isOpen: false, node: null, isLoading: false });

  const [buildDialog, setBuildDialog] = useState<{
    isOpen: boolean;
    projectId?: string;
    projectName?: string;
    folderId?: string;
    folderName?: string;
  }>({ isOpen: false });

  // Refresh tree
  const refresh = () => setRefreshTrigger((prev) => prev + 1);

  // Handle node selection
  const handleSelect = (node: TreeNode) => {
    setSelectedNode(node);
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
          label: 'Edit Folder',
          icon: MenuIcons.edit,
          onClick: () => setFolderDialog({ isOpen: true, folder: node.data as Folder }),
        },
        {
          label: 'Delete Folder',
          icon: MenuIcons.delete,
          danger: true,
          onClick: () => setDeleteDialog({ isOpen: true, node, isLoading: false }),
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
      return [
        {
          label: 'View Details',
          icon: MenuIcons.edit,
          onClick: () => setSelectedNode(node),
        },
      ];
    }

    if (node.type === 'model') {
      return [
        {
          label: 'View Details',
          icon: MenuIcons.edit,
          onClick: () => setSelectedNode(node),
        },
      ];
    }

    return [];
  };

  // Handle delete
  const handleDelete = async () => {
    if (!deleteDialog.node) return;

    setDeleteDialog((prev) => ({ ...prev, isLoading: true }));

    try {
      if (deleteDialog.node.type === 'folder') {
        await folderApi.delete(deleteDialog.node.id);
      } else if (deleteDialog.node.type === 'project') {
        await projectApi.delete(deleteDialog.node.id);
      }

      // Clear selection if deleted node was selected
      if (selectedNode?.id === deleteDialog.node.id) {
        setSelectedNode(null);
      }

      refresh();
      setDeleteDialog({ isOpen: false, node: null, isLoading: false });
    } catch (error) {
      console.error('Delete failed:', error);
      setDeleteDialog((prev) => ({ ...prev, isLoading: false }));
    }
  };

  // Handle edit from detail panel
  const handleEdit = () => {
    if (!selectedNode) return;

    if (selectedNode.type === 'folder') {
      setFolderDialog({ isOpen: true, folder: selectedNode.data as Folder });
    } else if (selectedNode.type === 'project') {
      setProjectDialog({ isOpen: true, project: selectedNode.data as Project });
    }
  };

  // Handle delete from detail panel
  const handleDeleteFromPanel = () => {
    if (!selectedNode) return;
    setDeleteDialog({ isOpen: true, node: selectedNode, isLoading: false });
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
          <button
            onClick={() => setFolderDialog({ 
              isOpen: true, 
              parentId: selectedNode?.type === 'folder' ? selectedNode.id : undefined 
            })}
            className="p-1.5 text-slate-500 hover:text-slate-700 hover:bg-slate-100 rounded transition-colors"
            title={selectedNode?.type === 'folder' ? 'New Subfolder' : 'New Folder'}
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
            className="p-1.5 text-slate-500 hover:text-slate-700 hover:bg-slate-100 rounded transition-colors"
            title={selectedNode?.type === 'folder' ? `New Project in "${selectedNode.name}"` : 'New Project'}
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 13h6m-3-3v6m5 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
          </button>
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
        />
      </div>
    </div>
  );

  // Sidebar content with tabs
  const sidebar = (
    <div className="flex flex-col h-full">
      {/* Tab bar */}
      <div className="flex border-b border-slate-200">
        <button
          onClick={() => setActiveTab('explorer')}
          className={`flex-1 px-4 py-2.5 text-sm font-medium transition-colors relative ${
            activeTab === 'explorer'
              ? 'text-blue-600'
              : 'text-slate-500 hover:text-slate-700 hover:bg-slate-50'
          }`}
        >
          <div className="flex items-center justify-center space-x-2">
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
            </svg>
            <span>Explorer</span>
          </div>
          {activeTab === 'explorer' && (
            <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600" />
          )}
        </button>
        <button
          onClick={() => setActiveTab('datasource')}
          className={`flex-1 px-4 py-2.5 text-sm font-medium transition-colors relative ${
            activeTab === 'datasource'
              ? 'text-blue-600'
              : 'text-slate-500 hover:text-slate-700 hover:bg-slate-50'
          }`}
        >
          <div className="flex items-center justify-center space-x-2">
            <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4" />
            </svg>
            <span>Data</span>
          </div>
          {activeTab === 'datasource' && (
            <div className="absolute bottom-0 left-0 right-0 h-0.5 bg-blue-600" />
          )}
        </button>
      </div>

      {/* Tab content */}
      <div className="flex-1 overflow-hidden">
        {activeTab === 'explorer' ? explorerContent : <DataSourcePanel />}
      </div>
    </div>
  );

  return (
    <Layout sidebar={sidebar}>
      <DetailPanel
        node={selectedNode}
        onEdit={handleEdit}
        onDelete={handleDeleteFromPanel}
        onBuildModel={
          selectedNode?.type === 'project'
            ? () => setBuildDialog({ isOpen: true, projectId: selectedNode.id, projectName: selectedNode.name })
            : selectedNode?.type === 'folder'
            ? () => setBuildDialog({ isOpen: true, folderId: selectedNode.id, folderName: selectedNode.name })
            : undefined
        }
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
        onSuccess={refresh}
        parentId={folderDialog.parentId}
        folder={folderDialog.folder}
      />

      {/* Project Dialog */}
      <ProjectDialog
        isOpen={projectDialog.isOpen}
        onClose={() => setProjectDialog({ isOpen: false })}
        onSuccess={refresh}
        folderId={projectDialog.folderId}
        folderName={projectDialog.folderName}
        project={projectDialog.project}
      />

      {/* Delete Confirmation Dialog */}
      <ConfirmDialog
        isOpen={deleteDialog.isOpen}
        onClose={() => setDeleteDialog({ isOpen: false, node: null, isLoading: false })}
        onConfirm={handleDelete}
        title={`Delete ${deleteDialog.node?.type || 'item'}?`}
        message={`Are you sure you want to delete "${deleteDialog.node?.name}"? This action cannot be undone.`}
        confirmText="Delete"
        isLoading={deleteDialog.isLoading}
      />

      {/* Build Model Dialog */}
      <BuildModelDialog
        isOpen={buildDialog.isOpen}
        onClose={() => setBuildDialog({ isOpen: false })}
        onSuccess={refresh}
        projectId={buildDialog.projectId}
        projectName={buildDialog.projectName}
        folderId={buildDialog.folderId}
        folderName={buildDialog.folderName}
      />
    </Layout>
  );
}

