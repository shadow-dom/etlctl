export interface StorageConfig {
  name: string;
  type: string;
  connection: Record<string, string>;
  connection_ref?: string;
}

export interface FieldMapping {
  source: string;
  target: string;
  transform?: string;
}

export interface TrackingSpec {
  field: string;
  storageType: string;
}

export interface PipelineConfig {
  name: string;
  sources: string[];
  target?: string; // legacy, read-only for backwards compat
  targets: string[];
  fields: FieldMapping[];
  query?: string;
  functions?: string[];
  uniqueFields?: string[];
  keepLast?: boolean;
  tracking?: TrackingSpec;
}

export interface FunctionConfig {
  name: string;
  code?: string;
}

export interface QueryConfig {
  name: string;
  sql: string;
}

export interface DeployConfig {
  schedule?: string;
  namespace?: string;
  image?: string;
}

export interface ETLConfig {
  name: string;
  sources: StorageConfig[];
  targets: StorageConfig[];
  pipelines: PipelineConfig[];
  queries: QueryConfig[];
  functions?: FunctionConfig[];
  deploy?: DeployConfig;
}
