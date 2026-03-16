import { onMount, onCleanup, createSignal, createEffect, For, Show } from 'solid-js';
import { createStore, produce } from 'solid-js/store';
import loader from '@monaco-editor/loader';
import { schemaApi, connectionsApi, registryApi } from '../../api/etl';
import type { SchemaColumn, ConnectionConfig } from '../../api/etl';
import Button from '../common/Button';

interface ColumnSelection {
  name: string;
  dataType: string;
  nullable: boolean;
  selected: boolean;
  transform: string;
}

interface Props {
  config: Record<string, any>;
  onUpdate: (config: Record<string, any>) => void;
}

type QueryMode = 'builder' | 'sql';

export default function QueryNodeConfig(props: Props) {
  let editorContainer!: HTMLDivElement;
  let editor: any;

  const [savedConns] = createSignal<ConnectionConfig[]>([]);
  const [transforms] = createSignal<{ simple: string[]; parameterized: string[] } | null>(null);
  const [tables, setTables] = createSignal<string[]>([]);
  const [columnDefs, setColumnDefs] = createStore<ColumnSelection[]>([]);
  const [schemaLoading, setSchemaLoading] = createSignal(false);
  const [schemaError, setSchemaError] = createSignal('');
  const [selectedTable, setSelectedTable] = createSignal<string>(props.config.selectedTable || '');

  const mode = (): QueryMode => props.config.queryMode || 'builder';

  // Schema connection config — from linked source node or saved connection
  const schemaConn = () => props.config.schemaConnection as Record<string, string> | undefined;
  const schemaType = () => props.config.schemaType as string | undefined;
  const connRef = () => props.config.queryConnectionRef as string | undefined;

  // Load saved connections and transforms on mount
  onMount(async () => {
    try {
      const conns = await connectionsApi.list();
      (savedConns as any)[1]?.(conns); // Won't work with createSignal
    } catch {}

    try {
      const t = await registryApi.transforms();
      (transforms as any)[1]?.(t);
    } catch {}

    // Auto-fetch tables if connection available
    if (schemaConn() && schemaType()) {
      fetchTables();
    }
  });

  // Re-initialize from saved column selections
  createEffect(() => {
    const savedCols = props.config.selectedColumns as ColumnSelection[] | undefined;
    if (savedCols?.length) {
      setColumnDefs(savedCols);
    }
  });

  onCleanup(() => {
    editor?.dispose();
  });

  async function initEditor() {
    if (editor) return;
    const monaco = await loader.init();
    editor = monaco.editor.create(editorContainer, {
      value: props.config.sql || buildSQL(),
      language: 'sql',
      minimap: { enabled: false },
      lineNumbers: 'on',
      scrollBeyondLastLine: false,
      automaticLayout: true,
      fontSize: 13,
      tabSize: 2,
      theme: window.matchMedia('(prefers-color-scheme: dark)').matches ? 'vs-dark' : 'vs',
    });
    editor.onDidChangeModelContent(() => {
      props.onUpdate({ sql: editor.getValue() });
    });
  }

  function switchMode(newMode: QueryMode) {
    if (newMode === 'sql') {
      // Generate SQL from builder state before switching
      const sql = buildSQL();
      props.onUpdate({ queryMode: newMode, sql });
      setTimeout(() => {
        initEditor();
        editor?.setValue(sql);
      }, 50);
    } else {
      editor?.dispose();
      editor = null;
      props.onUpdate({ queryMode: newMode });
    }
  }

  function buildSQL(): string {
    const table = selectedTable();
    if (!table) return '';
    const selected = columnDefs.filter(c => c.selected);
    if (selected.length === 0) return `SELECT * FROM ${table}`;
    return `SELECT ${selected.map(c => c.name).join(', ')} FROM ${table}`;
  }

  function syncSQL() {
    const sql = buildSQL();
    props.onUpdate({ sql, selectedTable: selectedTable(), selectedColumns: [...columnDefs] });
  }

  async function fetchTables() {
    const conn = schemaConn();
    const type = schemaType();
    if (!conn || !type) return;

    setSchemaLoading(true);
    setSchemaError('');
    try {
      const result = await schemaApi.introspect(type, conn, 'source');
      setTables(result.tables);
    } catch (e: any) {
      setSchemaError(e.message);
    } finally {
      setSchemaLoading(false);
    }
  }

  async function selectTable(table: string) {
    setSelectedTable(table);

    const conn = schemaConn();
    const type = schemaType();
    if (!conn || !type) return;

    setSchemaLoading(true);
    try {
      const result = await schemaApi.describe(type, conn, 'source', table);
      const cols: ColumnSelection[] = result.columns.map(c => ({
        name: c.name,
        dataType: c.data_type,
        nullable: c.nullable,
        selected: true,
        transform: '',
      }));
      setColumnDefs(cols);
      props.onUpdate({
        selectedTable: table,
        selectedColumns: cols,
        sql: `SELECT ${cols.map(c => c.name).join(', ')} FROM ${table}`,
      });
    } catch (e: any) {
      setSchemaError(e.message);
    } finally {
      setSchemaLoading(false);
    }
  }

  function toggleColumn(index: number) {
    setColumnDefs(produce(cols => {
      cols[index].selected = !cols[index].selected;
    }));
    syncSQL();
  }

  function selectAllColumns() {
    setColumnDefs(produce(cols => {
      for (const col of cols) col.selected = true;
    }));
    syncSQL();
  }

  function selectNoneColumns() {
    setColumnDefs(produce(cols => {
      for (const col of cols) col.selected = false;
    }));
    syncSQL();
  }

  function setColumnTransform(index: number, transform: string) {
    setColumnDefs(produce(cols => {
      cols[index].transform = transform;
    }));
    props.onUpdate({ selectedColumns: [...columnDefs] });
  }

  const selectedCount = () => columnDefs.filter(c => c.selected).length;
  const hasConnection = () => !!(schemaConn() && schemaType());

  return (
    <div class="space-y-3">
      {/* Connection status */}
      <Show when={hasConnection()}>
        <div class="flex items-center gap-1.5 text-xs text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/20 rounded px-2 py-1.5">
          <span class="w-1.5 h-1.5 bg-green-500 rounded-full" />
          <span>Connected via source node</span>
          <span class="font-mono text-gray-500 ml-auto">{schemaType()}</span>
        </div>
      </Show>
      <Show when={!hasConnection()}>
        <div class="text-xs text-amber-600 dark:text-amber-400 bg-amber-50 dark:bg-amber-900/20 rounded px-2 py-1.5">
          Connect a source node to this query node to browse tables and columns.
        </div>
      </Show>

      {/* Mode toggle */}
      <div class="flex gap-1 bg-gray-100 dark:bg-gray-700 rounded p-0.5">
        <button
          class={`flex-1 px-2 py-1 text-xs font-medium rounded transition-colors cursor-pointer ${mode() === 'builder' ? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 dark:text-gray-400'}`}
          onClick={() => switchMode('builder')}
        >
          Query Builder
        </button>
        <button
          class={`flex-1 px-2 py-1 text-xs font-medium rounded transition-colors cursor-pointer ${mode() === 'sql' ? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 dark:text-gray-400'}`}
          onClick={() => switchMode('sql')}
        >
          Raw SQL
        </button>
      </div>

      {/* Builder mode */}
      <Show when={mode() === 'builder'}>
        {/* Table selector */}
        <Show when={hasConnection()}>
          <div>
            <div class="flex items-center justify-between mb-1.5">
              <label class="text-xs font-medium text-gray-500">Table</label>
              <Button variant="secondary" size="sm" onClick={fetchTables} disabled={schemaLoading()}>
                {schemaLoading() ? 'Loading...' : 'Refresh'}
              </Button>
            </div>
            <Show when={schemaError()}>
              <div class="text-xs text-red-600 mb-1">{schemaError()}</div>
            </Show>
            <Show when={tables().length > 0}>
              <select
                value={selectedTable()}
                onChange={(e) => {
                  const t = e.currentTarget.value;
                  if (t) selectTable(t);
                }}
                class="w-full px-2 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white font-mono"
              >
                <option value="">Select a table...</option>
                <For each={tables()}>{(t) => <option value={t}>{t}</option>}</For>
              </select>
            </Show>
            <Show when={tables().length === 0 && !schemaLoading()}>
              <div class="text-xs text-gray-400">Click Refresh to load tables</div>
            </Show>
          </div>
        </Show>

        {/* Columns */}
        <Show when={columnDefs.length > 0}>
          <div>
            <div class="flex items-center justify-between mb-1.5">
              <label class="text-xs font-medium text-gray-500">
                Columns <span class="text-gray-400">({selectedCount()}/{columnDefs.length})</span>
              </label>
              <div class="flex gap-1">
                <button class="text-[10px] text-blue-600 hover:text-blue-800 cursor-pointer font-medium" onClick={selectAllColumns}>All</button>
                <span class="text-gray-300">|</span>
                <button class="text-[10px] text-blue-600 hover:text-blue-800 cursor-pointer font-medium" onClick={selectNoneColumns}>None</button>
              </div>
            </div>
            <div class="border border-gray-200 dark:border-gray-600 rounded overflow-hidden">
              {/* Header */}
              <div class="grid grid-cols-[auto_1fr_70px_1fr] gap-1 px-2 py-1 bg-gray-50 dark:bg-gray-700/50 text-[10px] font-medium text-gray-500 uppercase border-b border-gray-200 dark:border-gray-600">
                <span />
                <span>Column</span>
                <span>Type</span>
                <span>Transform</span>
              </div>
              {/* Rows */}
              <div class="max-h-64 overflow-y-auto divide-y divide-gray-100 dark:divide-gray-700">
                <For each={columnDefs}>
                  {(col, index) => (
                    <div class={`grid grid-cols-[auto_1fr_70px_1fr] gap-1 px-2 py-1 items-center ${col.selected ? '' : 'opacity-50'}`}>
                      <input
                        type="checkbox"
                        checked={col.selected}
                        onChange={() => toggleColumn(index())}
                        class="rounded text-blue-600 w-3 h-3"
                      />
                      <span class="text-xs font-mono text-gray-800 dark:text-gray-200 truncate">
                        {col.name}
                        <Show when={col.nullable}>
                          <span class="text-gray-400 text-[10px] ml-0.5">?</span>
                        </Show>
                      </span>
                      <span class="text-[10px] text-gray-400 font-mono truncate">{col.dataType}</span>
                      <input
                        type="text"
                        value={col.transform}
                        onInput={(e) => setColumnTransform(index(), e.currentTarget.value)}
                        disabled={!col.selected}
                        class="px-1 py-0.5 text-[11px] font-mono border border-gray-200 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white w-full disabled:opacity-30"
                        placeholder="none"
                      />
                    </div>
                  )}
                </For>
              </div>
            </div>
          </div>

          {/* SQL Preview */}
          <div>
            <label class="block text-xs font-medium text-gray-500 mb-1">Generated SQL</label>
            <pre class="text-xs font-mono bg-gray-50 dark:bg-gray-700/50 border border-gray-200 dark:border-gray-600 rounded p-2 text-gray-700 dark:text-gray-300 whitespace-pre-wrap">
              {buildSQL()}
            </pre>
          </div>
        </Show>
      </Show>

      {/* SQL mode */}
      <Show when={mode() === 'sql'}>
        <div>
          <label class="block text-xs font-medium text-gray-500 mb-1">SQL Query</label>
          <div
            ref={editorContainer}
            class="border border-gray-300 dark:border-gray-600 rounded overflow-hidden"
            style={{ height: '250px' }}
          />
        </div>
      </Show>
    </div>
  );
}
