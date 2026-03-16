import { Show } from 'solid-js';
import Button from '../common/Button';
import type { NodeType } from '../../types/dag';

interface DagToolbarProps {
  onAddNode: (type: NodeType) => void;
  onSave: () => void;
  onValidate: () => void;
  onTest: () => void;
  onTestStep: () => void;
  onRun: () => void;
  isDirty: boolean;
  validationResult?: string | null;
  testMode?: boolean;
  paused?: boolean;
  currentStep?: number;
  totalSteps?: number;
  error?: string | null;
  onContinue?: () => void;
  onStepNext?: () => void;
  onSkip?: () => void;
  onRestart?: () => void;
  onStopTest?: () => void;
}

export default function DagToolbar(props: DagToolbarProps) {
  return (
    <div class="h-12 bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700 flex items-center gap-2 px-4">
      <Show when={!props.testMode}>
        {/* Normal mode */}
        <span class="text-xs font-medium text-gray-500 uppercase mr-2">Add:</span>
        <Button variant="secondary" size="sm" onClick={() => props.onAddNode('source')}>
          Source
        </Button>
        <Button variant="secondary" size="sm" onClick={() => props.onAddNode('query')}>
          Query
        </Button>
        <Button variant="secondary" size="sm" onClick={() => props.onAddNode('transform')}>
          Transform
        </Button>
        <Button variant="secondary" size="sm" onClick={() => props.onAddNode('function')}>
          Function
        </Button>
        <Button variant="secondary" size="sm" onClick={() => props.onAddNode('target')}>
          Target
        </Button>

        <div class="flex-1" />

        <Show when={props.validationResult}>
          <span class={`text-xs px-2 py-1 rounded ${props.validationResult?.startsWith('Valid') ? 'bg-green-100 text-green-700' : 'bg-red-100 text-red-700'}`}>
            {props.validationResult}
          </span>
        </Show>
        <Button variant="ghost" size="sm" onClick={props.onValidate}>
          Validate
        </Button>
        <div class="border-l border-gray-200 dark:border-gray-700 h-6 mx-1" />
        <Button variant="ghost" size="sm" onClick={props.onTest}>
          Test
        </Button>
        <Button variant="ghost" size="sm" onClick={props.onTestStep}>
          Step
        </Button>
        <Button variant="primary" size="sm" onClick={props.onRun}>
          Run
        </Button>
        <div class="border-l border-gray-200 dark:border-gray-700 h-6 mx-1" />
        <Button variant="primary" size="sm" onClick={props.onSave} disabled={!props.isDirty}>
          Save
        </Button>
      </Show>

      <Show when={props.testMode}>
        {/* Debug mode toolbar */}
        <div class="flex items-center gap-2">
          <span class="inline-block w-2 h-2 rounded-full bg-green-500 animate-pulse" />
          <span class="text-sm font-medium text-gray-700 dark:text-gray-300">Debug</span>
        </div>

        {/* Step counter */}
        <span class="text-xs font-mono text-gray-500 bg-gray-100 dark:bg-gray-700 px-2 py-0.5 rounded">
          {(props.currentStep ?? 0) + 1}/{props.totalSteps ?? 0}
        </span>

        <Show when={props.paused}>
          <span class="text-xs bg-orange-100 dark:bg-orange-900/40 text-orange-700 dark:text-orange-300 px-2 py-0.5 rounded-full font-medium">
            Paused
          </span>
        </Show>

        <Show when={!props.paused && (props.currentStep ?? 0) < (props.totalSteps ?? 0)}>
          <span class="text-xs bg-blue-100 dark:bg-blue-900/40 text-blue-700 dark:text-blue-300 px-2 py-0.5 rounded-full font-medium">
            Running
          </span>
        </Show>

        <Show when={!props.paused && !props.error && (props.currentStep ?? 0) >= ((props.totalSteps ?? 1) - 1) && (props.totalSteps ?? 0) > 0}>
          <span class="text-xs bg-green-100 dark:bg-green-900/40 text-green-700 dark:text-green-300 px-2 py-0.5 rounded-full font-medium">
            Complete
          </span>
        </Show>

        <Show when={props.error}>
          <span class="text-xs bg-red-100 dark:bg-red-900/40 text-red-700 dark:text-red-300 px-2 py-0.5 rounded font-medium max-w-[300px] truncate" title={props.error!}>
            {props.error}
          </span>
        </Show>

        <div class="border-l border-gray-200 dark:border-gray-700 h-6 mx-1" />

        {/* Debugger buttons */}
        <div class="flex items-center gap-1">
          <Show when={props.paused}>
            <button
              class="px-2 py-1 text-xs font-medium rounded bg-green-600 text-white hover:bg-green-700 cursor-pointer flex items-center gap-1"
              onClick={props.onContinue}
              title="Continue (run to next breakpoint)"
            >
              <svg class="w-3 h-3" viewBox="0 0 12 12" fill="currentColor"><polygon points="2,0 12,6 2,12" /></svg>
              Continue
            </button>

            <button
              class="px-2 py-1 text-xs font-medium rounded bg-blue-600 text-white hover:bg-blue-700 cursor-pointer flex items-center gap-1"
              onClick={props.onStepNext}
              title="Step to next"
            >
              <svg class="w-3 h-3" viewBox="0 0 12 12" fill="currentColor">
                <polygon points="1,0 7,6 1,12" />
                <rect x="8" y="0" width="3" height="12" />
              </svg>
              Step
            </button>

            <button
              class="px-2 py-1 text-xs font-medium rounded bg-gray-500 text-white hover:bg-gray-600 cursor-pointer flex items-center gap-1"
              onClick={props.onSkip}
              title="Skip (continue without stopping)"
            >
              <svg class="w-3 h-3" viewBox="0 0 12 12" fill="currentColor">
                <polygon points="0,0 5,6 0,12" />
                <polygon points="6,0 11,6 6,12" />
              </svg>
              Skip
            </button>
          </Show>

          <button
            class="px-2 py-1 text-xs font-medium rounded bg-yellow-600 text-white hover:bg-yellow-700 cursor-pointer flex items-center gap-1"
            onClick={props.onRestart}
            title="Restart test"
          >
            <svg class="w-3 h-3" viewBox="0 0 12 12" fill="currentColor">
              <path d="M6 1a5 5 0 1 0 4.5 2.8l-1.2.7A3.5 3.5 0 1 1 6 2.5V4l3-2-3-2v1z" />
            </svg>
            Restart
          </button>
        </div>

        <div class="flex-1" />

        <Button variant="danger" size="sm" onClick={props.onStopTest}>
          Stop
        </Button>
      </Show>
    </div>
  );
}
