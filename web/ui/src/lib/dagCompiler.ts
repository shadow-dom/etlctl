import type { DAGNode, DAGEdge } from '../types/dag';
import type { ETLConfig, PipelineConfig, StorageConfig, QueryConfig, FunctionConfig, FieldMapping } from '../types/etl';

/**
 * Compiles a visual DAG (nodes + edges) into the flat ETL YAML schema.
 *
 * Source nodes become ETL.sources, target nodes become ETL.targets,
 * query nodes become ETL.queries, function nodes become ETL.functions.
 * Paths from source -> [query] -> [transform] -> [function] -> target
 * become pipelines. Query nodes define the SQL used during extraction and are
 * traversed to discover upstream sources.
 *
 * Targets that share the same upstream path (sources, transforms, query) are
 * merged into a single pipeline with multiple targets to avoid redundant extraction.
 */
export function compileDag(
  name: string,
  nodes: DAGNode[],
  edges: DAGEdge[],
): ETLConfig {
  const sources: StorageConfig[] = [];
  const targets: StorageConfig[] = [];
  const queries: QueryConfig[] = [];
  const functions: FunctionConfig[] = [];
  const pipelines: PipelineConfig[] = [];

  // Index nodes
  const nodeMap = new Map(nodes.map(n => [n.id, n]));
  const outEdges = new Map<string, string[]>();
  const inEdges = new Map<string, string[]>();

  for (const edge of edges) {
    if (!outEdges.has(edge.source)) outEdges.set(edge.source, []);
    outEdges.get(edge.source)!.push(edge.target);
    if (!inEdges.has(edge.target)) inEdges.set(edge.target, []);
    inEdges.get(edge.target)!.push(edge.source);
  }

  // Collect sources, targets, queries, and functions
  for (const node of nodes) {
    if (node.type === 'source') {
      const src: StorageConfig = {
        name: node.label,
        type: node.config.storageType || 'sqlite3',
        connection: node.config.connection || {},
      };
      if (node.config.connectionRef) {
        src.connection_ref = node.config.connectionRef;
      }
      sources.push(src);
    } else if (node.type === 'target') {
      const tgt: StorageConfig = {
        name: node.label,
        type: node.config.storageType || 'sqlite3',
        connection: node.config.connection || {},
      };
      if (node.config.connectionRef) {
        tgt.connection_ref = node.config.connectionRef;
      }
      targets.push(tgt);
    } else if (node.type === 'query') {
      queries.push({
        name: node.label,
        sql: node.config.sql || '',
      });
    } else if (node.type === 'function') {
      functions.push({
        name: node.label,
        code: node.config.goCode || '',
      });
    }
  }

  // Build pipelines by tracing paths from target nodes backward.
  // Targets that share the same upstream path (sources, transforms, query) are
  // merged into a single pipeline with multiple targets to avoid redundant extraction.
  const targetNodes = nodes.filter(n => n.type === 'target');

  // First pass: compute upstream info per target
  type TargetUpstream = {
    targetNode: DAGNode;
    pipelineSources: string[];
    transforms: DAGNode[];
    functionNames: string[];
    queryName: string;
    fields: FieldMapping[];
    fingerprint: string; // for grouping
  };
  const upstreams: TargetUpstream[] = [];

  for (const targetNode of targetNodes) {
    const pipelineSources: string[] = [];
    const transformNodes: DAGNode[] = [];
    const functionNodes: DAGNode[] = [];
    let queryName = '';

    function walkBack(nodeId: string, visited = new Set<string>()) {
      if (visited.has(nodeId)) return;
      visited.add(nodeId);

      const incoming = inEdges.get(nodeId) || [];
      for (const prevId of incoming) {
        const prevNode = nodeMap.get(prevId);
        if (!prevNode) continue;

        if (prevNode.type === 'source') {
          if (!pipelineSources.includes(prevNode.label)) {
            pipelineSources.push(prevNode.label);
          }
        } else if (prevNode.type === 'transform') {
          if (!transformNodes.some(t => t.id === prevNode.id)) {
            transformNodes.push(prevNode);
          }
          walkBack(prevId, visited);
        } else if (prevNode.type === 'function') {
          if (!functionNodes.some(f => f.id === prevNode.id)) {
            functionNodes.push(prevNode);
          }
          walkBack(prevId, visited);
        } else if (prevNode.type === 'query') {
          queryName = prevNode.label;
          // Continue walking back through query to find its sources
          walkBack(prevId, visited);
        }
      }
    }

    walkBack(targetNode.id);

    if (pipelineSources.length === 0) continue;

    const fields: FieldMapping[] = [];
    for (const t of transformNodes) {
      const mappings = t.config.fieldMappings as FieldMapping[] || [];
      for (const m of mappings) {
        fields.push({
          source: m.source,
          target: m.target,
          transform: t.config.expression || m.transform || undefined,
        });
      }
    }

    const functionNames = functionNodes.map(f => f.label);

    // Fingerprint = sources + transform node IDs + function node IDs + query
    const fingerprint = [
      pipelineSources.slice().sort().join(','),
      transformNodes.map(t => t.id).sort().join(','),
      functionNodes.map(f => f.id).sort().join(','),
      queryName,
    ].join('|');

    upstreams.push({ targetNode, pipelineSources, transforms: transformNodes, functionNames, queryName, fields, fingerprint });
  }

  // Second pass: group targets by fingerprint and create merged pipelines
  const groups = new Map<string, TargetUpstream[]>();
  for (const u of upstreams) {
    if (!groups.has(u.fingerprint)) groups.set(u.fingerprint, []);
    groups.get(u.fingerprint)!.push(u);
  }

  for (const group of groups.values()) {
    const first = group[0];
    const targetRefs = group.map(u => {
      const table = u.targetNode.config.table || '';
      return table ? `${u.targetNode.label}.${table}` : u.targetNode.label;
    });
    const targetLabels = group.map(u => u.targetNode.label);

    // Per-target dedup/tracking from the first target that has it
    const dedupTarget = group.find(u => u.targetNode.config.dedupEnabled);

    const pipelineBase = {
      sources: first.pipelineSources,
      fields: first.fields,
      query: first.queryName || undefined,
      functions: first.functionNames.length > 0 ? first.functionNames : undefined,
      uniqueFields: dedupTarget?.targetNode.config.dedupEnabled ? (dedupTarget.targetNode.config.uniqueFields || undefined) : undefined,
      keepLast: dedupTarget?.targetNode.config.dedupEnabled ? (dedupTarget.targetNode.config.keepLast || undefined) : undefined,
      tracking: first.targetNode.config.tracking || undefined,
    };

    pipelines.push({
      name: targetLabels.length === 1
        ? `${first.pipelineSources.join('_')}_to_${targetLabels[0]}`
        : `${first.pipelineSources.join('_')}_to_${targetLabels.join('_')}`,
      targets: targetRefs,
      ...pipelineBase,
    });
  }

  return { name, sources, targets, pipelines, queries, functions: functions.length > 0 ? functions : undefined };
}
