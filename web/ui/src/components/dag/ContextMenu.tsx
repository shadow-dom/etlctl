import { Show, onMount, onCleanup } from 'solid-js';

export interface ContextMenuItem {
  label: string;
  icon?: string;
  danger?: boolean;
  separator?: boolean;
  action: () => void;
}

interface ContextMenuProps {
  x: number;
  y: number;
  items: ContextMenuItem[];
  onClose: () => void;
}

export default function ContextMenu(props: ContextMenuProps) {
  let menuRef!: HTMLDivElement;

  function handleClickOutside(e: MouseEvent) {
    if (menuRef && !menuRef.contains(e.target as Node)) {
      props.onClose();
    }
  }

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Escape') props.onClose();
  }

  onMount(() => {
    document.addEventListener('click', handleClickOutside, true);
    document.addEventListener('keydown', handleKeyDown);
  });

  onCleanup(() => {
    document.removeEventListener('click', handleClickOutside, true);
    document.removeEventListener('keydown', handleKeyDown);
  });

  return (
    <div
      ref={menuRef}
      class="fixed z-50 min-w-[160px] bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-lg py-1 text-sm"
      style={{ left: `${props.x}px`, top: `${props.y}px` }}
    >
      {props.items.map((item) => (
        <Show when={!item.separator} fallback={
          <div class="border-t border-gray-200 dark:border-gray-700 my-1" />
        }>
          <button
            class={`w-full text-left px-3 py-1.5 cursor-pointer flex items-center gap-2 ${
              item.danger
                ? 'text-red-600 hover:bg-red-50 dark:hover:bg-red-900/30'
                : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
            }`}
            onClick={() => { item.action(); props.onClose(); }}
          >
            <Show when={item.icon}>
              <span class="w-4 text-center">{item.icon}</span>
            </Show>
            {item.label}
          </button>
        </Show>
      ))}
    </div>
  );
}
