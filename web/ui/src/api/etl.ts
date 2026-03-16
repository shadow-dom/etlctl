import type { ETLConfig } from '../types/etl';
import type { DAGMeta, TestResponse, ValidateResponse, TransformPreviewResponse } from '../types/dag';
import { get, post, put, del } from './client';

export const etlApi = {
  list: () => get<ETLConfig[]>('/etls'),
  get: (name: string) => get<ETLConfig>(`/etls/${name}`),
  create: (etl: ETLConfig) => post<ETLConfig>('/etls', etl),
  update: (name: string, etl: ETLConfig) => put<ETLConfig>(`/etls/${name}`, etl),
  delete: (name: string) => del<void>(`/etls/${name}`),
  validate: (name: string) => post<ValidateResponse>(`/etls/${name}/validate`),
  run: (name: string) => post<TestResponse>(`/etls/${name}/run`),
  test: (name: string, dryRun = true, resetState = true) => post<TestResponse>(`/etls/${name}/test?dryRun=${dryRun}&resetState=${resetState}`),
  getDag: (name: string) => get<DAGMeta>(`/etls/${name}/dag`),
  saveDag: (name: string, dag: DAGMeta) => put<void>(`/etls/${name}/dag`, dag),
  getRawYaml: (name: string): Promise<string> =>
    fetch(`/api/v1/etls/${name}/yaml`).then(r => {
      if (!r.ok) throw new Error('Failed to fetch YAML');
      return r.text();
    }),
  updateRawYaml: (name: string, yaml: string): Promise<void> =>
    fetch(`/api/v1/etls/${name}/yaml`, {
      method: 'PUT',
      headers: { 'Content-Type': 'text/plain' },
      body: yaml,
    }).then(r => {
      if (!r.ok) throw new Error('Failed to save YAML');
    }),
};

export interface SchemaColumn {
  name: string;
  data_type: string;
  nullable: boolean;
}

export interface SchemaDescribeResponse {
  table: string;
  columns: SchemaColumn[];
}

export interface SchemaAutoMapResponse {
  fields: { source: string; target: string }[];
  unmatched_source: string[];
  unmatched_target: string[];
}

export const schemaApi = {
  introspect: (type: string, connection: Record<string, string>, role: string) =>
    post<{ tables: string[] }>('/schema/introspect', { type, connection, role }),
  describe: (type: string, connection: Record<string, string>, role: string, table: string) =>
    post<SchemaDescribeResponse>('/schema/describe', { type, connection, role, table }),
  autoMap: (
    source: { type: string; connection: Record<string, string>; table: string },
    target: { type: string; connection: Record<string, string>; table: string },
  ) => post<SchemaAutoMapResponse>('/schema/auto-map', { source, target }),
};

export interface ConnectionConfig {
  name: string;
  type: string;
  connection: Record<string, string>;
}

export const connectionsApi = {
  list: () => get<ConnectionConfig[]>('/connections'),
  get: (name: string) => get<ConnectionConfig>(`/connections/${name}`),
  save: (conn: ConnectionConfig) => post<ConnectionConfig>('/connections', conn),
  update: (name: string, conn: ConnectionConfig) => put<ConnectionConfig>(`/connections/${name}`, conn),
  delete: (name: string) => del<void>(`/connections/${name}`),
};

export interface ETLVersion {
  version: number;
  label: string;
  createdAt: string;
  isDraft: boolean;
}

export const versionsApi = {
  list: (etlName: string) => get<ETLVersion[]>(`/etls/${etlName}/versions`),
  save: (etlName: string, label: string, isDraft: boolean) =>
    post<ETLVersion>(`/etls/${etlName}/versions`, { label, isDraft }),
  restore: (etlName: string, version: number) =>
    post<void>(`/etls/${etlName}/versions/${version}/restore`),
};

export const registryApi = {
  sourceTypes: () => get<{ types: string[] }>('/registry/sources'),
  targetTypes: () => get<{ types: string[] }>('/registry/targets'),
  transforms: () => get<{ simple: string[]; parameterized: string[] }>('/registry/transforms'),
  functionTypes: () => get<{ types: string[] }>('/registry/functions'),
  previewTransform: (expression: string, value: string) =>
    post<TransformPreviewResponse>('/transforms/preview', { expression, value }),
};
