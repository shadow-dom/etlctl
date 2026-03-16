import { createResource, createSignal, createEffect, For, Show, onMount, onCleanup } from 'solid-js';
import { createStore, produce } from 'solid-js/store';
import loader from '@monaco-editor/loader';
import { registryApi, schemaApi } from '../../api/etl';
import Button from '../common/Button';

interface MappingEntry {
  source: string;
  target: string;
  transform: string;
}

interface Props {
  config: Record<string, any>;
  onUpdate: (config: Record<string, any>) => void;
}

const DEFAULT_GO_CODE = `// Transform function body.
// Receives: rows []map[string]string
// Must return: []map[string]string, error
//
// Example: add a computed field
//   for i, row := range rows {
//       first := row["first_name"]
//       last := row["last_name"]
//       rows[i]["full_name"] = first + " " + last
//   }
//   return rows, nil

return rows, nil`;

type TransformMode = 'expressions' | 'go';

export default function TransformNodeConfig(props: Props) {
  let editorContainer!: HTMLDivElement;
  let editor: any;

  const [transforms] = createResource(() => registryApi.transforms());
  const [previewValue, setPreviewValue] = createSignal('hello world');
  const [previewResult, setPreviewResult] = createSignal('');
  const [previewError, setPreviewError] = createSignal('');
  const [autoMapLoading, setAutoMapLoading] = createSignal(false);
  const [autoMapError, setAutoMapError] = createSignal('');
  const [unmatchedSource, setUnmatchedSource] = createSignal<string[]>([]);
  const [unmatchedTarget, setUnmatchedTarget] = createSignal<string[]>([]);

  const mode = (): TransformMode => props.config.mode || 'expressions';
  const goCode = () => props.config.goCode || DEFAULT_GO_CODE;

  // Local store for field mappings — avoids re-rendering the whole list on each keystroke
  const [mappings, setMappings] = createStore<MappingEntry[]>(
    (props.config.fieldMappings || []).map((m: any) => ({ source: m.source || '', target: m.target || '', transform: m.transform || '' }))
  );

  // Sync from parent when fieldMappings changes externally (e.g. auto-map)
  let skipSync = false;
  createEffect(() => {
    const incoming = props.config.fieldMappings || [];
    if (skipSync) { skipSync = false; return; }
    setMappings(incoming.map((m: any) => ({ source: m.source || '', target: m.target || '', transform: m.transform || '' })));
  });

  function flushMappings() {
    skipSync = true;
    props.onUpdate({ fieldMappings: [...mappings] });
  }

  onMount(async () => {
    if (mode() === 'go') {
      await initEditor();
    }
  });

  onCleanup(() => {
    editor?.dispose();
  });

  async function initEditor() {
    if (editor) return;
    const monaco = await loader.init();

    editor = monaco.editor.create(editorContainer, {
      value: goCode(),
      language: 'go',
      minimap: { enabled: false },
      lineNumbers: 'on',
      scrollBeyondLastLine: false,
      automaticLayout: true,
      fontSize: 13,
      tabSize: 4,
      theme: window.matchMedia('(prefers-color-scheme: dark)').matches
        ? 'vs-dark'
        : 'vs',
    });

    editor.onDidChangeModelContent(() => {
      props.onUpdate({ goCode: editor.getValue() });
    });
  }

  function switchMode(newMode: TransformMode) {
    props.onUpdate({ mode: newMode });
    if (newMode === 'go') {
      // Need to wait for DOM to render before initializing editor
      setTimeout(() => initEditor(), 50);
    } else {
      editor?.dispose();
      editor = null;
    }
  }

  async function preview() {
    // Build expression from first field mapping that has a transform
    const mapping = mappings.find((m) => m.transform);
    const expression = mapping?.transform || '';
    if (!expression) {
      setPreviewError('Add a field mapping with a transform expression first');
      return;
    }
    try {
      const result = await registryApi.previewTransform(expression, previewValue());
      setPreviewResult(result.result);
      setPreviewError(result.error || '');
    } catch (e: any) {
      setPreviewError(e.message);
    }
  }

  function addFieldMapping() {
    setMappings(m => [...m, { source: '', target: '', transform: '' }]);
    flushMappings();
  }

  function updateFieldMapping(index: number, key: keyof MappingEntry, value: string) {
    setMappings(produce(m => { m[index][key] = value; }));
  }

  function commitFieldMappings() {
    flushMappings();
  }

  function removeFieldMapping(index: number) {
    setMappings(m => m.filter((_, i) => i !== index));
    flushMappings();
  }

  async function autoMap() {
    const srcConfig = props.config.autoMapSource as { type: string; connection: Record<string, string>; table: string } | undefined;
    const tgtConfig = props.config.autoMapTarget as { type: string; connection: Record<string, string>; table: string } | undefined;

    if (!srcConfig || !tgtConfig) {
      setAutoMapError('Source and target connection info not available. Configure source and target nodes first.');
      return;
    }

    setAutoMapLoading(true);
    setAutoMapError('');
    setUnmatchedSource([]);
    setUnmatchedTarget([]);

    try {
      const result = await schemaApi.autoMap(srcConfig, tgtConfig);
      const newMappings = result.fields.map((f) => ({ source: f.source, target: f.target, transform: '' }));
      setMappings(newMappings);
      skipSync = true;
      props.onUpdate({ fieldMappings: newMappings });
      setUnmatchedSource(result.unmatched_source || []);
      setUnmatchedTarget(result.unmatched_target || []);
    } catch (e: any) {
      setAutoMapError(e.message || 'Auto-map failed');
    } finally {
      setAutoMapLoading(false);
    }
  }

  return (
    <div class="space-y-4">
      {/* Mode toggle */}
      <div class="flex gap-1 bg-gray-100 dark:bg-gray-700 rounded p-0.5">
        <button
          class={`flex-1 px-2 py-1 text-xs font-medium rounded transition-colors cursor-pointer ${mode() === 'expressions' ? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 dark:text-gray-400'}`}
          onClick={() => switchMode('expressions')}
        >
          Field Expressions
        </button>
        <button
          class={`flex-1 px-2 py-1 text-xs font-medium rounded transition-colors cursor-pointer ${mode() === 'go' ? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 dark:text-gray-400'}`}
          onClick={() => switchMode('go')}
        >
          Go Function
        </button>
      </div>

      {/* Expression mode */}
      <Show when={mode() === 'expressions'}>
        {/* Field mappings with inline transform expressions */}
        <div>
          <div class="flex items-center justify-between mb-2">
            <label class="text-xs font-medium text-gray-500">Field Mappings</label>
            <div class="flex gap-1">
              <Button variant="secondary" size="sm" onClick={autoMap} disabled={autoMapLoading()}>
                {autoMapLoading() ? 'Mapping...' : 'Auto-Map'}
              </Button>
              <Button variant="ghost" size="sm" onClick={addFieldMapping}>+ Add</Button>
            </div>
          </div>
          <Show when={autoMapError()}>
            <div class="text-xs text-red-600 mb-1">{autoMapError()}</div>
          </Show>
          <Show when={unmatchedSource().length > 0 || unmatchedTarget().length > 0}>
            <div class="text-xs text-amber-600 dark:text-amber-400 mb-1 space-y-0.5">
              <Show when={unmatchedSource().length > 0}>
                <div>Unmatched source: {unmatchedSource().join(', ')}</div>
              </Show>
              <Show when={unmatchedTarget().length > 0}>
                <div>Unmatched target: {unmatchedTarget().join(', ')}</div>
              </Show>
            </div>
          </Show>
          <div class="space-y-2">
            <For each={mappings}>
              {(mapping, index) => (
                <div class="border border-gray-200 dark:border-gray-600 rounded p-2 space-y-1">
                  <div class="flex gap-1 items-center">
                    <input
                      type="text"
                      value={mapping.source}
                      onInput={(e) => updateFieldMapping(index(), 'source', e.currentTarget.value)}
                      onBlur={commitFieldMappings}
                      class="flex-1 px-1.5 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                      placeholder="source field"
                    />
                    <span class="text-gray-400 text-xs">&rarr;</span>
                    <input
                      type="text"
                      value={mapping.target}
                      onInput={(e) => updateFieldMapping(index(), 'target', e.currentTarget.value)}
                      onBlur={commitFieldMappings}
                      class="flex-1 px-1.5 py-1 text-xs border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                      placeholder="target field"
                    />
                    <button
                      class="text-red-400 hover:text-red-600 text-xs cursor-pointer"
                      onClick={() => removeFieldMapping(index())}
                    >
                      &times;
                    </button>
                  </div>
                  <input
                    type="text"
                    value={mapping.transform}
                    onInput={(e) => updateFieldMapping(index(), 'transform', e.currentTarget.value)}
                    onBlur={commitFieldMappings}
                    class="w-full px-1.5 py-1 text-xs font-mono border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                    placeholder="none (pass-through) — e.g. trim|uppercase"
                  />
                </div>
              )}
            </For>
          </div>
        </div>

        {/* Available transforms */}
        <Show when={transforms()}>
          <div>
            <label class="block text-xs font-medium text-gray-500 mb-1">Available Transforms</label>
            <div class="flex flex-wrap gap-1">
              <For each={transforms()!.simple}>
                {(t) => (
                  <span class="px-1.5 py-0.5 text-xs bg-gray-100 dark:bg-gray-700 rounded font-mono">
                    {t}
                  </span>
                )}
              </For>
              <For each={transforms()!.parameterized}>
                {(t) => (
                  <span class="px-1.5 py-0.5 text-xs bg-purple-50 dark:bg-purple-900/30 text-purple-700 dark:text-purple-300 rounded font-mono">
                    {t}:<em>arg</em>
                  </span>
                )}
              </For>
            </div>
          </div>
        </Show>

        {/* Preview */}
        <div class="border border-gray-200 dark:border-gray-600 rounded p-2 space-y-2">
          <label class="block text-xs font-medium text-gray-500">Preview</label>
          <input
            type="text"
            value={previewValue()}
            onInput={(e) => setPreviewValue(e.currentTarget.value)}
            class="w-full px-2 py-1 text-sm border border-gray-300 dark:border-gray-600 rounded bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
            placeholder="test value"
          />
          <Button variant="secondary" size="sm" onClick={preview}>
            Preview
          </Button>
          <Show when={previewResult()}>
            <div class="text-sm text-green-600 font-mono">{previewResult()}</div>
          </Show>
          <Show when={previewError()}>
            <div class="text-sm text-red-600">{previewError()}</div>
          </Show>
        </div>
      </Show>

      {/* Go function mode */}
      <Show when={mode() === 'go'}>
        <div>
          <label class="block text-xs font-medium text-gray-500 mb-1">Go Transform Function</label>
          <div
            ref={editorContainer}
            class="border border-gray-300 dark:border-gray-600 rounded overflow-hidden"
            style={{ height: '400px' }}
          />
          <p class="text-xs text-gray-400 mt-1">
            Function body receives <code class="bg-gray-100 dark:bg-gray-700 px-1 rounded">rows []map[string]string</code> and must return <code class="bg-gray-100 dark:bg-gray-700 px-1 rounded">[]map[string]string, error</code>.
          </p>
        </div>
      </Show>
    </div>
  );
}
