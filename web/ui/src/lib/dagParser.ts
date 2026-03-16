import type { ETLConfig } from '../types/etl';
import type { DAGNode, DAGEdge } from '../types/dag';

/**
 * Converts a flat ETL config into a DAG layout for the visual editor.
 * Places sources on the left, targets on the right, transforms/functions in the middle.
 */
export function parseToDag(etl: ETLConfig): { nodes: DAGNode[]; edges: DAGEdge[] } {
  const nodes: DAGNode[] = [];
  const edges: DAGEdge[] = [];
  let nodeId = 1;

  const NODE_WIDTH = 200;
  const NODE_GAP = 40;
  const COL_GAP = 300;

  const etlSources = etl.sources || [];
  const etlTargets = etl.targets || [];
  const etlQueries = etl.queries || [];
  const etlPipelines = etl.pipelines || [];
  const etlFunctions = etl.functions || [];

  // Source column (x=50)
  const sourceIds = new Map<string, string>();
  etlSources.forEach((src, i) => {
    const id = `node_${nodeId++}`;
    sourceIds.set(src.name, id);
    nodes.push({
      id,
      type: 'source',
      label: src.name,
      x: 50,
      y: 50 + i * (NODE_WIDTH / 2 + NODE_GAP),
      config: {
        storageType: src.type,
        connection: { ...src.connection },
      },
    });
  });

  // Query nodes (x=200)
  const queryIds = new Map<string, string>();
  etlQueries.forEach((q, i) => {
    const id = `node_${nodeId++}`;
    queryIds.set(q.name, id);
    nodes.push({
      id,
      type: 'query',
      label: q.name,
      x: 50 + COL_GAP * 0.5,
      y: 50 + i * (NODE_WIDTH / 2 + NODE_GAP),
      config: { sql: q.sql },
    });
  });

  // Function nodes (placed between transforms and targets)
  const functionIds = new Map<string, string>();
  etlFunctions.forEach((fn, i) => {
    const id = `node_${nodeId++}`;
    functionIds.set(fn.name, id);
    nodes.push({
      id,
      type: 'function',
      label: fn.name,
      x: 50 + COL_GAP * 1.5,
      y: 50 + i * (NODE_WIDTH / 2 + NODE_GAP),
      config: {
        goCode: fn.code || '',
      },
    });
  });

  // Target column (x=far right)
  const targetIds = new Map<string, string>();
  etlTargets.forEach((tgt, i) => {
    const id = `node_${nodeId++}`;
    targetIds.set(tgt.name, id);
    nodes.push({
      id,
      type: 'target',
      label: tgt.name,
      x: 50 + COL_GAP * 2.5,
      y: 50 + i * (NODE_WIDTH / 2 + NODE_GAP),
      config: {
        storageType: tgt.type,
        connection: { ...tgt.connection },
      },
    });
  });

  // Helper: collect all target names for a pipeline (supports singular + plural)
  const getAllTargets = (p: { target?: string; targets?: string[] }): string[] => {
    if (p.targets && p.targets.length > 0) {
      return p.targets.map(t => t.split('.')[0]);
    }
    if (p.target) {
      return [p.target.split('.')[0]];
    }
    return [];
  };

  // Create transform nodes and edges from pipelines
  etlPipelines.forEach((p, pi) => {
    const hasTransforms = p.fields.some(f => f.transform);
    const hasFunctions = (p.functions || []).length > 0;

    if (hasTransforms) {
      // Create a transform node
      const transformId = `node_${nodeId++}`;
      nodes.push({
        id: transformId,
        type: 'transform',
        label: p.name || `transform_${pi}`,
        x: 50 + COL_GAP,
        y: 50 + pi * (NODE_WIDTH / 2 + NODE_GAP),
        config: {
          expression: p.fields.find(f => f.transform)?.transform || '',
          fieldMappings: p.fields.map(f => ({
            source: f.source,
            target: f.target,
            transform: f.transform || '',
          })),
        },
      });

      // Edges: sources -> transform
      for (const srcName of p.sources) {
        const srcId = sourceIds.get(srcName);
        if (srcId) {
          edges.push({ id: `edge_${srcId}_${transformId}`, source: srcId, target: transformId });
        }
      }

      // Query -> transform
      if (p.query) {
        const qId = queryIds.get(p.query);
        if (qId) {
          edges.push({ id: `edge_${qId}_${transformId}`, source: qId, target: transformId });
        }
      }

      if (hasFunctions) {
        // transform -> first function, chain functions, last function -> target
        let prevId = transformId;
        for (const fnName of p.functions!) {
          const fnId = functionIds.get(fnName);
          if (fnId) {
            edges.push({ id: `edge_${prevId}_${fnId}`, source: prevId, target: fnId });
            prevId = fnId;
          }
        }
        for (const targetName of getAllTargets(p)) {
          const tgtId = targetIds.get(targetName);
          if (tgtId) {
            edges.push({ id: `edge_${prevId}_${tgtId}`, source: prevId, target: tgtId });
          }
        }
      } else {
        // transform -> target(s)
        for (const targetName of getAllTargets(p)) {
          const tgtId = targetIds.get(targetName);
          if (tgtId) {
            edges.push({ id: `edge_${transformId}_${tgtId}`, source: transformId, target: tgtId });
          }
        }
      }
    } else if (hasFunctions) {
      // No transforms, but has functions: sources -> first function -> ... -> target
      let prevIds: string[] = [];

      for (const srcName of p.sources) {
        const srcId = sourceIds.get(srcName);
        if (srcId) prevIds.push(srcId);
      }
      if (p.query) {
        const qId = queryIds.get(p.query);
        if (qId) prevIds.push(qId);
      }

      for (const fnName of p.functions!) {
        const fnId = functionIds.get(fnName);
        if (fnId) {
          for (const prevId of prevIds) {
            edges.push({ id: `edge_${prevId}_${fnId}`, source: prevId, target: fnId });
          }
          prevIds = [fnId];
        }
      }

      for (const targetName of getAllTargets(p)) {
        const tgtId = targetIds.get(targetName);
        if (tgtId) {
          for (const prevId of prevIds) {
            edges.push({ id: `edge_${prevId}_${tgtId}`, source: prevId, target: tgtId });
          }
        }
      }
    } else {
      // Direct source -> target(s) edges
      for (const targetName of getAllTargets(p)) {
        const tgtId = targetIds.get(targetName);

        for (const srcName of p.sources) {
          const srcId = sourceIds.get(srcName);
          if (srcId && tgtId) {
            edges.push({ id: `edge_${srcId}_${tgtId}`, source: srcId, target: tgtId });
          }
        }

        // Query -> target
        if (p.query) {
          const qId = queryIds.get(p.query);
          if (qId && tgtId) {
            edges.push({ id: `edge_${qId}_${tgtId}`, source: qId, target: tgtId });
          }
        }
      }
    }
  });

  return { nodes, edges };
}
