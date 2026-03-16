import { Show, For } from 'solid-js';

interface DataTableProps {
  data: Record<string, string>[];
  maxRows?: number;
}

export default function DataTable(props: DataTableProps) {
  const rows = () => {
    const max = props.maxRows || 50;
    return props.data.slice(0, max);
  };

  const columns = () => {
    if (props.data.length === 0) return [];
    return Object.keys(props.data[0]);
  };

  return (
    <Show when={props.data.length > 0}>
      <div class="mt-2 overflow-x-auto">
        <table class="w-full text-xs border-collapse">
          <thead>
            <tr>
              <For each={columns()}>
                {(col) => (
                  <th class="text-left px-2 py-1 bg-gray-100 dark:bg-gray-700 border border-gray-200 dark:border-gray-600 font-medium text-gray-600 dark:text-gray-400">
                    {col}
                  </th>
                )}
              </For>
            </tr>
          </thead>
          <tbody>
            <For each={rows()}>
              {(row) => (
                <tr>
                  <For each={columns()}>
                    {(col) => (
                      <td class="px-2 py-1 border border-gray-200 dark:border-gray-600 text-gray-800 dark:text-gray-300 truncate max-w-[200px]">
                        {row[col]}
                      </td>
                    )}
                  </For>
                </tr>
              )}
            </For>
          </tbody>
        </table>
        <Show when={props.data.length > (props.maxRows || 50)}>
          <div class="text-xs text-gray-500 mt-1">
            Showing {props.maxRows || 50} of {props.data.length} rows
          </div>
        </Show>
      </div>
    </Show>
  );
}
