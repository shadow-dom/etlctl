import { Show } from 'solid-js';
import type { DAGEdge, DAGNode } from '../../types/dag';
import { NODE_W, NODE_H } from './DagNode';

interface DagEdgeProps {
  edge: DAGEdge;
  nodes: DAGNode[];
  active?: boolean;
  rowCount?: number;
  selected?: boolean;
  onSelect?: (edgeId: string) => void;
  onDelete?: (edgeId: string) => void;
}

export default function DagEdge(props: DagEdgeProps) {
  const sourceNode = () => props.nodes.find(n => n.id === props.edge.source);
  const targetNode = () => props.nodes.find(n => n.id === props.edge.target);

  const coords = () => {
    const src = sourceNode();
    const tgt = targetNode();
    if (!src || !tgt) return null;

    const x1 = src.x + NODE_W;
    const y1 = src.y + NODE_H / 2;
    const x2 = tgt.x;
    const y2 = tgt.y + NODE_H / 2;
    return { x1, y1, x2, y2 };
  };

  const path = () => {
    const c = coords();
    if (!c) return '';
    const cx = (c.x1 + c.x2) / 2;
    return `M ${c.x1} ${c.y1} C ${cx} ${c.y1}, ${cx} ${c.y2}, ${c.x2} ${c.y2}`;
  };

  const midpoint = () => {
    const c = coords();
    if (!c) return { x: 0, y: 0 };
    return { x: (c.x1 + c.x2) / 2, y: (c.y1 + c.y2) / 2 };
  };

  function handleClick(e: MouseEvent) {
    e.stopPropagation();
    props.onSelect?.(props.edge.id);
  }

  return (
    <>
      {/* Invisible wider hit area for clicking */}
      <path
        d={path()}
        fill="none"
        stroke="transparent"
        stroke-width="12"
        style={{ cursor: 'pointer' }}
        onClick={handleClick}
      />

      {/* Visible edge */}
      <path
        d={path()}
        fill="none"
        stroke={props.selected ? '#ef4444' : props.active ? '#3b82f6' : '#94a3b8'}
        stroke-width={props.selected ? 3 : props.active ? 3 : 2}
        stroke-dasharray={props.selected ? '6 3' : 'none'}
        marker-end={props.active ? 'url(#arrowhead-active)' : 'url(#arrowhead)'}
        style={{ cursor: 'pointer', "pointer-events": 'none' }}
      />

      {/* Animated flow dots when active */}
      <Show when={props.active}>
        <circle r="4" fill="#3b82f6">
          <animateMotion dur="1.5s" repeatCount="indefinite" path={path()} />
        </circle>
      </Show>

      {/* Delete button when selected */}
      <Show when={props.selected}>
        <g
          transform={`translate(${midpoint().x}, ${midpoint().y - 18})`}
          style={{ cursor: 'pointer' }}
          onClick={(e: MouseEvent) => { e.stopPropagation(); props.onDelete?.(props.edge.id); }}
        >
          <rect x="-12" y="-12" width="24" height="24" rx="12" fill="#ef4444" />
          <line x1="-5" y1="-5" x2="5" y2="5" stroke="white" stroke-width="2" stroke-linecap="round" />
          <line x1="5" y1="-5" x2="-5" y2="5" stroke="white" stroke-width="2" stroke-linecap="round" />
        </g>
      </Show>

      {/* Row count badge on edge */}
      <Show when={!props.selected && props.rowCount !== undefined && props.rowCount! > 0}>
        <g transform={`translate(${midpoint().x}, ${midpoint().y})`}>
          <rect
            x="-20"
            y="-10"
            width="40"
            height="20"
            rx="10"
            fill={props.active ? '#3b82f6' : '#64748b'}
          />
          <text
            x="0"
            y="5"
            text-anchor="middle"
            font-size="10"
            font-weight="bold"
            fill="white"
            font-family="system-ui"
          >
            {props.rowCount}
          </text>
        </g>
      </Show>
    </>
  );
}
