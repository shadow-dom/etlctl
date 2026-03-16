import { createResource, createSignal, For, Show, onCleanup } from 'solid-js';
import { useParams } from '@solidjs/router';
import { etlApi, versionsApi } from '../api/etl';
import type { ETLVersion } from '../api/etl';
import Header from '../components/layout/Header';
import Button from '../components/common/Button';
import DagEditor from '../components/dag/DagEditor';
import loader from '@monaco-editor/loader';

export default function EtlDetail() {
  const params = useParams<{ name: string }>();
  const [etl, { refetch }] = createResource(() => params.name, (name) => etlApi.get(name));
  const [validationResult, setValidationResult] = createSignal<string | null>(null);
  const [editMode, setEditMode] = createSignal<'dag' | 'yaml'>('dag');
  const [yamlContent, setYamlContent] = createSignal('');
  const [yamlDirty, setYamlDirty] = createSignal(false);
  const [yamlSaving, setYamlSaving] = createSignal(false);
  const [showVersions, setShowVersions] = createSignal(false);
  const [versionsList, setVersionsList] = createSignal<ETLVersion[]>([]);

  let yamlEditorContainer: HTMLDivElement | undefined;
  let yamlEditor: any = null;

  onCleanup(() => {
    yamlEditor?.dispose();
    yamlEditor = null;
  });

  async function handleValidate() {
    try {
      const result = await etlApi.validate(params.name);
      if (result.valid) {
        setValidationResult('Valid!');
      } else {
        setValidationResult(`Errors: ${result.errors.join(', ')}`);
      }
      setTimeout(() => setValidationResult(null), 5000);
    } catch (e: any) {
      setValidationResult(`Error: ${e.message}`);
    }
  }

  async function switchToYaml() {
    setEditMode('yaml');
    try {
      const yaml = await etlApi.getRawYaml(params.name);
      setYamlContent(yaml);
      setYamlDirty(false);
      setTimeout(() => initYamlEditor(), 50);
    } catch (e: any) {
      setYamlContent(`# Error loading YAML: ${e.message}`);
      setTimeout(() => initYamlEditor(), 50);
    }
  }

  async function switchToDag() {
    if (yamlDirty() && !confirm('You have unsaved YAML changes. Discard them?')) return;
    yamlEditor?.dispose();
    yamlEditor = null;
    setEditMode('dag');
    refetch();
  }

  async function initYamlEditor() {
    if (yamlEditor || !yamlEditorContainer) return;
    const monaco = await loader.init();
    yamlEditor = monaco.editor.create(yamlEditorContainer, {
      value: yamlContent(),
      language: 'yaml',
      minimap: { enabled: false },
      lineNumbers: 'on',
      scrollBeyondLastLine: false,
      automaticLayout: true,
      fontSize: 13,
      tabSize: 2,
      theme: window.matchMedia('(prefers-color-scheme: dark)').matches ? 'vs-dark' : 'vs',
    });
    yamlEditor.onDidChangeModelContent(() => {
      setYamlContent(yamlEditor.getValue());
      setYamlDirty(true);
    });
  }

  async function saveYaml() {
    setYamlSaving(true);
    try {
      await etlApi.updateRawYaml(params.name, yamlContent());
      setYamlDirty(false);
    } catch (e: any) {
      alert(`Failed to save: ${e.message}`);
    } finally {
      setYamlSaving(false);
    }
  }

  async function toggleVersions() {
    if (!showVersions()) {
      try {
        const v = await versionsApi.list(params.name);
        setVersionsList(v);
      } catch {
        setVersionsList([]);
      }
    }
    setShowVersions(!showVersions());
  }

  async function saveVersion() {
    const label = prompt('Version label (optional):');
    if (label === null) return;
    try {
      await versionsApi.save(params.name, label || '', false);
      const v = await versionsApi.list(params.name);
      setVersionsList(v);
    } catch (e: any) {
      alert(`Failed to save version: ${e.message}`);
    }
  }

  async function restoreVersion(version: number) {
    if (!confirm(`Restore version ${version}? This will overwrite the current config.`)) return;
    try {
      await versionsApi.restore(params.name, version);
      setShowVersions(false);
      refetch();
      if (editMode() === 'yaml') {
        const yaml = await etlApi.getRawYaml(params.name);
        setYamlContent(yaml);
        setYamlDirty(false);
        yamlEditor?.setValue(yaml);
      }
    } catch (e: any) {
      alert(`Failed to restore: ${e.message}`);
    }
  }

  return (
    <div class="flex flex-col h-full">
      <Header
        title={params.name}
        actions={
          <>
            {/* Mode toggle */}
            <div class="flex gap-0.5 bg-gray-100 dark:bg-gray-700 rounded p-0.5 mr-2">
              <button
                class={`px-3 py-1 text-xs font-medium rounded transition-colors cursor-pointer ${editMode() === 'dag' ? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 dark:text-gray-400'}`}
                onClick={() => switchToDag()}
              >
                DAG
              </button>
              <button
                class={`px-3 py-1 text-xs font-medium rounded transition-colors cursor-pointer ${editMode() === 'yaml' ? 'bg-white dark:bg-gray-600 text-gray-900 dark:text-white shadow-sm' : 'text-gray-500 dark:text-gray-400'}`}
                onClick={() => switchToYaml()}
              >
                YAML
              </button>
            </div>

            <Show when={editMode() === 'yaml'}>
              <Show when={validationResult()}>
                <span class={`text-xs px-2 py-1 rounded ${validationResult()?.startsWith('Valid') ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>
                  {validationResult()}
                </span>
              </Show>
              <Button variant="ghost" size="sm" onClick={handleValidate}>Validate</Button>
              <Button variant="primary" size="sm" onClick={saveYaml} disabled={!yamlDirty() || yamlSaving()}>
                {yamlSaving() ? 'Saving...' : 'Save YAML'}
              </Button>
            </Show>

            {/* Version management */}
            <div class="border-l border-gray-200 dark:border-gray-700 h-6 mx-1" />
            <div class="relative">
              <div class="flex gap-1">
                <Button variant="ghost" size="sm" onClick={toggleVersions}>Versions</Button>
                <Button variant="ghost" size="sm" onClick={saveVersion}>Save Version</Button>
              </div>
              <Show when={showVersions()}>
                <div class="absolute right-0 top-8 w-72 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg z-50 max-h-64 overflow-auto">
                  <div class="p-2 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
                    <span class="text-xs font-medium text-gray-500">Version History</span>
                    <button class="text-xs text-gray-400 hover:text-gray-600 cursor-pointer" onClick={() => setShowVersions(false)}>x</button>
                  </div>
                  <For each={versionsList()}>
                    {(v) => (
                      <div class="px-3 py-2 border-b border-gray-100 dark:border-gray-700 flex items-center justify-between hover:bg-gray-50 dark:hover:bg-gray-700/50">
                        <div>
                          <div class="text-sm font-medium text-gray-900 dark:text-white">
                            v{v.version}
                            <Show when={v.label}>
                              <span class="ml-1 text-gray-500">— {v.label}</span>
                            </Show>
                            <Show when={v.isDraft}>
                              <span class="ml-1 text-xs bg-yellow-100 text-yellow-700 px-1 rounded">draft</span>
                            </Show>
                          </div>
                          <div class="text-xs text-gray-400">{new Date(v.createdAt).toLocaleString()}</div>
                        </div>
                        <button
                          class="text-xs text-blue-600 hover:text-blue-800 cursor-pointer"
                          onClick={() => restoreVersion(v.version)}
                        >
                          Restore
                        </button>
                      </div>
                    )}
                  </For>
                  <Show when={versionsList().length === 0}>
                    <div class="p-3 text-xs text-gray-500 text-center">No versions saved yet</div>
                  </Show>
                </div>
              </Show>
            </div>

            <div class="border-l border-gray-200 dark:border-gray-700 h-6 mx-1" />
            <Button variant="danger" size="sm" onClick={async () => {
              if (confirm(`Delete ${params.name}?`)) {
                await etlApi.delete(params.name);
                window.location.href = '/';
              }
            }}>
              Delete
            </Button>
          </>
        }
      />

      <Show
        when={!etl.loading && etl()}
        fallback={<div class="flex-1 flex items-center justify-center text-gray-500">Loading...</div>}
      >
        {/* DAG mode */}
        <Show when={editMode() === 'dag'}>
          <div class="flex-1 overflow-hidden">
            <DagEditor
              etl={etl()!}
              onValidate={handleValidate}
              onTest={() => {}}
              validationResult={validationResult()}
            />
          </div>
        </Show>

        {/* YAML mode */}
        <Show when={editMode() === 'yaml'}>
          <div class="flex-1 overflow-hidden">
            <div
              ref={yamlEditorContainer}
              class="h-full w-full"
            />
          </div>
        </Show>
      </Show>
    </div>
  );
}
