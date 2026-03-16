import { createResource, createSignal, For, Show } from 'solid-js';
import { A } from '@solidjs/router';
import { etlApi } from '../../api/etl';
import Button from '../common/Button';

export default function Sidebar() {
  const [etls, { refetch }] = createResource(() => etlApi.list());
  const [collapsed, setCollapsed] = createSignal(false);

  return (
    <aside
      class={`${collapsed() ? 'w-12' : 'w-64'} bg-gray-50 dark:bg-gray-900 border-r border-gray-200 dark:border-gray-700 flex flex-col h-full transition-[width] duration-200 flex-shrink-0`}
    >
      {/* Header */}
      <div class="border-b border-gray-200 dark:border-gray-700 flex items-center justify-between px-3 py-3">
        <Show when={!collapsed()}>
          <A href="/" class="text-lg font-bold text-gray-900 dark:text-white no-underline">
            etlctl
          </A>
        </Show>
        <button
          class="p-1 rounded hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-500 cursor-pointer text-xs"
          onClick={() => setCollapsed(!collapsed())}
          title={collapsed() ? 'Expand sidebar' : 'Collapse sidebar'}
        >
          {collapsed() ? '\u25B6' : '\u25C0'}
        </button>
      </div>

      {/* Collapsed view */}
      <Show when={collapsed()}>
        <div class="flex flex-col items-center gap-2 py-3">
          <A href="/new" title="New ETL">
            <button class="w-8 h-8 rounded bg-blue-600 text-white text-lg font-bold cursor-pointer hover:bg-blue-700">+</button>
          </A>
        </div>
        <nav class="flex-1 overflow-y-auto">
          <Show when={!etls.loading}>
            <For each={etls()}>
              {(etl) => (
                <A
                  href={`/etl/${etl.name}`}
                  class="block px-2 py-2 text-center no-underline"
                  activeClass="bg-blue-50 dark:bg-blue-900/30"
                  title={etl.name}
                >
                  <span class="text-xs font-bold text-gray-600 dark:text-gray-400 uppercase">
                    {etl.name.substring(0, 2)}
                  </span>
                </A>
              )}
            </For>
          </Show>
        </nav>
        <div class="p-2 border-t border-gray-200 dark:border-gray-700">
          <button
            class="w-8 h-8 mx-auto block rounded text-gray-500 hover:bg-gray-200 dark:hover:bg-gray-700 cursor-pointer text-xs"
            onClick={() => refetch()}
            title="Refresh"
          >
            &#x21BB;
          </button>
        </div>
      </Show>

      {/* Expanded view */}
      <Show when={!collapsed()}>
        <div class="p-3">
          <A href="/new">
            <Button variant="primary" class="w-full">+ New ETL</Button>
          </A>
        </div>

        <nav class="flex-1 overflow-y-auto">
          <A
            href="/connections"
            class="block px-4 py-2 text-sm text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800 no-underline border-b border-gray-200 dark:border-gray-700"
            activeClass="bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300"
          >
            Connections
          </A>
          <Show when={!etls.loading} fallback={<p class="p-4 text-sm text-gray-500">Loading...</p>}>
            <For each={etls()}>
              {(etl) => (
                <A
                  href={`/etl/${etl.name}`}
                  class="block px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-800 no-underline"
                  activeClass="bg-blue-50 dark:bg-blue-900/30 text-blue-700 dark:text-blue-300"
                >
                  <div class="font-medium">{etl.name}</div>
                  <div class="text-xs text-gray-500">
                    {etl.sources.length}S / {etl.targets.length}T / {etl.pipelines.length}P
                  </div>
                </A>
              )}
            </For>
          </Show>
        </nav>

        <div class="p-3 border-t border-gray-200 dark:border-gray-700">
          <Button variant="ghost" size="sm" onClick={() => refetch()} class="w-full">
            Refresh
          </Button>
        </div>
      </Show>
    </aside>
  );
}
