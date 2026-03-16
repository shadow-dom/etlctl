import { createResource, createSignal, For, Show } from 'solid-js';
import { registryApi, schemaApi, connectionsApi } from '../../api/etl';
import type { SchemaColumn } from '../../api/etl';
import Button from '../common/Button';

interface Props {
  config: Record<string, any>;
  onUpdate: (config: Record<string, any>) => void;
}

interface FieldDef {
  key: string;
  label: string;
  type?: 'text' | 'password' | 'select' | 'textarea';
  options?: string[];
  placeholder?: string;
}

const connectionFields: Record<string, FieldDef[]> = {
  sqlite3: [
    { key: 'filepath', label: 'File Path', placeholder: '/path/to/database.db' },
  ],
  postgres: [
    { key: 'host', label: 'Host', placeholder: 'localhost' },
    { key: 'port', label: 'Port', placeholder: '5432' },
    { key: 'user', label: 'User', placeholder: 'postgres' },
    { key: 'password', label: 'Password', type: 'password' },
    { key: 'database', label: 'Database', placeholder: 'mydb' },
    { key: 'sslmode', label: 'SSL Mode', type: 'select', options: ['disable', 'require', 'verify-ca', 'verify-full'] },
  ],
  sqlserver: [
    { key: 'host', label: 'Host', placeholder: 'localhost' },
    { key: 'port', label: 'Port', placeholder: '1433' },
    { key: 'user', label: 'User' },
    { key: 'password', label: 'Password', type: 'password' },
    { key: 'database', label: 'Database' },
  ],
  csv: [
    { key: 'filepath', label: 'File Path', placeholder: '/path/to/output.csv' },
    { key: 'delimiter', label: 'Delimiter', placeholder: ',' },
  ],
  json: [
    { key: 'filepath', label: 'File Path', placeholder: '/path/to/output.json' },
  ],
  api: [
    { key: 'url', label: 'URL', placeholder: 'https://api.example.com/data' },
    { key: 'method', label: 'Method', type: 'select', options: ['POST', 'PUT', 'PATCH'] },
    { key: 'auth_type', label: 'Auth Type', type: 'select', options: ['none', 'Bearer', 'Basic', 'Api-Key'] },
    { key: 'auth_token', label: 'Auth Token / API Key', type: 'password' },
    { key: 'headers', label: 'Headers', type: 'textarea', placeholder: 'Content-Type: application/json; Accept: application/json' },
  ],
};

const dbTypes = ['sqlite3', 'postgres', 'sqlserver'];

export default function TargetNodeConfig(props: Props) {
  const [types] = createResource(() => registryApi.targetTypes());
  const [savedConns] = createResource(() => connectionsApi.list());
  const [schemaTables, setSchemaTables] = createSignal<string[]>([]);
  const [schemaColumns, setSchemaColumns] = createSignal<Record<string, SchemaColumn[]>>({});
  const [schemaLoading, setSchemaLoading] = createSignal(false);
  const [schemaError, setSchemaError] = createSignal('');
  const [expandedTable, setExpandedTable] = createSignal<string | null>(null);

  const currentType = () => props.config.storageType || 'sqlite3';
  const fields = () => connectionFields[currentType()] || connectionFields.sqlite3;

  function setType(type: string) {
    props.onUpdate({ storageType: type, connection: {} });
    setSchemaTables([]);
    setSchemaColumns({});
  }

  function setField(key: string, value: string) {
    const conn = { ...(props.config.connection || {}), [key]: value };
    props.onUpdate({ connection: conn });
  }

  async function fetchSchema() {
    setSchemaLoading(true);
    setSchemaError('');
    setSchemaTables([]);
    setSchemaColumns({});

    try {
      const conn = props.config.connection || {};
      const result = await schemaApi.introspect(currentType(), conn, 'target');
      setSchemaTables(result.tables);
    } catch (e: any) {
      setSchemaError(e.message || 'Failed to fetch schema');
    } finally {
      setSchemaLoading(false);
    }
  }

  async function toggleTable(table: string) {
    if (expandedTable() === table) {
      setExpandedTable(null);
      return;
    }
    setExpandedTable(table);

    if (!schemaColumns()[table]) {
      try {
        const conn = props.config.connection || {};
        const result = await schemaApi.describe(currentType(), conn, 'target', table);
        setSchemaColumns((prev) => ({ ...prev, [table]: result.columns }));
      } catch (e: any) {
        setSchemaError(e.message || 'Failed to describe table');
      }
    }
  }

  function selectTable(table: string) {
    props.onUpdate({ table });
  }

  return (
    <div class="space-y-3">
      {/* Saved connection selector */}
      <Show when={savedConns()?.length}>
        <div>
          <label class="block text-xs font-medium text-gray-500 mb-1">Saved Connection</label>
          <select
            value={props.config.connectionRef || ''}
            onChange={(e) => {
              const name = e.currentTarget.value;
              if (!name) {
                props.onUpdate({ connectionRef: undefined });
                return;
              }
              const conn = savedConns()!.find(c => c.name === name);
              if (conn) {
                props.onUpdate({ storageType: conn.type, connection: { ...conn.connection }, connectionRef: name });
              }
            }}
            class="w-full px-2 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
          >
            <option value="">Manual connection...</option>
            <For each={savedConns()!}>{(c) => <option value={c.name}>{c.name} ({c.type})</option>}</For>
          </select>
        </div>
        <Show when={props.config.connectionRef}>
          <div class="flex items-center gap-1.5 text-xs text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/20 rounded px-2 py-1">
            <span>Linked to</span>
            <span class="font-medium">{props.config.connectionRef}</span>
            <button
              class="ml-auto text-gray-400 hover:text-red-500 cursor-pointer"
              onClick={() => props.onUpdate({ connectionRef: undefined })}
              title="Unlink connection"
            >
              &times;
            </button>
          </div>
        </Show>
      </Show>

      <div>
        <label class="block text-xs font-medium text-gray-500 mb-1">Type</label>
        <select
          value={currentType()}
          onChange={(e) => setType(e.currentTarget.value)}
          class="w-full px-2 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
        >
          <For each={types()?.types || [currentType()]}>
            {(t) => <option value={t}>{t}</option>}
          </For>
        </select>
      </div>

      {/* Table name for DB targets */}
      <Show when={dbTypes.includes(currentType())}>
        <div>
          <label class="block text-xs text-gray-500 mb-0.5">Table</label>
          <input
            type="text"
            value={props.config.table || ''}
            onInput={(e) => props.onUpdate({ table: e.currentTarget.value })}
            class="w-full px-2 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
            placeholder="table_name"
          />
        </div>
      </Show>

      <div class="text-xs font-medium text-gray-500 mt-2">Connection</div>
      <For each={fields()}>
        {(field) => (
          <div>
            <label class="block text-xs text-gray-500 mb-0.5">{field.label}</label>
            <Show when={field.type === 'select'}>
              <select
                value={props.config.connection?.[field.key] || field.options?.[0] || ''}
                onChange={(e) => setField(field.key, e.currentTarget.value)}
                class="w-full px-2 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              >
                <For each={field.options || []}>{(opt) => <option value={opt}>{opt}</option>}</For>
              </select>
            </Show>
            <Show when={field.type === 'textarea'}>
              <textarea
                value={props.config.connection?.[field.key] || ''}
                onInput={(e) => setField(field.key, e.currentTarget.value)}
                class="w-full px-2 py-1 text-sm font-mono border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white resize-none"
                rows={3}
                placeholder={field.placeholder}
              />
            </Show>
            <Show when={!field.type || field.type === 'text' || field.type === 'password'}>
              <input
                type={field.type === 'password' ? 'password' : 'text'}
                value={props.config.connection?.[field.key] || ''}
                onInput={(e) => setField(field.key, e.currentTarget.value)}
                class="w-full px-2 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                placeholder={field.placeholder || field.key}
              />
            </Show>
          </div>
        )}
      </For>

      {/* Fetch Schema button */}
      <div class="pt-1">
        <Button
          variant="secondary"
          size="sm"
          onClick={fetchSchema}
          disabled={schemaLoading()}
        >
          {schemaLoading() ? 'Fetching...' : 'Fetch Schema'}
        </Button>
      </div>

      <Show when={schemaError()}>
        <div class="text-xs text-red-600">{schemaError()}</div>
      </Show>

      {/* Schema browser */}
      <Show when={schemaTables().length > 0}>
        <div class="border border-gray-200 dark:border-gray-600 rounded p-2">
          <label class="block text-xs font-medium text-gray-500 mb-1">Tables</label>
          <div class="space-y-1 max-h-48 overflow-y-auto">
            <For each={schemaTables()}>
              {(table) => (
                <div>
                  <button
                    class={`w-full text-left text-xs px-1.5 py-1 rounded hover:bg-gray-100 dark:hover:bg-gray-700 font-mono cursor-pointer flex items-center gap-1 ${props.config.table === table ? 'bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300' : ''}`}
                    onClick={() => toggleTable(table)}
                    onDblClick={() => selectTable(table)}
                    title="Double-click to set as target table"
                  >
                    <span class="text-gray-400">{expandedTable() === table ? '\u25BC' : '\u25B6'}</span>
                    {table || '(flat)'}
                  </button>
                  <Show when={expandedTable() === table && schemaColumns()[table]}>
                    <div class="ml-4 mt-0.5 space-y-0.5">
                      <For each={schemaColumns()[table]}>
                        {(col) => (
                          <div class="text-xs font-mono text-gray-600 dark:text-gray-400 flex gap-2">
                            <span>{col.name}</span>
                            <span class="text-gray-400">{col.data_type}</span>
                            <Show when={col.nullable}>
                              <span class="text-gray-400 italic">null</span>
                            </Show>
                          </div>
                        )}
                      </For>
                    </div>
                  </Show>
                </div>
              )}
            </For>
          </div>
        </div>
      </Show>

      {/* Deduplication settings */}
      <div class="border-t border-gray-200 dark:border-gray-700 pt-3 mt-3">
        <label class="flex items-center gap-2 text-xs font-medium text-gray-500 mb-2">
          <input
            type="checkbox"
            checked={props.config.dedupEnabled || false}
            onChange={(e) => props.onUpdate({ dedupEnabled: e.currentTarget.checked })}
            class="rounded"
          />
          Enable Deduplication
        </label>
        <Show when={props.config.dedupEnabled}>
          <div class="space-y-2 ml-5">
            <div>
              <label class="block text-xs text-gray-500 mb-0.5">Unique Key Fields (comma-separated)</label>
              <input
                type="text"
                value={(props.config.uniqueFields || []).join(', ')}
                onInput={(e) => {
                  const fields = e.currentTarget.value.split(',').map(f => f.trim()).filter(Boolean);
                  props.onUpdate({ uniqueFields: fields });
                }}
                class="w-full px-2 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                placeholder="id, email"
              />
            </div>
            <div>
              <label class="block text-xs text-gray-500 mb-0.5">Keep</label>
              <select
                value={props.config.keepLast ? 'last' : 'first'}
                onChange={(e) => props.onUpdate({ keepLast: e.currentTarget.value === 'last' })}
                class="w-full px-2 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
              >
                <option value="first">First Record</option>
                <option value="last">Last Record</option>
              </select>
            </div>
          </div>
        </Show>
      </div>
    </div>
  );
}
