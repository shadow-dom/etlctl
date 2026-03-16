import { createSignal, For, Show } from 'solid-js';
import Button from '../common/Button';

interface Props {
  value: string;
  onChange: (csv: string) => void;
}

export default function CsvInlineEditor(props: Props) {
  const parseCSV = (csv: string): string[][] => {
    return csv.split('\n').filter(line => line.trim()).map(line => line.split(',').map(cell => cell.trim()));
  };

  const [rows, setRows] = createSignal<string[][]>(parseCSV(props.value));

  function syncToParent() {
    const csv = rows().map(row => row.join(',')).join('\n');
    props.onChange(csv);
  }

  function updateCell(rowIdx: number, colIdx: number, value: string) {
    const updated = rows().map((row, ri) =>
      ri === rowIdx ? row.map((cell, ci) => ci === colIdx ? value : cell) : [...row]
    );
    setRows(updated);
    const csv = updated.map(row => row.join(',')).join('\n');
    props.onChange(csv);
  }

  function addRow() {
    const colCount = rows()[0]?.length || 2;
    const newRow = Array(colCount).fill('');
    setRows([...rows(), newRow]);
    syncToParent();
  }

  function addColumn() {
    const updated = rows().map((row, i) => [...row, i === 0 ? `col${row.length + 1}` : '']);
    setRows(updated);
    syncToParent();
  }

  function removeRow(idx: number) {
    if (rows().length <= 1) return; // Keep at least header
    setRows(rows().filter((_, i) => i !== idx));
    syncToParent();
  }

  function removeColumn(colIdx: number) {
    if ((rows()[0]?.length || 0) <= 1) return;
    const updated = rows().map(row => row.filter((_, ci) => ci !== colIdx));
    setRows(updated);
    syncToParent();
  }

  return (
    <div class="space-y-2">
      <div class="border border-gray-200 dark:border-gray-600 rounded overflow-auto max-h-64">
        <table class="w-full text-xs">
          <For each={rows()}>
            {(row, rowIdx) => (
              <tr class={rowIdx() === 0 ? 'bg-gray-50 dark:bg-gray-700 font-medium' : 'border-t border-gray-200 dark:border-gray-600'}>
                <For each={row}>
                  {(cell, colIdx) => (
                    <td class="p-0">
                      <input
                        type="text"
                        value={cell}
                        onInput={(e) => updateCell(rowIdx(), colIdx(), e.currentTarget.value)}
                        class="w-full px-1.5 py-1 text-xs border-0 bg-transparent focus:bg-blue-50 dark:focus:bg-blue-900/20 outline-none text-gray-900 dark:text-white"
                        style={{ "min-width": "60px" }}
                      />
                    </td>
                  )}
                </For>
                <td class="w-6 text-center">
                  <Show when={rowIdx() > 0}>
                    <button
                      class="text-red-400 hover:text-red-600 text-xs cursor-pointer"
                      onClick={() => removeRow(rowIdx())}
                      title="Remove row"
                    >&times;</button>
                  </Show>
                </td>
              </tr>
            )}
          </For>
        </table>
      </div>
      <div class="flex gap-1">
        <Button variant="ghost" size="sm" onClick={addRow}>+ Row</Button>
        <Button variant="ghost" size="sm" onClick={addColumn}>+ Column</Button>
      </div>
    </div>
  );
}
