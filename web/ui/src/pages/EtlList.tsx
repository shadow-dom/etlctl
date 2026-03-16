import { createResource, For, Show } from 'solid-js';
import { A } from '@solidjs/router';
import { etlApi } from '../api/etl';
import Header from '../components/layout/Header';
import Button from '../components/common/Button';

export default function EtlList() {
  const [etls] = createResource(() => etlApi.list());

  return (
    <div class="flex flex-col h-full">
      <Header
        title="ETL Pipelines"
        actions={
          <A href="/new">
            <Button>+ New ETL</Button>
          </A>
        }
      />
      <div class="flex-1 overflow-y-auto p-6">
        <Show
          when={!etls.loading}
          fallback={<div class="text-gray-500">Loading...</div>}
        >
          <Show
            when={etls() && etls()!.length > 0}
            fallback={
              <div class="text-center text-gray-500 py-12">
                <p>No ETL pipelines yet.</p>
                <A href="/new">
                  <Button class="mt-4">Create your first ETL</Button>
                </A>
              </div>
            }
          >
            <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
              <For each={etls()}>
                {(etl) => (
                  <A
                    href={`/etl/${etl.name}`}
                    class="block p-4 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg hover:shadow-md transition-shadow no-underline"
                  >
                    <h3 class="text-base font-semibold text-gray-900 dark:text-white m-0">
                      {etl.name}
                    </h3>
                    <div class="mt-2 flex gap-4 text-xs text-gray-500">
                      <span>{etl.sources.length} sources</span>
                      <span>{etl.targets.length} targets</span>
                      <span>{etl.pipelines.length} pipelines</span>
                    </div>
                    <Show when={etl.queries.length > 0}>
                      <div class="mt-1 text-xs text-gray-400">
                        {etl.queries.length} queries
                      </div>
                    </Show>
                  </A>
                )}
              </For>
            </div>
          </Show>
        </Show>
      </div>
    </div>
  );
}
