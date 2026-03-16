import { For, Show, createSignal } from 'solid-js';
import type { TestStepResult } from '../../types/dag';
import DataTable from './DataTable';
import ResizeHandle from '../common/ResizeHandle';

interface TestOverlayProps {
  steps: TestStepResult[];
  expandedStep: number | null;
  onExpandStep: (index: number) => void;
  paused: boolean;
  onContinue: () => void;
  onStop: () => void;
}

const stepColors: Record<string, { text: string; bar: string }> = {
  extract:   { text: 'text-green-700 dark:text-green-400',  bar: 'bg-green-500' },
  dedup:     { text: 'text-yellow-700 dark:text-yellow-400', bar: 'bg-yellow-500' },
  transform: { text: 'text-purple-700 dark:text-purple-400', bar: 'bg-purple-500' },
  load:      { text: 'text-blue-700 dark:text-blue-400',    bar: 'bg-blue-500' },
};

export default function TestOverlay(props: TestOverlayProps) {
  const [collapsed, setCollapsed] = createSignal(false);
  const [rawView, setRawView] = createSignal<number | null>(null);
  const [height, setHeight] = createSignal(300);

  const totalRows = () => {
    const lastStep = props.steps[props.steps.length - 1];
    return lastStep?.rowCount ?? 0;
  };

  const maxRows = () => Math.max(...props.steps.map(s => s.rowCount), 1);

  const totalTime = () => props.steps.reduce((sum, s) => sum + (s.durationMs || 0), 0);

  function handleResize(delta: number) {
    setHeight(h => Math.max(100, Math.min(window.innerHeight * 0.8, h - delta)));
  }

  return (
    <div
      class="bg-white dark:bg-gray-800 border-t border-gray-300 dark:border-gray-600 flex flex-col flex-shrink-0"
      style={{ height: collapsed() ? '40px' : `${height()}px` }}
    >
      {/* Resize handle (top edge) */}
      <Show when={!collapsed()}>
        <ResizeHandle direction="vertical" onResize={handleResize} />
      </Show>

      {/* Header bar */}
      <div
        class="h-10 flex items-center justify-between px-4 border-b border-gray-200 dark:border-gray-700 cursor-pointer select-none flex-shrink-0"
        onClick={() => setCollapsed(!collapsed())}
      >
        <div class="flex items-center gap-3">
          <span class="text-xs font-semibold text-gray-600 dark:text-gray-400 uppercase">
            Test Results
          </span>
          <span class="text-xs text-gray-500 font-mono">
            {props.steps.length} steps
          </span>
          <span class="text-xs text-gray-500 font-mono">
            {totalRows()} rows
          </span>
          <span class="text-xs text-gray-500 font-mono">
            {totalTime()}ms
          </span>
          <Show when={props.steps.some(s => s.error)}>
            <span class="text-xs bg-red-100 text-red-700 px-1.5 py-0.5 rounded">
              errors
            </span>
          </Show>
        </div>
        <span class="text-gray-400 text-xs">{collapsed() ? '\u25B2' : '\u25BC'}</span>
      </div>

      {/* Steps list */}
      <Show when={!collapsed()}>
        <div class="overflow-y-auto flex-1">
          <For each={props.steps}>
            {(step, index) => {
              const colors = () => stepColors[step.step] || stepColors.extract;
              const isExpanded = () => props.expandedStep === index();
              const isRawView = () => rawView() === index();
              const isPausedHere = () => props.paused && index() === props.steps.length - 1;

              return (
                <div class={`border-b border-gray-100 dark:border-gray-700/50 ${step.error ? 'bg-red-50/50 dark:bg-red-900/10' : ''} ${isPausedHere() ? 'ring-2 ring-inset ring-orange-400' : ''}`}>
                  {/* Step row */}
                  <div
                    class="flex items-center gap-3 px-4 py-2 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700/30"
                    onClick={() => props.onExpandStep(index())}
                  >
                    {/* Step number */}
                    <span class="text-xs font-mono text-gray-400 w-5 text-right">
                      {index() + 1}
                    </span>

                    {/* Step type badge */}
                    <div class="flex items-center gap-1.5 min-w-[90px]">
                      <span class={`w-2 h-2 rounded-full ${colors().bar}`} />
                      <span class={`text-xs font-semibold uppercase ${colors().text}`}>
                        {step.step}
                      </span>
                    </div>

                    {/* Node name */}
                    <span class="text-sm text-gray-700 dark:text-gray-300 min-w-[120px] truncate">
                      {step.node}
                    </span>

                    {/* Row count with visual bar */}
                    <div class="flex items-center gap-2 min-w-[140px]">
                      <div class="w-20 h-2 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                        <div
                          class={`h-full ${colors().bar} rounded-full transition-all`}
                          style={{ width: `${Math.min(100, (step.rowCount / maxRows()) * 100)}%` }}
                        />
                      </div>
                      <span class="text-xs font-mono text-gray-600 dark:text-gray-400">
                        {step.rowCount.toLocaleString()} rows
                      </span>
                    </div>

                    {/* Duration */}
                    <span class="text-xs font-mono text-gray-500 min-w-[60px] text-right">
                      {step.durationMs || 0}ms
                    </span>

                    {/* Status indicators */}
                    <Show when={step.error}>
                      <span class="text-xs bg-red-100 dark:bg-red-900/40 text-red-600 px-1.5 py-0.5 rounded">
                        error
                      </span>
                    </Show>
                    <Show when={isPausedHere()}>
                      <span class="text-xs bg-orange-100 dark:bg-orange-900/40 text-orange-600 px-1.5 py-0.5 rounded font-medium">
                        paused
                      </span>
                    </Show>

                    {/* Expand arrow */}
                    <span class="text-gray-400 text-xs ml-auto">
                      {isExpanded() ? '\u25BC' : '\u25B6'}
                    </span>
                  </div>

                  {/* Expanded detail */}
                  <Show when={isExpanded()}>
                    <div class="px-4 pb-3 space-y-2 bg-gray-50/50 dark:bg-gray-900/30">
                      {/* Step metadata */}
                      <div class="flex items-center gap-4 text-xs text-gray-500 pt-1">
                        <span>Type: <strong class={colors().text}>{step.step}</strong></span>
                        <span>Node: <strong>{step.node}</strong></span>
                        <span>Rows: <strong>{step.rowCount.toLocaleString()}</strong></span>
                        <span>Duration: <strong>{step.durationMs || 0}ms</strong></span>
                        <Show when={step.sample}>
                          <span>Sample: <strong>{step.sample.length} rows</strong></span>
                        </Show>
                        <div class="flex-1" />
                        <Show when={step.sample && step.sample.length > 0}>
                          <button
                            class="text-xs text-blue-600 hover:text-blue-800 underline cursor-pointer"
                            onClick={(e) => { e.stopPropagation(); setRawView(isRawView() ? null : index()); }}
                          >
                            {isRawView() ? 'Table View' : 'Raw JSON'}
                          </button>
                        </Show>
                      </div>

                      <Show when={step.error}>
                        <div class="text-sm text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 rounded p-2 font-mono">
                          {step.error}
                        </div>
                      </Show>

                      <Show when={step.sample && step.sample.length > 0}>
                        <Show when={!isRawView()}>
                          <div class="text-xs font-medium text-gray-500 mb-1">
                            Data Preview ({Math.min(step.sample.length, 20)} of {step.rowCount.toLocaleString()} rows)
                          </div>
                          <DataTable data={step.sample} maxRows={20} />
                        </Show>
                        <Show when={isRawView()}>
                          <div class="text-xs font-medium text-gray-500 mb-1">
                            Raw JSON ({Math.min(step.sample.length, 10)} of {step.rowCount.toLocaleString()} rows)
                          </div>
                          <pre class="text-xs font-mono bg-gray-100 dark:bg-gray-900 rounded p-2 overflow-auto max-h-48 text-gray-700 dark:text-gray-300">
                            {JSON.stringify(step.sample.slice(0, 10), null, 2)}
                          </pre>
                        </Show>
                      </Show>

                      <Show when={!step.sample || step.sample.length === 0}>
                        <div class="text-xs text-gray-400 italic py-2">
                          {step.rowCount === 0
                            ? 'No data at this step - 0 rows returned'
                            : 'No sample data available'}
                        </div>
                      </Show>
                    </div>
                  </Show>
                </div>
              );
            }}
          </For>
        </div>
      </Show>
    </div>
  );
}
