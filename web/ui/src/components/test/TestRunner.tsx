import { createSignal, For, Show } from 'solid-js';
import { etlApi } from '../../api/etl';
import type { TestResponse, TestStepResult } from '../../types/dag';
import Button from '../common/Button';
import DataTable from './DataTable';

interface TestRunnerProps {
  etlName: string;
  onClose: () => void;
}

export default function TestRunner(props: TestRunnerProps) {
  const [result, setResult] = createSignal<TestResponse | null>(null);
  const [loading, setLoading] = createSignal(false);
  const [error, setError] = createSignal('');
  const [dryRun, setDryRun] = createSignal(true);

  async function runTest() {
    setLoading(true);
    setError('');
    try {
      const res = await etlApi.test(props.etlName, dryRun());
      setResult(res);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setLoading(false);
    }
  }

  const stepColors: Record<string, string> = {
    extract: 'border-green-400 bg-green-50 dark:bg-green-900/20',
    dedup: 'border-yellow-400 bg-yellow-50 dark:bg-yellow-900/20',
    transform: 'border-purple-400 bg-purple-50 dark:bg-purple-900/20',
    load: 'border-blue-400 bg-blue-50 dark:bg-blue-900/20',
  };

  return (
    <div class="fixed inset-0 z-50 bg-black/50 flex items-center justify-center p-4">
      <div class="bg-white dark:bg-gray-800 rounded-lg shadow-xl w-full max-w-4xl max-h-[90vh] flex flex-col">
        <div class="p-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
          <h2 class="text-lg font-semibold text-gray-900 dark:text-white m-0">
            Test: {props.etlName}
          </h2>
          <button onClick={props.onClose} class="text-gray-400 hover:text-gray-600 text-xl cursor-pointer">
            x
          </button>
        </div>

        <div class="p-4 flex items-center gap-4 border-b border-gray-200 dark:border-gray-700">
          <label class="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
            <input
              type="checkbox"
              checked={dryRun()}
              onChange={(e) => setDryRun(e.currentTarget.checked)}
              class="rounded"
            />
            Dry Run (skip load)
          </label>
          <Button onClick={runTest} disabled={loading()}>
            {loading() ? 'Running...' : 'Run Test'}
          </Button>
        </div>

        <div class="flex-1 overflow-y-auto p-4 space-y-4">
          <Show when={error()}>
            <div class="p-3 bg-red-50 dark:bg-red-900/20 border border-red-300 rounded text-sm text-red-700 dark:text-red-400">
              {error()}
            </div>
          </Show>

          <Show when={result()}>
            <div class="flex items-center gap-2 mb-4">
              <span class={`inline-block w-3 h-3 rounded-full ${result()!.success ? 'bg-green-500' : 'bg-red-500'}`} />
              <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
                {result()!.success ? 'All steps passed' : 'Some steps failed'}
              </span>
            </div>

            <For each={result()!.steps}>
              {(step: TestStepResult) => (
                <div class={`border-l-4 rounded p-3 ${stepColors[step.step] || 'border-gray-300 bg-gray-50'}`}>
                  <div class="flex items-center justify-between mb-1">
                    <span class="text-sm font-semibold uppercase text-gray-700 dark:text-gray-300">
                      {step.step}
                    </span>
                    <div class="flex items-center gap-3 text-xs text-gray-500">
                      <span>{step.node}</span>
                      <span>{step.rowCount} rows</span>
                      <Show when={step.durationMs}>
                        <span>{step.durationMs}ms</span>
                      </Show>
                    </div>
                  </div>

                  <Show when={step.error}>
                    <div class="text-sm text-red-600 mt-1">{step.error}</div>
                  </Show>

                  <Show when={step.sample && step.sample.length > 0}>
                    <DataTable data={step.sample} maxRows={10} />
                  </Show>
                </div>
              )}
            </For>
          </Show>

          <Show when={!result() && !loading()}>
            <div class="text-center text-gray-500 py-12">
              Click "Run Test" to walk through each pipeline step
            </div>
          </Show>
        </div>
      </div>
    </div>
  );
}
