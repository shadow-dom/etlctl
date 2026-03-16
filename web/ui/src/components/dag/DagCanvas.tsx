import { For, Show, createSignal } from 'solid-js';
import type { DAGNode, DAGEdge, TestStepResult } from '../../types/dag';
import { isValidConnection } from '../../stores/dagStore';
import DagNodeComponent, { NODE_W, NODE_H } from './DagNode';
import DagEdgeComponent from './DagEdge';

interface DagCanvasProps {
  nodes: DAGNode[];
  edges: DAGEdge[];
  selectedNodeId: string | null;
  onSelectNode: (id: string | null) => void;
  onMoveNode: (id: string, x: number, y: number) => void;
  onAddEdge: (sourceId: string, targetId: string) => void;
  onRemoveEdge: (edgeId: string) => void;
  onContextMenu: (nodeId: string, x: number, y: number) => void;
  // Test overlay
  testResults?: Map<string, TestStepResult>;
  breakpoints?: Set<string>;
  activeTestNode?: string | null;
  onToggleBreakpoint?: (nodeId: string) => void;
}

export default function DagCanvas(props: DagCanvasProps) {
  let svgRef!: SVGSVGElement;
  let containerRef!: HTMLDivElement;

  // Edge drawing state
  const [edgeDrawing, setEdgeDrawing] = createSignal<{
    sourceId: string;
    mouseX: number;
    mouseY: number;
  } | null>(null);

  // Selected edge
  const [selectedEdgeId, setSelectedEdgeId] = createSignal<string | null>(null);

  // Pan/zoom state
  const [pan, setPan] = createSignal({ x: 0, y: 0 });
  const [zoom, setZoom] = createSignal(1);
  const [panning, setPanning] = createSignal(false);
  let panStart = { x: 0, y: 0, panX: 0, panY: 0 };

  function screenToSvg(clientX: number, clientY: number) {
    const rect = containerRef.getBoundingClientRect();
    const z = zoom();
    const p = pan();
    return {
      x: (clientX - rect.left - p.x) / z,
      y: (clientY - rect.top - p.y) / z,
    };
  }

  function handlePortDragStart(nodeId: string) {
    const node = props.nodes.find(n => n.id === nodeId);
    if (!node) return;
    setEdgeDrawing({
      sourceId: nodeId,
      mouseX: node.x + NODE_W,
      mouseY: node.y + NODE_H / 2,
    });
  }

  function handleSvgPointerMove(e: PointerEvent) {
    // Pan
    if (panning()) {
      const dx = e.clientX - panStart.x;
      const dy = e.clientY - panStart.y;
      if (Math.abs(dx) > 3 || Math.abs(dy) > 3) didPan = true;
      setPan({ x: panStart.panX + dx, y: panStart.panY + dy });
      return;
    }

    // Edge drawing
    const drawing = edgeDrawing();
    if (!drawing) return;
    const pt = screenToSvg(e.clientX, e.clientY);
    setEdgeDrawing({
      ...drawing,
      mouseX: pt.x,
      mouseY: pt.y,
    });
  }

  function handlePointerUp(e: PointerEvent) {
    if (panning()) {
      setPanning(false);
      return;
    }

    const drawing = edgeDrawing();
    if (!drawing) return;

    const pt = screenToSvg(e.clientX, e.clientY);

    // Check proximity to any node's input port (left side)
    const sourceNode = props.nodes.find(n => n.id === drawing.sourceId);
    for (const node of props.nodes) {
      if (node.id === drawing.sourceId) continue;
      if (node.type === 'source') continue; // Sources don't have input ports
      if (sourceNode && !isValidConnection(sourceNode.type, node.type)) continue;
      const portX = node.x;
      const portY = node.y + NODE_H / 2;
      const dist = Math.sqrt((pt.x - portX) ** 2 + (pt.y - portY) ** 2);
      if (dist < 30) {
        props.onAddEdge(drawing.sourceId, node.id);
        break;
      }
    }

    setEdgeDrawing(null);
  }

  let didPan = false;

  function handleCanvasPointerDown(e: PointerEvent) {
    // Left-click or middle-click on canvas background starts panning
    if (e.button === 0 || e.button === 1) {
      e.preventDefault();
      setPanning(true);
      didPan = false;
      panStart = { x: e.clientX, y: e.clientY, panX: pan().x, panY: pan().y };
      return;
    }
  }

  function handleCanvasClick(e: MouseEvent) {
    // Only deselect if this was a click (not a drag) on the background
    if (didPan) return;
    if (e.target === svgRef || (e.target as SVGElement).tagName === 'rect') {
      props.onSelectNode(null);
      setSelectedEdgeId(null);
    }
  }

  function handleWheel(e: WheelEvent) {
    e.preventDefault();
    const rect = containerRef.getBoundingClientRect();
    const mouseX = e.clientX - rect.left;
    const mouseY = e.clientY - rect.top;

    const oldZoom = zoom();
    const delta = e.deltaY > 0 ? 0.9 : 1.1;
    const newZoom = Math.min(3, Math.max(0.2, oldZoom * delta));

    // Zoom toward cursor
    const p = pan();
    const newPanX = mouseX - (mouseX - p.x) * (newZoom / oldZoom);
    const newPanY = mouseY - (mouseY - p.y) * (newZoom / oldZoom);

    setZoom(newZoom);
    setPan({ x: newPanX, y: newPanY });
  }

  function handleKeyDown(e: KeyboardEvent) {
    // Cancel edge drawing with Escape
    if (e.key === 'Escape' && edgeDrawing()) {
      setEdgeDrawing(null);
      return;
    }
    if ((e.key === 'Delete' || e.key === 'Backspace') && selectedEdgeId()) {
      props.onRemoveEdge(selectedEdgeId()!);
      setSelectedEdgeId(null);
    }
    // Reset zoom with Ctrl+0
    if (e.key === '0' && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      setZoom(1);
      setPan({ x: 0, y: 0 });
    }
  }

  function resetView() {
    setZoom(1);
    setPan({ x: 0, y: 0 });
  }

  function fitToView() {
    if (props.nodes.length === 0) return;
    const rect = containerRef.getBoundingClientRect();
    const minX = Math.min(...props.nodes.map(n => n.x));
    const maxX = Math.max(...props.nodes.map(n => n.x + NODE_W));
    const minY = Math.min(...props.nodes.map(n => n.y));
    const maxY = Math.max(...props.nodes.map(n => n.y + NODE_H));

    const dagW = maxX - minX + 100;
    const dagH = maxY - minY + 100;
    const scaleX = rect.width / dagW;
    const scaleY = rect.height / dagH;
    const newZoom = Math.min(scaleX, scaleY, 1.5);

    const centerX = (minX + maxX) / 2;
    const centerY = (minY + maxY) / 2;

    setZoom(newZoom);
    setPan({
      x: rect.width / 2 - centerX * newZoom,
      y: rect.height / 2 - centerY * newZoom,
    });
  }

  const drawingPath = () => {
    const d = edgeDrawing();
    if (!d) return '';
    const sourceNode = props.nodes.find(n => n.id === d.sourceId);
    if (!sourceNode) return '';

    const x1 = sourceNode.x + NODE_W;
    const y1 = sourceNode.y + NODE_H / 2;
    const x2 = d.mouseX;
    const y2 = d.mouseY;
    const cx = (x1 + x2) / 2;

    return `M ${x1} ${y1} C ${cx} ${y1}, ${cx} ${y2}, ${x2} ${y2}`;
  };

  function getEdgeTestInfo(edge: DAGEdge) {
    if (!props.testResults) return null;
    return props.testResults.get(edge.source) || null;
  }

  function nodeTestState(nodeId: string): 'idle' | 'running' | 'done' | 'error' | 'breakpoint' {
    if (props.activeTestNode === nodeId) return 'running';
    if (props.breakpoints?.has(nodeId) && props.activeTestNode === nodeId) return 'breakpoint';
    const result = props.testResults?.get(nodeId);
    if (!result) return 'idle';
    if (result.error) return 'error';
    return 'done';
  }

  return (
    <div
      ref={containerRef}
      class="flex-1 relative overflow-hidden bg-gray-100 dark:bg-gray-950"
      tabIndex={0}
      onKeyDown={handleKeyDown}
      onPointerMove={handleSvgPointerMove}
      onPointerUp={handlePointerUp}
    >
      {/* SVG layer: grid + edges */}
      <svg
        ref={svgRef}
        class="w-full h-full absolute inset-0"
        style={{ "min-height": "500px", cursor: panning() ? 'grabbing' : 'grab' }}
        onClick={handleCanvasClick}
        onPointerDown={handleCanvasPointerDown}
        onPointerMove={handleSvgPointerMove}
        onWheel={handleWheel}
      >
        <defs>
          <marker id="arrowhead" markerWidth="10" markerHeight="7" refX="10" refY="3.5" orient="auto">
            <polygon points="0 0, 10 3.5, 0 7" fill="#94a3b8" />
          </marker>
          <marker id="arrowhead-active" markerWidth="10" markerHeight="7" refX="10" refY="3.5" orient="auto">
            <polygon points="0 0, 10 3.5, 0 7" fill="#3b82f6" />
          </marker>
          <pattern id="grid" width="20" height="20" patternUnits="userSpaceOnUse" patternTransform={`translate(${pan().x},${pan().y}) scale(${zoom()})`}>
            <path d="M 20 0 L 0 0 0 20" fill="none" stroke="#e5e7eb" stroke-width="0.5" />
          </pattern>
        </defs>

        <rect width="100%" height="100%" fill="url(#grid)" />

        {/* Transform group for pan/zoom (edges only) */}
        <g transform={`translate(${pan().x},${pan().y}) scale(${zoom()})`}>
          <For each={props.edges}>
            {(edge) => {
              const testInfo = () => getEdgeTestInfo(edge);
              return (
                <DagEdgeComponent
                  edge={edge}
                  nodes={props.nodes}
                  active={!!testInfo()}
                  rowCount={testInfo()?.rowCount}
                  selected={selectedEdgeId() === edge.id}
                  onSelect={(id) => { setSelectedEdgeId(id); props.onSelectNode(null); }}
                  onDelete={(id) => { props.onRemoveEdge(id); setSelectedEdgeId(null); }}
                />
              );
            }}
          </For>

          {/* Edge drawing in progress */}
          <Show when={edgeDrawing()}>
            <path
              d={drawingPath()}
              fill="none"
              stroke="#3b82f6"
              stroke-width="2"
              stroke-dasharray="6 3"
              pointer-events="none"
            />
          </Show>
        </g>
      </svg>

      {/* HTML overlay layer: nodes (scales properly with CSS transform) */}
      <div
        class="absolute inset-0"
        style={{
          "pointer-events": "none",
          "transform-origin": "0 0",
          transform: `translate(${pan().x}px, ${pan().y}px) scale(${zoom()})`,
        }}
      >
        <For each={props.nodes}>
          {(node) => {
            const result = () => props.testResults?.get(node.label);
            return (
              <DagNodeComponent
                node={node}
                selected={props.selectedNodeId === node.id}
                onSelect={(id) => { props.onSelectNode(id); setSelectedEdgeId(null); }}
                onMove={props.onMoveNode}
                onPortDragStart={handlePortDragStart}
                onPortDragEnd={() => {}}
                onContextMenu={props.onContextMenu}
                testState={nodeTestState(node.id)}
                testRowCount={result()?.rowCount}
                testDurationMs={result()?.durationMs}
                hasBreakpoint={props.breakpoints?.has(node.id)}
                onToggleBreakpoint={props.onToggleBreakpoint}
                zoom={zoom()}
              />
            );
          }}
        </For>
      </div>

      {/* Zoom controls */}
      <div class="absolute bottom-3 right-3 flex items-center gap-1 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg shadow-sm px-1 py-0.5 z-10">
        <button
          class="px-2 py-1 text-xs text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded cursor-pointer"
          onClick={() => { const z = zoom(); setZoom(Math.min(3, z * 1.2)); }}
          title="Zoom in"
        >+</button>
        <button
          class="px-2 py-1 text-xs text-gray-500 font-mono min-w-[40px] text-center hover:bg-gray-100 dark:hover:bg-gray-700 rounded cursor-pointer"
          onClick={resetView}
          title="Reset zoom"
        >{Math.round(zoom() * 100)}%</button>
        <button
          class="px-2 py-1 text-xs text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded cursor-pointer"
          onClick={() => { const z = zoom(); setZoom(Math.max(0.2, z * 0.8)); }}
          title="Zoom out"
        >-</button>
        <div class="w-px h-4 bg-gray-200 dark:bg-gray-700 mx-0.5" />
        <button
          class="px-2 py-1 text-xs text-gray-600 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-700 rounded cursor-pointer"
          onClick={fitToView}
          title="Fit to view"
        >Fit</button>
      </div>
    </div>
  );
}
