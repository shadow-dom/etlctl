import { createSignal, onCleanup } from 'solid-js';

interface ResizeHandleProps {
  /** 'horizontal' = drag left/right, 'vertical' = drag up/down */
  direction: 'horizontal' | 'vertical';
  /** Called with the delta in pixels while dragging */
  onResize: (delta: number) => void;
  /** Which side the handle sits on. Affects cursor and hit area placement. */
  side?: 'left' | 'right' | 'top' | 'bottom';
}

export default function ResizeHandle(props: ResizeHandleProps) {
  const [dragging, setDragging] = createSignal(false);
  let startPos = 0;

  function onPointerDown(e: PointerEvent) {
    e.preventDefault();
    e.stopPropagation();
    setDragging(true);
    startPos = props.direction === 'horizontal' ? e.clientX : e.clientY;
    (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
  }

  function onPointerMove(e: PointerEvent) {
    if (!dragging()) return;
    const current = props.direction === 'horizontal' ? e.clientX : e.clientY;
    const delta = current - startPos;
    startPos = current;
    props.onResize(delta);
  }

  function onPointerUp() {
    setDragging(false);
  }

  // Prevent text selection while dragging
  function preventSelect(e: Event) {
    if (dragging()) e.preventDefault();
  }

  if (typeof document !== 'undefined') {
    document.addEventListener('selectstart', preventSelect);
    onCleanup(() => document.removeEventListener('selectstart', preventSelect));
  }

  const isHorizontal = () => props.direction === 'horizontal';

  return (
    <div
      class={`${isHorizontal() ? 'w-1.5 cursor-col-resize hover:bg-blue-400/40' : 'h-1.5 cursor-row-resize hover:bg-blue-400/40'} ${dragging() ? 'bg-blue-500/50' : 'bg-transparent'} transition-colors flex-shrink-0 z-30`}
      onPointerDown={onPointerDown}
      onPointerMove={onPointerMove}
      onPointerUp={onPointerUp}
    >
      {/* Visible drag indicator on hover */}
      <div class={`${isHorizontal() ? 'w-0.5 h-8 mx-auto mt-[50%] -translate-y-1/2' : 'h-0.5 w-8 my-auto ml-[50%] -translate-x-1/2'} rounded-full bg-gray-400/0 group-hover:bg-gray-400/60`} />
    </div>
  );
}
