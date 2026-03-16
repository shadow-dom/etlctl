export type NodeType = 'source' | 'transform' | 'target' | 'query' | 'function';

export interface DAGNode {
  id: string;
  type: NodeType;
  label: string;
  x: number;
  y: number;
  config: Record<string, any>;
}

export interface DAGEdge {
  id: string;
  source: string;
  target: string;
}

export interface DAGMeta {
  nodes: DAGNode[];
  edges: DAGEdge[];
}

export interface TestStepResult {
  step: string;
  node: string;
  rowCount: number;
  sample: Record<string, string>[];
  error?: string;
  durationMs: number;
}

export interface TestResponse {
  steps: TestStepResult[];
  success: boolean;
}

export interface ValidateResponse {
  valid: boolean;
  errors: string[];
}

export interface TransformPreviewResponse {
  result: string;
  error?: string;
}
