import { createEffect, createSignal, Show } from 'solid-js';
import { useDagStore } from '../../stores/dagStore';
import { etlApi } from '../../api/etl';
import { parseToDag } from '../../lib/dagParser';
import { compileDag } from '../../lib/dagCompiler';
import type { ETLConfig } from '../../types/etl';
import type { NodeType, TestStepResult } from '../../types/dag';
import DagCanvas from './DagCanvas';
import DagToolbar from './DagToolbar';
import NodeConfigPanel from './NodeConfigPanel';
import ContextMenu from './ContextMenu';
import type { ContextMenuItem } from './ContextMenu';
import TestOverlay from '../test/TestOverlay';

interface DagEditorProps {
  etl: ETLConfig;
  onValidate: () => void;
  onTest: () => void;
  validationResult?: string | null;
}

export default function DagEditor(props: DagEditorProps) {
  const dag = useDagStore();

  // Context menu state
  const [contextMenu, setContextMenu] = createSignal<{
    x: number;
    y: number;
    nodeId: string;
    items: ContextMenuItem[];
  } | null>(null);

  // Test mode state
  const [testMode, setTestMode] = createSignal(false);
  const [testResults, setTestResults] = createSignal<Map<string, TestStepResult>>(new Map());
  const [breakpoints, setBreakpoints] = createSignal<Set<string>>(new Set());
  const [activeTestNode, setActiveTestNode] = createSignal<string | null>(null);
  const [testSteps, setTestSteps] = createSignal<TestStepResult[]>([]);
  const [expandedStep, setExpandedStep] = createSignal<number | null>(null);

  // Debugger state
  const [paused, setPaused] = createSignal(false);
  const [currentStepIndex, setCurrentStepIndex] = createSignal(0);
  const [allFetchedSteps, setAllFetchedSteps] = createSignal<TestStepResult[]>([]);
  const [debugMode, setDebugMode] = createSignal<'run' | 'step'>('run');
  const [testError, setTestError] = createSignal<string | null>(null);

  let continueResolve: (() => void) | null = null;
  let testStopped = false;

  // Load DAG
  createEffect(async () => {
    try {
      const saved = await etlApi.getDag(props.etl.name);
      if (saved.nodes.length > 0) {
        dag.loadDag(saved.nodes, saved.edges);
      } else {
        const parsed = parseToDag(props.etl);
        dag.loadDag(parsed.nodes, parsed.edges);
      }
    } catch {
      const parsed = parseToDag(props.etl);
      dag.loadDag(parsed.nodes, parsed.edges);
    }
  });

  function addNode(type: NodeType) {
    const xByType: Record<string, number> = {
      source: 50,
      query: 250,
      transform: 450,
      function: 550,
      target: 750,
    };
    const existingOfType = dag.state.nodes.filter(n => n.type === type).length;
    dag.addNode(type, xByType[type] || 200, 50 + existingOfType * 100);
  }

  async function save() {
    await syncDagToBackend();
    dag.clearDirty();
  }

  // Compile the current DAG and push both the visual layout and the YAML config to the backend
  async function syncDagToBackend() {
    await etlApi.saveDag(props.etl.name, {
      nodes: [...dag.state.nodes],
      edges: [...dag.state.edges],
    });
    const compiled = compileDag(props.etl.name, dag.state.nodes, dag.state.edges);
    await etlApi.update(props.etl.name, compiled);
  }

  function handleContextMenu(nodeId: string, x: number, y: number) {
    const node = dag.state.nodes.find(n => n.id === nodeId);
    if (!node) return;

    const hasBp = breakpoints().has(nodeId);
    const items: ContextMenuItem[] = [
      { label: 'Configure', icon: '\u2699', action: () => dag.selectNode(nodeId) },
      {
        label: hasBp ? 'Remove Breakpoint' : 'Set Breakpoint',
        icon: hasBp ? '\u25CB' : '\u25CF',
        action: () => toggleBreakpoint(nodeId),
      },
      { label: 'Duplicate', icon: '\u2398', action: () => {
        dag.addNode(node.type, node.x + 30, node.y + 30, `${node.label}_copy`);
      }},
      { label: '', separator: true, action: () => {} },
      { label: 'Delete', icon: '\u2715', danger: true, action: () => dag.removeNode(nodeId) },
    ];

    setContextMenu({ x, y, nodeId, items });
  }

  function toggleBreakpoint(nodeId: string) {
    setBreakpoints(prev => {
      const next = new Set(prev);
      if (next.has(nodeId)) {
        next.delete(nodeId);
      } else {
        next.add(nodeId);
      }
      return next;
    });
  }

  async function startTest(mode: 'run' | 'step', dryRun = true) {
    setTestMode(true);
    setTestResults(new Map());
    setTestSteps([]);
    setExpandedStep(null);
    setCurrentStepIndex(0);
    setDebugMode(mode);
    setPaused(false);
    setTestError(null);
    testStopped = false;

    try {
      // Sync current DAG to backend so the test reflects the visual graph
      await syncDagToBackend();
      const result = await etlApi.test(props.etl.name, dryRun, true);

      if (!result.steps || result.steps.length === 0) {
        setTestError('Test returned no steps. Check that your DAG has connected sources and targets.');
        return;
      }

      setAllFetchedSteps(result.steps);
      await walkSteps(result.steps);
    } catch (e: any) {
      console.error('Test failed:', e);
      setTestError(e.message || 'Test failed');
    }
  }

  function runTest() { startTest('run'); }
  function runTestStepByStep() { startTest('step'); }
  function runForReal() { startTest('run', false); }

  async function walkSteps(steps: TestStepResult[]) {
    const resultsMap = new Map(testResults());

    for (let i = 0; i < steps.length; i++) {
      if (testStopped) return;

      const step = steps[i];
      setCurrentStepIndex(i);

      // Find the node that matches this step
      const matchingNode = dag.state.nodes.find(n => n.label === step.node);
      if (matchingNode) {
        setActiveTestNode(matchingNode.id);
      }

      // Determine if we should pause here
      const shouldPause =
        debugMode() === 'step' ||
        (matchingNode && breakpoints().has(matchingNode.id));

      if (shouldPause) {
        // Show this step's data before pausing
        resultsMap.set(step.node, step);
        setTestResults(new Map(resultsMap));
        setTestSteps(prev => [...prev, step]);
        setExpandedStep(i);

        await waitForContinue();
        if (testStopped) return;
      }

      // Animate with a small delay
      await new Promise(r => setTimeout(r, 300));
      if (testStopped) return;

      // Commit step
      resultsMap.set(step.node, step);
      setTestResults(new Map(resultsMap));
      // Only add to visible steps if not already added during pause
      if (!shouldPause) {
        setTestSteps(prev => [...prev, step]);
      }
      setActiveTestNode(null);
    }
  }

  function waitForContinue(): Promise<void> {
    setPaused(true);
    return new Promise(resolve => {
      continueResolve = () => {
        setPaused(false);
        resolve();
      };
    });
  }

  function continueFromBreakpoint() {
    setDebugMode('run');
    if (continueResolve) {
      continueResolve();
      continueResolve = null;
    }
  }

  function stepNext() {
    // Keep in step mode and advance one step
    setDebugMode('step');
    if (continueResolve) {
      continueResolve();
      continueResolve = null;
    }
  }

  function skipStep() {
    // Skip this step: advance without pausing at next
    if (continueResolve) {
      continueResolve();
      continueResolve = null;
    }
  }

  function stopTest() {
    testStopped = true;
    setTestMode(false);
    setTestResults(new Map());
    setTestSteps([]);
    setAllFetchedSteps([]);
    setActiveTestNode(null);
    setPaused(false);
    setCurrentStepIndex(0);
    setTestError(null);
    if (continueResolve) {
      continueResolve();
      continueResolve = null;
    }
  }

  function restartTest() {
    stopTest();
    setTimeout(() => startTest('run'), 100);
  }

  const selectedNode = () =>
    dag.state.selectedNodeId
      ? dag.state.nodes.find(n => n.id === dag.state.selectedNodeId)
      : undefined;

  return (
    <div class="flex flex-col h-full">
      <DagToolbar
        onAddNode={addNode}
        onSave={save}
        onValidate={props.onValidate}
        onTest={runTest}
        onTestStep={runTestStepByStep}
        onRun={runForReal}
        isDirty={dag.state.isDirty}
        validationResult={props.validationResult}
        testMode={testMode()}
        paused={paused()}
        currentStep={currentStepIndex()}
        totalSteps={allFetchedSteps().length}
        error={testError()}
        onContinue={continueFromBreakpoint}
        onStepNext={stepNext}
        onSkip={skipStep}
        onRestart={restartTest}
        onStopTest={stopTest}
      />
      {/* Vertical split: top (canvas + config) / bottom (test overlay) */}
      <div class="flex flex-col flex-1 overflow-hidden">
        {/* Top row: canvas + config panel side by side */}
        <div class="flex flex-1 overflow-hidden relative min-h-0">
          <DagCanvas
            nodes={dag.state.nodes}
            edges={dag.state.edges}
            selectedNodeId={dag.state.selectedNodeId}
            onSelectNode={(id) => dag.selectNode(id)}
            onMoveNode={(id, x, y) => dag.moveNode(id, x, y)}
            onAddEdge={(src, tgt) => dag.addEdge(src, tgt)}
            onRemoveEdge={(id) => dag.removeEdge(id)}
            onContextMenu={handleContextMenu}
            testResults={testMode() ? testResults() : undefined}
            breakpoints={breakpoints()}
            activeTestNode={activeTestNode()}
            onToggleBreakpoint={toggleBreakpoint}
          />

          {/* Config panel (right) */}
          <Show when={selectedNode()}>
            <NodeConfigPanel
              node={selectedNode()}
              nodes={dag.state.nodes}
              edges={dag.state.edges}
              onUpdateConfig={(id, cfg) => dag.updateNodeConfig(id, cfg)}
              onUpdateLabel={(id, label) => dag.updateNodeLabel(id, label)}
              onDelete={(id) => dag.removeNode(id)}
              onClose={() => dag.selectNode(null)}
            />
          </Show>

          {/* Context menu */}
          <Show when={contextMenu()}>
            <ContextMenu
              x={contextMenu()!.x}
              y={contextMenu()!.y}
              items={contextMenu()!.items}
              onClose={() => setContextMenu(null)}
            />
          </Show>
        </div>

        {/* Test results panel (bottom) — in flow, not absolute */}
        <Show when={testMode() && testSteps().length > 0}>
          <TestOverlay
            steps={testSteps()}
            expandedStep={expandedStep()}
            onExpandStep={(i) => setExpandedStep(expandedStep() === i ? null : i)}
            paused={paused()}
            onContinue={continueFromBreakpoint}
            onStop={stopTest}
          />
        </Show>
      </div>
    </div>
  );
}
