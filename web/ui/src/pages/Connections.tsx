import { createResource, createSignal, For, Show } from 'solid-js';
import { connectionsApi } from '../api/etl';
import type { ConnectionConfig } from '../api/etl';
import Header from '../components/layout/Header';
import Button from '../components/common/Button';

export default function Connections() {
  const [connections, { refetch }] = createResource(() => connectionsApi.list());
  const [editing, setEditing] = createSignal<ConnectionConfig | null>(null);
  const [isNew, setIsNew] = createSignal(false);

  function startNew() {
    setEditing({ name: '', type: 'sqlite3', connection: {} });
    setIsNew(true);
  }

  function startEdit(conn: ConnectionConfig) {
    setEditing({ ...conn, connection: { ...conn.connection } });
    setIsNew(false);
  }

  async function save() {
    const conn = editing();
    if (!conn || !conn.name) return;
    try {
      if (isNew()) {
        await connectionsApi.save(conn);
      } else {
        await connectionsApi.update(conn.name, conn);
      }
      setEditing(null);
      refetch();
    } catch (e: any) {
      alert(e.message);
    }
  }

  async function remove(name: string) {
    if (!confirm(`Delete connection "${name}"?`)) return;
    await connectionsApi.delete(name);
    refetch();
  }

  const dbTypes = ['sqlite3', 'postgres', 'sqlserver', 'csv', 'json', 'api'];

  const connectionFieldDefs: Record<string, { key: string; label: string; type?: string; placeholder?: string }[]> = {
    sqlite3: [{ key: 'filepath', label: 'File Path', placeholder: '/path/to/db.db' }],
    postgres: [
      { key: 'host', label: 'Host', placeholder: 'localhost' },
      { key: 'port', label: 'Port', placeholder: '5432' },
      { key: 'user', label: 'User' },
      { key: 'password', label: 'Password', type: 'password' },
      { key: 'database', label: 'Database' },
      { key: 'sslmode', label: 'SSL Mode', placeholder: 'disable' },
    ],
    sqlserver: [
      { key: 'host', label: 'Host', placeholder: 'localhost' },
      { key: 'port', label: 'Port', placeholder: '1433' },
      { key: 'user', label: 'User' },
      { key: 'password', label: 'Password', type: 'password' },
      { key: 'database', label: 'Database' },
    ],
  };

  return (
    <div class="flex flex-col h-full">
      <Header
        title="Connections"
        actions={<Button variant="primary" size="sm" onClick={startNew}>+ New Connection</Button>}
      />
      <div class="flex-1 overflow-auto p-6">
        <Show when={!connections.loading} fallback={<p class="text-gray-500">Loading...</p>}>
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            <For each={connections() || []}>
              {(conn) => (
                <div class="border border-gray-200 dark:border-gray-700 rounded-lg p-4 bg-white dark:bg-gray-800">
                  <div class="flex items-center justify-between mb-2">
                    <h3 class="font-medium text-gray-900 dark:text-white">{conn.name}</h3>
                    <span class="text-xs px-2 py-0.5 bg-gray-100 dark:bg-gray-700 rounded text-gray-600 dark:text-gray-400">{conn.type}</span>
                  </div>
                  <div class="text-xs text-gray-500 space-y-0.5 mb-3">
                    {Object.entries(conn.connection || {}).filter(([k]) => k !== 'password').map(([k, v]) => (
                      <div><span class="font-medium">{k}:</span> {v}</div>
                    ))}
                  </div>
                  <div class="flex gap-1">
                    <Button variant="ghost" size="sm" onClick={() => startEdit(conn)}>Edit</Button>
                    <Button variant="danger" size="sm" onClick={() => remove(conn.name)}>Delete</Button>
                  </div>
                </div>
              )}
            </For>
          </div>
        </Show>

        {/* Edit/Create modal */}
        <Show when={editing()}>
          <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
            <div class="bg-white dark:bg-gray-800 rounded-lg p-6 w-[480px] max-h-[80vh] overflow-auto">
              <h2 class="text-lg font-semibold mb-4 text-gray-900 dark:text-white">
                {isNew() ? 'New Connection' : 'Edit Connection'}
              </h2>
              <div class="space-y-3">
                <div>
                  <label class="block text-xs font-medium text-gray-500 mb-1">Name</label>
                  <input
                    type="text"
                    value={editing()!.name}
                    onInput={(e) => setEditing({ ...editing()!, name: e.currentTarget.value })}
                    disabled={!isNew()}
                    class="w-full px-2 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                    placeholder="my-database"
                  />
                </div>
                <div>
                  <label class="block text-xs font-medium text-gray-500 mb-1">Type</label>
                  <select
                    value={editing()!.type}
                    onChange={(e) => setEditing({ ...editing()!, type: e.currentTarget.value, connection: {} })}
                    class="w-full px-2 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                  >
                    <For each={dbTypes}>{(t) => <option value={t}>{t}</option>}</For>
                  </select>
                </div>
                <For each={connectionFieldDefs[editing()!.type] || []}>
                  {(field) => (
                    <div>
                      <label class="block text-xs font-medium text-gray-500 mb-1">{field.label}</label>
                      <input
                        type={field.type || 'text'}
                        value={editing()!.connection[field.key] || ''}
                        onInput={(e) => {
                          const conn = { ...editing()!.connection, [field.key]: e.currentTarget.value };
                          setEditing({ ...editing()!, connection: conn });
                        }}
                        class="w-full px-2 py-1.5 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                        placeholder={field.placeholder || ''}
                      />
                    </div>
                  )}
                </For>
              </div>
              <div class="flex gap-2 mt-4 justify-end">
                <Button variant="ghost" size="sm" onClick={() => setEditing(null)}>Cancel</Button>
                <Button variant="primary" size="sm" onClick={save}>Save</Button>
              </div>
            </div>
          </div>
        </Show>
      </div>
    </div>
  );
}
