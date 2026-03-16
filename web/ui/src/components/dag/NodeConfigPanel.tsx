import { createSignal, Show, Switch, Match } from 'solid-js';
import type { DAGNode, DAGEdge } from '../../types/dag';
import SourceNodeConfig from './SourceNodeConfig';
import TargetNodeConfig from './TargetNodeConfig';
import TransformNodeConfig from './TransformNodeConfig';
import QueryNodeConfig from './QueryNodeConfig';
import FunctionNodeConfig from './FunctionNodeConfig';
import ResizeHandle from '../common/ResizeHandle';
import Button from '../common/Button';

interface NodeConfigPanelProps {
  node: DAGNode | undefined;
  nodes: DAGNode[];
  edges: DAGEdge[];
  onUpdateConfig: (id: string, config: Record<string, any>) => void;
  onUpdateLabel: (id: string, label: string) => void;
  onDelete: (id: string) => void;
  onClose: () => void;
}

/** Find the first source node connected to a given node (walking backward through edges). */
function findConnectedSource(nodeId: string, nodes: DAGNode[], edges: DAGEdge[]): DAGNode | undefined {
  const nodeMap = new Map(nodes.map(n => [n.id, n]));
  const visited = new Set<string>();

  function walk(id: string): DAGNode | undefined {
    if (visited.has(id)) return undefined;
    visited.add(id);

    for (const edge of edges) {
      if (edge.target === id) {
        const src = nodeMap.get(edge.source);
        if (!src) continue;
        if (src.type === 'source') return src;
        const deeper = walk(edge.source);
        if (deeper) return deeper;
      }
    }
    return undefined;
  }

  return walk(nodeId);
}

/** Find the first target node connected to a given node (walking forward through edges). */
function findConnectedTarget(nodeId: string, nodes: DAGNode[], edges: DAGEdge[]): DAGNode | undefined {
  const nodeMap = new Map(nodes.map(n => [n.id, n]));
  const visited = new Set<string>();

  function walk(id: string): DAGNode | undefined {
    if (visited.has(id)) return undefined;
    visited.add(id);

    for (const edge of edges) {
      if (edge.source === id) {
        const tgt = nodeMap.get(edge.target);
        if (!tgt) continue;
        if (tgt.type === 'target') return tgt;
        const deeper = walk(edge.target);
        if (deeper) return deeper;
      }
    }
    return undefined;
  }

  return walk(nodeId);
}

const DEFAULT_WIDTHS: Record<string, number> = {
  transform: 480,
  query: 480,
  function: 480,
  source: 320,
  target: 320,
};

export default function NodeConfigPanel(props: NodeConfigPanelProps) {
  const [width, setWidth] = createSignal(320);

  // Reset width when node type changes
  const nodeType = () => props.node?.type || 'source';
  let lastType = nodeType();
  const effectiveWidth = () => {
    const t = nodeType();
    if (t !== lastType) {
      lastType = t;
      setWidth(DEFAULT_WIDTHS[t] || 320);
    }
    return width();
  };

  // Derive schema connection for query nodes from their connected source
  const querySchemaConfig = () => {
    const node = props.node;
    if (!node || node.type !== 'query') return node?.config || {};

    const sourceNode = findConnectedSource(node.id, props.nodes, props.edges);
    if (sourceNode) {
      return {
        ...node.config,
        schemaType: sourceNode.config.storageType,
        schemaConnection: sourceNode.config.connection,
      };
    }
    return node.config;
  };

  // Derive source/target connection info for transform nodes (for auto-map)
  const transformAutoMapConfig = () => {
    const node = props.node;
    if (!node || node.type !== 'transform') return node?.config || {};

    const config = { ...node.config };

    const sourceNode = findConnectedSource(node.id, props.nodes, props.edges);
    if (sourceNode) {
      config.autoMapSource = {
        type: sourceNode.config.storageType,
        connection: sourceNode.config.connection,
        table: '', // flat sources use empty table
      };
    }

    const targetNode = findConnectedTarget(node.id, props.nodes, props.edges);
    if (targetNode) {
      config.autoMapTarget = {
        type: targetNode.config.storageType,
        connection: targetNode.config.connection,
        table: targetNode.config.table || '',
      };
    }

    return config;
  };

  function handleResize(delta: number) {
    setWidth(w => Math.max(240, Math.min(800, w - delta)));
  }

  return (
    <Show when={props.node}>
      {(node) => (
        <div class="flex h-full flex-shrink-0">
          <ResizeHandle direction="horizontal" onResize={handleResize} />
          <div
            class="border-l border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 flex flex-col h-full overflow-y-auto"
            style={{ width: `${effectiveWidth()}px` }}
          >
            <div class="p-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
              <span class="text-sm font-semibold text-gray-900 dark:text-white capitalize">
                {node().type} Config
              </span>
              <button onClick={props.onClose} class="text-gray-400 hover:text-gray-600 cursor-pointer">x</button>
            </div>

            <div class="p-4 space-y-4 flex-1 overflow-y-auto">
              {/* Name / label */}
              <div>
                <label class="block text-xs font-medium text-gray-500 mb-1">Name</label>
                <input
                  type="text"
                  value={node().label}
                  onInput={(e) => props.onUpdateLabel(node().id, e.currentTarget.value)}
                  class="w-full px-2 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                />
              </div>

              {/* Type-specific config */}
              <Switch>
                <Match when={node().type === 'source'}>
                  <SourceNodeConfig
                    config={node().config}
                    onUpdate={(cfg) => props.onUpdateConfig(node().id, cfg)}
                  />
                </Match>
                <Match when={node().type === 'target'}>
                  <TargetNodeConfig
                    config={node().config}
                    onUpdate={(cfg) => props.onUpdateConfig(node().id, cfg)}
                  />
                </Match>
                <Match when={node().type === 'transform'}>
                  <TransformNodeConfig
                    config={transformAutoMapConfig()}
                    onUpdate={(cfg) => props.onUpdateConfig(node().id, cfg)}
                  />
                </Match>
                <Match when={node().type === 'function'}>
                  <FunctionNodeConfig
                    config={node().config}
                    onUpdate={(cfg) => props.onUpdateConfig(node().id, cfg)}
                  />
                </Match>
                <Match when={node().type === 'query'}>
                  <QueryNodeConfig
                    config={querySchemaConfig()}
                    onUpdate={(cfg) => props.onUpdateConfig(node().id, cfg)}
                  />
                </Match>
              </Switch>
            </div>

            <div class="p-4 border-t border-gray-200 dark:border-gray-700 flex-shrink-0">
              <Button variant="danger" size="sm" class="w-full" onClick={() => props.onDelete(node().id)}>
                Delete Node
              </Button>
            </div>
          </div>
        </div>
      )}
    </Show>
  );
}
