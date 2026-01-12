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
  refreshNodeId?: string; // ID of a specific node to refresh (folder or project)
  refreshNodeType?: TreeNodeType; // Type of the node to refresh
  onNodeRefreshed?: () => void; // Callback when node refresh is complete
  hideBuilds?: boolean; // Filter out build nodes from display
  hideModels?: boolean; // Filter out model nodes from display
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
  isRefreshing?: boolean;
  refreshingNodes?: Set<string>;
}

function TreeNodeItem({ node, level, selectedId, onSelect, onToggle, onContextMenu, isRefreshing, refreshingNodes }: TreeNodeItemProps) {
  const isSelected = node.id === selectedId;
  const hasChildren = node.type === 'folder' || node.type === 'project';
  const paddingLeft = 12 + level * 20;

  const handleClick = () => {
    onSelect(node);
    if (hasChildren) {
      onToggle(node);
    }
  };

  // Helper to check if a child node is refreshing
  const isChildRefreshing = (child: TreeNode) => {
    return refreshingNodes?.has(`${child.type}-${child.id}`) || false;
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

      {node.isExpanded && (
        <div>
          {/* Show refresh indicator */}
          {isRefreshing && (
            <div
              className="flex items-center py-1 px-2 text-blue-500 bg-blue-50/50"
              style={{ paddingLeft: paddingLeft + 20 }}
            >
              <svg className="animate-spin h-3 w-3 mr-1.5" fill="none" viewBox="0 0 24 24">
                <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z"></path>
              </svg>
              <span className="text-xs">Refreshing...</span>
            </div>
          )}
          {/* Show children (keep showing old ones during refresh) */}
          {node.children?.map((child) => (
            <TreeNodeItem
              key={`${child.type}-${child.id}`}
              node={child}
              level={level + 1}
              selectedId={selectedId}
              onSelect={onSelect}
              onToggle={onToggle}
              onContextMenu={onContextMenu}
              isRefreshing={isChildRefreshing(child)}
              refreshingNodes={refreshingNodes}
            />
          ))}
        </div>
      )}
    </div>
  );
}

export default function TreeView({ onSelect, selectedId, onContextMenu, refreshTrigger, refreshNodeId, refreshNodeType, onNodeRefreshed, hideBuilds = false, hideModels = false }: TreeViewProps) {
  const [rootNodes, setRootNodes] = useState<TreeNode[]>([]);
  const [isInitialLoad, setIsInitialLoad] = useState(true);
  // Track expanded nodes by "type-id" key to preserve state across refreshes
  const [expandedNodes, setExpandedNodes] = useState<Set<string>>(new Set());
  // Track which nodes are currently refreshing
  const [refreshingNodes, setRefreshingNodes] = useState<Set<string>>(new Set());

  // Helper to create node key
  const nodeKey = (type: TreeNodeType, id: string) => `${type}-${id}`;

  // Load children for expanded nodes recursively
  const loadExpandedChildren = async (nodes: TreeNode[], expanded: Set<string>): Promise<TreeNode[]> => {
    const result: TreeNode[] = [];
    
    for (const node of nodes) {
      const key = nodeKey(node.type, node.id);
      if (expanded.has(key) && (node.type === 'folder' || node.type === 'project')) {
        // This node should be expanded, load its children
        try {
          const children = await loadChildrenForNode(node, expanded);
          result.push({
            ...node,
            isExpanded: true,
            children: await loadExpandedChildren(children, expanded),
          });
        } catch {
          result.push({ ...node, isExpanded: true });
        }
      } else {
        result.push(node);
      }
    }
    
    return result;
  };

  // Load children for a specific node (without using expandedNodes state to avoid stale closure)
  const loadChildrenForNode = async (node: TreeNode, expanded: Set<string>): Promise<TreeNode[]> => {
    if (node.type === 'folder') {
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
        isExpanded: expanded.has(nodeKey('folder', folder.id)),
      }));

      const projectNodes: TreeNode[] = childProjects.map((project) => ({
        id: project.id,
        name: project.name,
        type: 'project' as TreeNodeType,
        data: project,
        isExpanded: expanded.has(nodeKey('project', project.id)),
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
      const [projectBuilds, projectModels] = await Promise.all([
        buildApi.getBuildsInProject(node.id).catch(() => [] as ModelBuild[]),
        modelApi.getModelsInProject(node.id).catch(() => [] as Model[]),
      ]);

      return [
        ...projectBuilds.map((build) => ({
          id: build.id,
          name: build.name,
          type: 'build' as TreeNodeType,
          data: build,
        })),
        ...projectModels.map((model) => ({
          id: model.id,
          name: model.name,
          type: 'model' as TreeNodeType,
          data: model,
        })),
      ];
    }

    return [];
  };

  // Load root folders and projects
  const loadRootData = useCallback(async (expanded: Set<string>, isRefresh: boolean) => {
    // Mark expanded nodes as refreshing (only on refresh, not initial load)
    if (isRefresh && expanded.size > 0) {
      setRefreshingNodes(new Set(expanded));
    }

    try {
      const folders = await folderApi.getRootFolders();
      let allNodes: TreeNode[] = folders.map((folder) => ({
        id: folder.id,
        name: folder.name,
        type: 'folder' as TreeNodeType,
        data: folder,
        isExpanded: expanded.has(nodeKey('folder', folder.id)),
      }));

      // Also load root projects (not in any folder)
      try {
        const projects = await projectApi.getRootProjects();
        const projectNodes: TreeNode[] = projects.map((project) => ({
          id: project.id,
          name: project.name,
          type: 'project' as TreeNodeType,
          data: project,
          isExpanded: expanded.has(nodeKey('project', project.id)),
        }));
        allNodes = [...allNodes, ...projectNodes];
      } catch {
        // If root projects API fails, just show folders
      }

      // Load children for expanded nodes
      const nodesWithChildren = await loadExpandedChildren(allNodes, expanded);
      setRootNodes(nodesWithChildren);
    } catch (error) {
      console.error('Failed to load tree data:', error);
      if (!isRefresh) {
        setRootNodes([]);
      }
    } finally {
      setIsInitialLoad(false);
      setRefreshingNodes(new Set());
    }
  }, []);

  useEffect(() => {
    const isRefresh = !isInitialLoad && rootNodes.length > 0;
    loadRootData(expandedNodes, isRefresh);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refreshTrigger]); // Only trigger on refreshTrigger change (full tree refresh)

  // Handle single node refresh
  useEffect(() => {
    if (!refreshNodeId || !refreshNodeType) return;

    const refreshSingleNode = async () => {
      const key = nodeKey(refreshNodeType, refreshNodeId);
      setRefreshingNodes(new Set([key]));

      try {
        // Find and refresh the node
        const updateNodeInTree = async (nodes: TreeNode[]): Promise<TreeNode[]> => {
          const result: TreeNode[] = [];
          for (const node of nodes) {
            if (node.id === refreshNodeId && node.type === refreshNodeType) {
              // Found the node - reload its children
              const children = await loadChildren(node);
              // Also load children for any expanded child nodes
              const childrenWithExpanded = await loadExpandedChildren(children, expandedNodes);
              result.push({
                ...node,
                children: childrenWithExpanded,
                isExpanded: true,
              });
              // Ensure node is marked as expanded
              setExpandedNodes((prev) => new Set([...prev, key]));
            } else if (node.children && node.children.length > 0) {
              // Check children recursively
              const updatedChildren = await updateNodeInTree(node.children);
              result.push({ ...node, children: updatedChildren });
            } else {
              result.push(node);
            }
          }
          return result;
        };

        const updatedNodes = await updateNodeInTree(rootNodes);
        setRootNodes(updatedNodes);
      } catch (error) {
        console.error('Failed to refresh node:', error);
      } finally {
        setRefreshingNodes(new Set());
        onNodeRefreshed?.();
      }
    };

    refreshSingleNode();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [refreshNodeId, refreshNodeType]);

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
        isExpanded: expandedNodes.has(nodeKey('folder', folder.id)),
      }));

      const projectNodes: TreeNode[] = childProjects.map((project) => ({
        id: project.id,
        name: project.name,
        type: 'project' as TreeNodeType,
        data: project,
        isExpanded: expandedNodes.has(nodeKey('project', project.id)),
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
    const key = nodeKey(targetNode.type, targetNode.id);
    const willExpand = !targetNode.isExpanded;

    // Update expandedNodes set
    setExpandedNodes((prev) => {
      const next = new Set(prev);
      if (willExpand) {
        next.add(key);
      } else {
        next.delete(key);
      }
      return next;
    });

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

  // Filter nodes based on hideBuilds and hideModels settings
  // Must be defined before any early returns to maintain hooks order
  const filterNodes = useCallback((nodes: TreeNode[]): TreeNode[] => {
    const filterRecursive = (nodeList: TreeNode[]): TreeNode[] => {
      return nodeList
        .filter((node) => {
          if (hideBuilds && node.type === 'build') return false;
          if (hideModels && node.type === 'model') return false;
          return true;
        })
        .map((node) => {
          if (node.children) {
            return { ...node, children: filterRecursive(node.children) };
          }
          return node;
        });
    };
    return filterRecursive(nodes);
  }, [hideBuilds, hideModels]);

  // Helper to check if a node is being refreshed
  const isNodeRefreshing = useCallback((node: TreeNode) => {
    return refreshingNodes.has(nodeKey(node.type, node.id));
  }, [refreshingNodes]);

  // Only show full-page loading on initial load
  if (isInitialLoad && rootNodes.length === 0) {
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

  if (!isInitialLoad && rootNodes.length === 0) {
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

  const filteredNodes = filterNodes(rootNodes);

  return (
    <div className="py-2">
      {filteredNodes.map((node) => (
        <TreeNodeItem
          key={`${node.type}-${node.id}`}
          node={node}
          level={0}
          selectedId={selectedId}
          onSelect={onSelect}
          onToggle={handleToggle}
          onContextMenu={onContextMenu}
          isRefreshing={isNodeRefreshing(node)}
          refreshingNodes={refreshingNodes}
        />
      ))}
    </div>
  );
}

