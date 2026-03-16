import { createResource, createSignal, For, Show, onCleanup } from 'solid-js';
import { registryApi, schemaApi, connectionsApi } from '../../api/etl';
import type { SchemaColumn } from '../../api/etl';
import loader from '@monaco-editor/loader';
import Button from '../common/Button';
import CsvInlineEditor from './CsvInlineEditor';

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
    { key: 'filepath', label: 'File Path', placeholder: '/path/to/data.csv' },
    { key: 'delimiter', label: 'Delimiter', placeholder: ',' },
  ],
  json: [
    { key: 'filepath', label: 'File Path', placeholder: '/path/to/data.json' },
  ],
  api: [
    { key: 'url', label: 'URL', placeholder: 'https://api.example.com/data' },
    { key: 'method', label: 'Method', type: 'select', options: ['GET', 'POST', 'PUT'] },
    { key: 'auth_type', label: 'Auth Type', type: 'select', options: ['none', 'Bearer', 'Basic', 'Api-Key'] },
    { key: 'auth_token', label: 'Auth Token / API Key', type: 'password' },
    { key: 'headers', label: 'Headers', type: 'textarea', placeholder: 'Content-Type: application/json; Accept: application/json' },
    { key: 'body', label: 'Request Body', type: 'textarea', placeholder: '{"query": "..."}' },
  ],
  rabbitmq: [
    { key: 'url', label: 'AMQP URL', placeholder: 'amqp://user:pass@localhost:5672/' },
    { key: 'queue', label: 'Queue Name', placeholder: 'my_queue' },
    { key: 'prefetch_count', label: 'Prefetch Count', placeholder: '10' },
    { key: 'timeout', label: 'Collect Timeout', placeholder: '5s' },
  ],
  webhook: [
    { key: 'path', label: 'Endpoint Path', placeholder: '/webhook/orders' },
    { key: 'port', label: 'Listen Port', placeholder: '9090' },
  ],
};

export default function SourceNodeConfig(props: Props) {
  const [types] = createResource(() => registryApi.sourceTypes());
  const [savedConns] = createResource(() => connectionsApi.list());
  const [schemaTables, setSchemaTables] = createSignal<string[]>([]);
  const [schemaColumns, setSchemaColumns] = createSignal<Record<string, SchemaColumn[]>>({});
  const [schemaLoading, setSchemaLoading] = createSignal(false);
  const [schemaError, setSchemaError] = createSignal('');
  const [expandedTable, setExpandedTable] = createSignal<string | null>(null);

  let jsonEditor: any = null;

  async function initJsonEditor(container: HTMLDivElement) {
    if (jsonEditor) return;
    const monaco = await loader.init();
    jsonEditor = monaco.editor.create(container, {
      value: props.config.connection?.inline_data || '[\n  {\n    "field": "value"\n  }\n]',
      language: 'json',
      minimap: { enabled: false },
      lineNumbers: 'on',
      scrollBeyondLastLine: false,
      automaticLayout: true,
      fontSize: 13,
      tabSize: 2,
      theme: window.matchMedia('(prefers-color-scheme: dark)').matches ? 'vs-dark' : 'vs',
    });
    jsonEditor.onDidChangeModelContent(() => {
      setField('inline_data', jsonEditor.getValue());
    });
  }

  onCleanup(() => {
    jsonEditor?.dispose();
    jsonEditor = null;
  });

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
      const result = await schemaApi.introspect(currentType(), conn, 'source');
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

    // Fetch columns if not already loaded
    if (!schemaColumns()[table]) {
      try {
        const conn = props.config.connection || {};
        const result = await schemaApi.describe(currentType(), conn, 'source', table);
        setSchemaColumns((prev) => ({ ...prev, [table]: result.columns }));
      } catch (e: any) {
        setSchemaError(e.message || 'Failed to describe table');
      }
    }
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

      {/* JSON inline editor */}
      <Show when={currentType() === 'json'}>
        <div class="space-y-2">
          <div class="flex gap-1 bg-gray-100 dark:bg-gray-700 rounded p-0.5">
            <button
              class={`flex-1 px-2 py-1 text-xs font-medium rounded transition-colors cursor-pointer ${(props.config.connection?.input_mode || 'file') === 'file' ? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 dark:text-gray-400'}`}
              onClick={() => setField('input_mode', 'file')}
            >
              File Path
            </button>
            <button
              class={`flex-1 px-2 py-1 text-xs font-medium rounded transition-colors cursor-pointer ${props.config.connection?.input_mode === 'inline' ? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 dark:text-gray-400'}`}
              onClick={() => {
                setField('input_mode', 'inline');
                jsonEditor?.dispose();
                jsonEditor = null;
                setTimeout(() => {
                  const el = document.getElementById('json-inline-editor');
                  if (el) initJsonEditor(el as HTMLDivElement);
                }, 50);
              }}
            >
              Inline Editor
            </button>
          </div>
          <Show when={props.config.connection?.input_mode === 'inline'}>
            <div
              id="json-inline-editor"
              class="border border-gray-300 dark:border-gray-600 rounded overflow-hidden"
              style={{ height: '250px' }}
              ref={(el) => setTimeout(() => initJsonEditor(el), 50)}
            />
          </Show>
        </div>
      </Show>

      {/* CSV inline editor */}
      <Show when={currentType() === 'csv'}>
        <div class="space-y-2">
          <div class="flex gap-1 bg-gray-100 dark:bg-gray-700 rounded p-0.5">
            <button
              class={`flex-1 px-2 py-1 text-xs font-medium rounded transition-colors cursor-pointer ${(props.config.connection?.input_mode || 'file') === 'file' ? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 dark:text-gray-400'}`}
              onClick={() => setField('input_mode', 'file')}
            >
              File Path
            </button>
            <button
              class={`flex-1 px-2 py-1 text-xs font-medium rounded transition-colors cursor-pointer ${props.config.connection?.input_mode === 'inline' ? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 dark:text-gray-400'}`}
              onClick={() => setField('input_mode', 'inline')}
            >
              Inline Editor
            </button>
          </div>
          <Show when={props.config.connection?.input_mode === 'inline'}>
            <CsvInlineEditor
              value={props.config.connection?.inline_data || 'column1,column2\nvalue1,value2'}
              onChange={(csv) => setField('inline_data', csv)}
            />
          </Show>
        </div>
      </Show>

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
                    class="w-full text-left text-xs px-1.5 py-1 rounded hover:bg-gray-100 dark:hover:bg-gray-700 font-mono cursor-pointer flex items-center gap-1"
                    onClick={() => toggleTable(table)}
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
    </div>
  );
}
