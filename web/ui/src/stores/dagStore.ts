import { createStore, produce } from 'solid-js/store';
import type { DAGNode, DAGEdge, NodeType } from '../types/dag';

/**
 * Defines which node types can connect to which other types.
 * source → query, transform, function, target
 * query → transform, function, target
 * transform → function, target
 * function → transform, function, target
 * target → (nothing, no output port)
 */
const validConnections: Record<string, Set<string>> = {
  source:    new Set(['query', 'transform', 'function', 'target']),
  query:     new Set(['transform', 'function', 'target']),
  transform: new Set(['function', 'target']),
  function:  new Set(['transform', 'function', 'target']),
  target:    new Set(),
};

export function isValidConnection(sourceType: string, targetType: string): boolean {
  return validConnections[sourceType]?.has(targetType) ?? false;
}

export interface DAGStore {
  nodes: DAGNode[];
  edges: DAGEdge[];
  selectedNodeId: string | null;
  isDirty: boolean;
}

const [state, setState] = createStore<DAGStore>({
  nodes: [],
  edges: [],
  selectedNodeId: null,
  isDirty: false,
});

let nextId = 1;
function genId() {
  return `node_${nextId++}`;
}

export function useDagStore() {
  return {
    state,

    addNode(type: NodeType, x: number, y: number, label?: string) {
      const id = genId();
      const defaultConfigs: Record<string, Record<string, any>> = {
        source: { storageType: 'sqlite3', connection: {} },
        target: { storageType: 'sqlite3', connection: {} },
        query: { sql: '' },
        transform: { expression: '', fieldMappings: [] },
        function: { goCode: '' },
      };
      const node: DAGNode = {
        id,
        type,
        label: label || `${type}_${id}`,
        x,
        y,
        config: defaultConfigs[type] || {},
      };
      setState(produce(s => {
        s.nodes.push(node);
        s.isDirty = true;
      }));
      return id;
    },

    removeNode(id: string) {
      setState(produce(s => {
        s.nodes = s.nodes.filter(n => n.id !== id);
        s.edges = s.edges.filter(e => e.source !== id && e.target !== id);
        if (s.selectedNodeId === id) s.selectedNodeId = null;
        s.isDirty = true;
      }));
    },

    updateNodeConfig(id: string, config: Record<string, any>) {
      setState(produce(s => {
        const node = s.nodes.find(n => n.id === id);
        if (node) {
          node.config = { ...node.config, ...config };
          s.isDirty = true;
        }
      }));
    },

    updateNodeLabel(id: string, label: string) {
      setState(produce(s => {
        const node = s.nodes.find(n => n.id === id);
        if (node) {
          node.label = label;
          s.isDirty = true;
        }
      }));
    },

    moveNode(id: string, x: number, y: number) {
      setState(produce(s => {
        const node = s.nodes.find(n => n.id === id);
        if (node) {
          node.x = x;
          node.y = y;
        }
      }));
    },

    addEdge(sourceId: string, targetId: string) {
      // Prevent duplicate edges
      if (state.edges.some(e => e.source === sourceId && e.target === targetId)) return;

      // Validate connection types
      const srcNode = state.nodes.find(n => n.id === sourceId);
      const tgtNode = state.nodes.find(n => n.id === targetId);
      if (!srcNode || !tgtNode) return;
      if (!isValidConnection(srcNode.type, tgtNode.type)) return;

      const id = `edge_${sourceId}_${targetId}`;
      setState(produce(s => {
        s.edges.push({ id, source: sourceId, target: targetId });
        s.isDirty = true;
      }));
    },

    removeEdge(id: string) {
      setState(produce(s => {
        s.edges = s.edges.filter(e => e.id !== id);
        s.isDirty = true;
      }));
    },

    selectNode(id: string | null) {
      setState('selectedNodeId', id);
    },

    loadDag(nodes: DAGNode[], edges: DAGEdge[]) {
      // Reset ID counter
      const maxId = nodes.reduce((max, n) => {
        const num = parseInt(n.id.replace('node_', ''));
        return isNaN(num) ? max : Math.max(max, num);
      }, 0);
      nextId = maxId + 1;

      setState({
        nodes: [...nodes],
        edges: [...edges],
        selectedNodeId: null,
        isDirty: false,
      });
    },

    clearDirty() {
      setState('isDirty', false);
    },
  };
}
