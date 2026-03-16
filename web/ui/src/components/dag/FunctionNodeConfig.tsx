import { onMount, onCleanup } from 'solid-js';
import loader from '@monaco-editor/loader';

interface Props {
  config: Record<string, any>;
  onUpdate: (config: Record<string, any>) => void;
}

const DEFAULT_GO_CODE = `// Function body.
// Receives: rows []map[string]string
// Must return: []map[string]string, error
//
// Example: filter rows and add a timestamp
//   var result []map[string]string
//   for _, row := range rows {
//       if row["status"] == "active" {
//           row["processed_at"] = time.Now().Format(time.RFC3339)
//           result = append(result, row)
//       }
//   }
//   return result, nil

return rows, nil`;

export default function FunctionNodeConfig(props: Props) {
  let editorContainer!: HTMLDivElement;
  let editor: any;

  const goCode = () => props.config.goCode || DEFAULT_GO_CODE;

  onMount(async () => {
    await initEditor();
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

  return (
    <div class="space-y-3">
      <div>
        <label class="block text-xs font-medium text-gray-500 mb-1">Go Function</label>
        <div
          ref={editorContainer}
          class="border border-gray-300 dark:border-gray-600 rounded overflow-hidden"
          style={{ height: '400px' }}
        />
        <p class="text-xs text-gray-400 mt-1">
          Receives <code class="bg-gray-100 dark:bg-gray-700 px-1 rounded">rows []map[string]string</code> and
          must return <code class="bg-gray-100 dark:bg-gray-700 px-1 rounded">[]map[string]string, error</code>.
        </p>
      </div>
    </div>
  );
}
