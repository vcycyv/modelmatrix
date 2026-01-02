import { useState, useEffect, useCallback, MouseEvent } from 'react';
import { folderApi, projectApi, buildApi, modelApi, Folder, Project, ModelBuild, Model } from '../lib/api';

// Tree node types
export type TreeNodeType = 'folder' | 'project' | 'build' | 'model';

export interface TreeNode {
  id: string;
  name: string;
  type: TreeNodeType;
  data: Folder | Project | ModelBuild | Model;
  children?: TreeNode[];
  isExpanded?: boolean;
  isLoading?: boolean;
}

interface TreeViewProps {
  onSelect: (node: TreeNode) => void;
  selectedId?: string;
  onContextMenu: (e: MouseEvent, node: TreeNode) => void;
  refreshTrigger?: number;
}

// Icons for different node types
const NodeIcon = ({ type, isExpanded }: { type: TreeNodeType; isExpanded?: boolean }) => {
  switch (type) {
    case 'folder':
      return isExpanded ? (
        <svg className="w-5 h-5 text-amber-500" fill="currentColor" viewBox="0 0 20 20">
          <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
        </svg>
      ) : (
        <svg className="w-5 h-5 text-amber-500" fill="currentColor" viewBox="0 0 20 20">
          <path d="M2 6a2 2 0 012-2h5l2 2h5a2 2 0 012 2v6a2 2 0 01-2 2H4a2 2 0 01-2-2V6z" />
        </svg>
      );
    case 'project':
      return (
        <svg className="w-5 h-5 text-blue-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
        </svg>
      );
    case 'build':
      return (
        <svg className="w-5 h-5 text-purple-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10.325 4.317c.426-1.756 2.924-1.756 3.35 0a1.724 1.724 0 002.573 1.066c1.543-.94 3.31.826 2.37 2.37a1.724 1.724 0 001.065 2.572c1.756.426 1.756 2.924 0 3.35a1.724 1.724 0 00-1.066 2.573c.94 1.543-.826 3.31-2.37 2.37a1.724 1.724 0 00-2.572 1.065c-.426 1.756-2.924 1.756-3.35 0a1.724 1.724 0 00-2.573-1.066c-1.543.94-3.31-.826-2.37-2.37a1.724 1.724 0 00-1.065-2.572c-1.756-.426-1.756-2.924 0-3.35a1.724 1.724 0 001.066-2.573c-.94-1.543.826-3.31 2.37-2.37.996.608 2.296.07 2.572-1.065z" />
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
        </svg>
      );
    case 'model':
      return (
        <svg className="w-5 h-5 text-green-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
        </svg>
      );
  }
};

// Chevron icon for expandable nodes
const ChevronIcon = ({ isExpanded, isLoading }: { isExpanded: boolean; isLoading?: boolean }) => {
  if (isLoading) {
    return (
      <svg className="w-4 h-4 text-slate-400 animate-spin" fill="none" viewBox="0 0 24 24">
        <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
        <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
      </svg>
    );
  }
  return (
    <svg
      className={`w-4 h-4 text-slate-400 transition-transform ${isExpanded ? 'rotate-90' : ''}`}
      fill="none"
      viewBox="0 0 24 24"
      stroke="currentColor"
    >
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 5l7 7-7 7" />
    </svg>
  );
};

// Tree node component
interface TreeNodeItemProps {
  node: TreeNode;
  level: number;
  selectedId?: string;
  onSelect: (node: TreeNode) => void;
  onToggle: (node: TreeNode) => void;
  onContextMenu: (e: MouseEvent, node: TreeNode) => void;
}

function TreeNodeItem({ node, level, selectedId, onSelect, onToggle, onContextMenu }: TreeNodeItemProps) {
  const isSelected = node.id === selectedId;
  const hasChildren = node.type === 'folder' || node.type === 'project';
  const paddingLeft = 12 + level * 20;

  const handleClick = () => {
    onSelect(node);
    if (hasChildren) {
      onToggle(node);
    }
  };

  return (
    <div>
      <div
        data-tree-node="true"
        className={`flex items-center py-1.5 px-2 cursor-pointer transition-colors ${
          isSelected
            ? 'bg-blue-100 text-blue-900'
            : 'hover:bg-slate-100 text-slate-700'
        }`}
        style={{ paddingLeft }}
        onClick={handleClick}
        onContextMenu={(e) => {
          e.stopPropagation();
          onContextMenu(e, node);
        }}
      >
        {hasChildren && (
          <span className="mr-1">
            <ChevronIcon isExpanded={node.isExpanded || false} isLoading={node.isLoading} />
          </span>
        )}
        {!hasChildren && <span className="w-4 mr-1" />}
        <span className="mr-2">
          <NodeIcon type={node.type} isExpanded={node.isExpanded} />
        </span>
        <span className="truncate text-sm font-medium">{node.name}</span>
      </div>

      {node.isExpanded && node.children && (
        <div>
          {node.children.map((child) => (
            <TreeNodeItem
              key={`${child.type}-${child.id}`}
              node={child}
              level={level + 1}
              selectedId={selectedId}
              onSelect={onSelect}
              onToggle={onToggle}
              onContextMenu={onContextMenu}
            />
          ))}
        </div>
      )}
    </div>
  );
}

export default function TreeView({ onSelect, selectedId, onContextMenu, refreshTrigger }: TreeViewProps) {
  const [rootNodes, setRootNodes] = useState<TreeNode[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  // Load root folders and projects
  const loadRootData = useCallback(async () => {
    setIsLoading(true);
    try {
      const folders = await folderApi.getRootFolders();
      const folderNodes: TreeNode[] = folders.map((folder) => ({
        id: folder.id,
        name: folder.name,
        type: 'folder' as TreeNodeType,
        data: folder,
        isExpanded: false,
      }));

      // Also load root projects (not in any folder)
      try {
        const projects = await projectApi.getRootProjects();
        const projectNodes: TreeNode[] = projects.map((project) => ({
          id: project.id,
          name: project.name,
          type: 'project' as TreeNodeType,
          data: project,
          isExpanded: false,
        }));
        setRootNodes([...folderNodes, ...projectNodes]);
      } catch {
        // If root projects API fails, just show folders
        setRootNodes(folderNodes);
      }
    } catch (error) {
      console.error('Failed to load tree data:', error);
      setRootNodes([]);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    loadRootData();
  }, [loadRootData, refreshTrigger]);

  // Load children for a node
  const loadChildren = async (node: TreeNode): Promise<TreeNode[]> => {
    if (node.type === 'folder') {
      // Load subfolders, projects, and direct builds/models in folder
      const [childFolders, childProjects, folderBuilds, folderModels] = await Promise.all([
        folderApi.getChildren(node.id),
        projectApi.getProjectsInFolder(node.id),
        folderApi.getBuilds(node.id).catch(() => [] as ModelBuild[]),
        folderApi.getModels(node.id).catch(() => [] as Model[]),
      ]);

      const folderNodes: TreeNode[] = childFolders.map((folder) => ({
        id: folder.id,
        name: folder.name,
        type: 'folder' as TreeNodeType,
        data: folder,
        isExpanded: false,
      }));

      const projectNodes: TreeNode[] = childProjects.map((project) => ({
        id: project.id,
        name: project.name,
        type: 'project' as TreeNodeType,
        data: project,
        isExpanded: false,
      }));

      const buildNodes: TreeNode[] = folderBuilds.map((build) => ({
        id: build.id,
        name: build.name,
        type: 'build' as TreeNodeType,
        data: build,
      }));

      const modelNodes: TreeNode[] = folderModels.map((model) => ({
        id: model.id,
        name: model.name,
        type: 'model' as TreeNodeType,
        data: model,
      }));

      return [...folderNodes, ...projectNodes, ...buildNodes, ...modelNodes];
    }

    if (node.type === 'project') {
      // Load builds and models in project
      const [projectBuilds, projectModels] = await Promise.all([
        buildApi.getBuildsInProject(node.id).catch(() => [] as ModelBuild[]),
        modelApi.getModelsInProject(node.id).catch(() => [] as Model[]),
      ]);

      const buildNodes: TreeNode[] = projectBuilds.map((build) => ({
        id: build.id,
        name: build.name,
        type: 'build' as TreeNodeType,
        data: build,
      }));

      const modelNodes: TreeNode[] = projectModels.map((model) => ({
        id: model.id,
        name: model.name,
        type: 'model' as TreeNodeType,
        data: model,
      }));

      return [...buildNodes, ...modelNodes];
    }

    return [];
  };

  // Toggle node expansion
  const handleToggle = async (targetNode: TreeNode) => {
    const updateNode = (nodes: TreeNode[]): TreeNode[] => {
      return nodes.map((node) => {
        if (node.id === targetNode.id && node.type === targetNode.type) {
          if (!node.isExpanded && !node.children) {
            // Need to load children
            return { ...node, isLoading: true };
          }
          return { ...node, isExpanded: !node.isExpanded };
        }
        if (node.children) {
          return { ...node, children: updateNode(node.children) };
        }
        return node;
      });
    };

    setRootNodes(updateNode(rootNodes));

    // Load children if needed
    if (!targetNode.isExpanded && !targetNode.children) {
      try {
        const children = await loadChildren(targetNode);
        setRootNodes((prev) => {
          const updateWithChildren = (nodes: TreeNode[]): TreeNode[] => {
            return nodes.map((node) => {
              if (node.id === targetNode.id && node.type === targetNode.type) {
                return { ...node, children, isExpanded: true, isLoading: false };
              }
              if (node.children) {
                return { ...node, children: updateWithChildren(node.children) };
              }
              return node;
            });
          };
          return updateWithChildren(prev);
        });
      } catch (error) {
        console.error('Failed to load children:', error);
        setRootNodes((prev) => {
          const updateWithError = (nodes: TreeNode[]): TreeNode[] => {
            return nodes.map((node) => {
              if (node.id === targetNode.id && node.type === targetNode.type) {
                return { ...node, isLoading: false };
              }
              if (node.children) {
                return { ...node, children: updateWithError(node.children) };
              }
              return node;
            });
          };
          return updateWithError(prev);
        });
      }
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full text-slate-500">
        <svg className="animate-spin h-6 w-6 mr-2" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
        </svg>
        Loading...
      </div>
    );
  }

  if (rootNodes.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-full text-slate-500 p-4">
        <svg className="w-12 h-12 mb-2 text-slate-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
        </svg>
        <p className="text-sm">No folders or projects</p>
        <p className="text-xs text-slate-400 mt-1">Right-click to create</p>
      </div>
    );
  }

  return (
    <div className="py-2">
      {rootNodes.map((node) => (
        <TreeNodeItem
          key={`${node.type}-${node.id}`}
          node={node}
          level={0}
          selectedId={selectedId}
          onSelect={onSelect}
          onToggle={handleToggle}
          onContextMenu={onContextMenu}
        />
      ))}
    </div>
  );
}

