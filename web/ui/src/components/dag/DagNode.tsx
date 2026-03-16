import { createSignal, Show } from 'solid-js';
import type { DAGNode } from '../../types/dag';

interface DagNodeProps {
  node: DAGNode;
  selected: boolean;
  onSelect: (id: string) => void;
  onMove: (id: string, x: number, y: number) => void;
  onPortDragStart: (id: string, port: 'out') => void;
  onPortDragEnd: (id: string, port: 'in') => void;
  onContextMenu: (id: string, x: number, y: number) => void;
  // Test overlay
  testState?: 'idle' | 'running' | 'done' | 'error' | 'breakpoint';
  testRowCount?: number;
  testDurationMs?: number;
  hasBreakpoint?: boolean;
  onToggleBreakpoint?: (id: string) => void;
  zoom?: number;
}

const typeColors: Record<string, { bg: string; border: string; accent: string }> = {
  source:    { bg: 'bg-green-50 dark:bg-green-900/30',  border: 'border-green-400',  accent: 'bg-green-600' },
  query:     { bg: 'bg-yellow-50 dark:bg-yellow-900/30', border: 'border-yellow-400', accent: 'bg-yellow-600' },
  transform: { bg: 'bg-purple-50 dark:bg-purple-900/30', border: 'border-purple-400', accent: 'bg-purple-600' },
  function:  { bg: 'bg-orange-50 dark:bg-orange-900/30', border: 'border-orange-400', accent: 'bg-orange-600' },
  target:    { bg: 'bg-blue-50 dark:bg-blue-900/30',    border: 'border-blue-400',   accent: 'bg-blue-600' },
};

const testStateStyles: Record<string, string> = {
  running:    'ring-2 ring-yellow-400 animate-pulse',
  done:       'ring-2 ring-green-400',
  error:      'ring-2 ring-red-500',
  breakpoint: 'ring-2 ring-orange-400 animate-pulse',
};

// Storage-type specific badges: short recognizable labels
const storageBadges: Record<string, string> = {
  postgres:  'PG',
  sqlite3:   'SQ',
  sqlserver: 'MS',
  rabbitmq:  'RQ',
  webhook:   'WH',
  api:       'API',
  csv:       'CSV',
  json:      'JS',
};

// Fallback badges by node type
const typeBadges: Record<string, string> = {
  source:    'SRC',
  query:     'SQL',
  transform: 'TF',
  function:  'FN',
  target:    'TGT',
};

function getBadge(node: DAGNode): string {
  if (node.type === 'source' || node.type === 'target') {
    const st = (node.config?.storageType || '') as string;
    return storageBadges[st] || typeBadges[node.type] || '?';
  }
  return typeBadges[node.type] || '?';
}

const NODE_W = 180;
const NODE_H = 72;

export { NODE_W, NODE_H };

export default function DagNode(props: DagNodeProps) {
  const [dragging, setDragging] = createSignal(false);
  let startX = 0, startY = 0, startNodeX = 0, startNodeY = 0;
  let didDrag = false;

  const colors = () => typeColors[props.node.type] || typeColors.source;
  const testStyle = () => props.testState ? testStateStyles[props.testState] || '' : '';

  function onPointerDown(e: PointerEvent) {
    // Ignore port clicks
    if ((e.target as HTMLElement).closest('[data-port]')) return;
    e.preventDefault();
    e.stopPropagation();
    setDragging(true);
    didDrag = false;
    startX = e.clientX;
    startY = e.clientY;
    startNodeX = props.node.x;
    startNodeY = props.node.y;

    const el = e.currentTarget as HTMLElement;
    el.setPointerCapture(e.pointerId);
  }

  function onPointerMove(e: PointerEvent) {
    if (!dragging()) return;
    const z = props.zoom || 1;
    const dx = (e.clientX - startX) / z;
    const dy = (e.clientY - startY) / z;
    if (Math.abs(dx) > 3 || Math.abs(dy) > 3) didDrag = true;
    props.onMove(props.node.id, startNodeX + dx, startNodeY + dy);
  }

  function onPointerUp(_e: PointerEvent) {
    if (!dragging()) return;
    setDragging(false);
    // Only select if this was a click, not a drag
    if (!didDrag) {
      props.onSelect(props.node.id);
    }
  }

  function onContext(e: MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    props.onContextMenu(props.node.id, e.clientX, e.clientY);
  }

  return (
    <div
      style={{
        position: 'absolute',
        left: `${props.node.x}px`,
        top: `${props.node.y}px`,
        width: `${NODE_W}px`,
        height: `${NODE_H}px`,
        "pointer-events": 'auto',
      }}
    >
      <div
        class={`${colors().bg} border-2 ${props.selected ? 'border-blue-600 ring-2 ring-blue-300' : colors().border} ${testStyle()} rounded-lg px-3 py-2 cursor-move select-none relative`}
        style={{ width: `${NODE_W}px`, height: `${NODE_H}px` }}
        onPointerDown={onPointerDown}
        onPointerMove={onPointerMove}
        onPointerUp={onPointerUp}
        onContextMenu={onContext}
      >
        {/* Breakpoint indicator */}
        <Show when={props.hasBreakpoint}>
          <div class="absolute -left-2 -top-2 w-4 h-4 bg-red-500 rounded-full border-2 border-white z-10" />
        </Show>

        <div class="flex items-center gap-2">
          <span class={`text-[10px] font-bold text-white ${colors().accent} rounded px-1.5 py-0.5 leading-none tracking-wide`}>
            {getBadge(props.node)}
          </span>
          <span class="text-sm font-medium text-gray-800 dark:text-gray-200 truncate flex-1">
            {props.node.label}
          </span>
        </div>
        <div class="text-xs text-gray-500 mt-0.5 truncate">
          {props.node.type === 'source' || props.node.type === 'target'
            ? props.node.config.storageType || 'unconfigured'
            : props.node.type === 'query'
              ? 'SQL'
              : props.node.type === 'function'
                ? (props.node.config.goCode ? 'Go function' : 'empty')
                : props.node.config.fieldMappings?.length
                  ? `${props.node.config.fieldMappings.length} field mappings`
                  : props.node.config.expression || 'no transform'}
        </div>

        {/* Test result badge */}
        <Show when={props.testState === 'done' || props.testState === 'error'}>
          <div class={`absolute -bottom-2 left-1/2 -translate-x-1/2 text-[10px] font-mono px-1.5 py-0.5 rounded-full text-white ${props.testState === 'error' ? 'bg-red-500' : 'bg-green-600'}`}>
            <Show when={props.testRowCount !== undefined}>
              {props.testRowCount} rows
            </Show>
            <Show when={props.testDurationMs !== undefined}>
              {' '}{props.testDurationMs}ms
            </Show>
          </div>
        </Show>

        {/* Output port (right side) */}
        <Show when={props.node.type !== 'target'}>
          <div
            data-port="out"
            class="absolute right-[-8px] top-1/2 -translate-y-1/2 w-4 h-4 bg-gray-400 rounded-full border-2 border-white cursor-crosshair hover:bg-blue-500 hover:scale-125 transition-transform z-10"
            onPointerDown={(e) => {
              e.stopPropagation();
              e.preventDefault();
              props.onPortDragStart(props.node.id, 'out');
            }}
          />
        </Show>

        {/* Input port (left side) */}
        <Show when={props.node.type !== 'source'}>
          <div
            data-port="in"
            class="absolute left-[-8px] top-1/2 -translate-y-1/2 w-4 h-4 bg-gray-400 rounded-full border-2 border-white cursor-crosshair hover:bg-green-500 hover:scale-125 transition-transform z-10"
          />
        </Show>
      </div>
    </div>
  );
}
