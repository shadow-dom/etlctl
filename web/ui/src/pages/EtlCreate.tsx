import { createSignal } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import { etlApi } from '../api/etl';
import Header from '../components/layout/Header';
import Button from '../components/common/Button';

export default function EtlCreate() {
  const navigate = useNavigate();
  const [name, setName] = createSignal('');
  const [error, setError] = createSignal('');

  async function create() {
    const n = name().trim();
    if (!n) {
      setError('Name is required');
      return;
    }
    if (!/^[a-z0-9-]+$/.test(n)) {
      setError('Name must be lowercase alphanumeric with hyphens');
      return;
    }

    try {
      await etlApi.create({
        name: n,
        sources: [],
        targets: [],
        pipelines: [],
        queries: [],
      });
      navigate(`/etl/${n}`);
    } catch (e: any) {
      setError(e.message);
    }
  }

  return (
    <div class="flex flex-col h-full">
      <Header title="New ETL Pipeline" />
      <div class="flex-1 flex items-center justify-center">
        <div class="w-96 p-6 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg">
          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
                ETL Name
              </label>
              <input
                type="text"
                value={name()}
                onInput={(e) => { setName(e.currentTarget.value); setError(''); }}
                onKeyDown={(e) => e.key === 'Enter' && create()}
                class="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-gray-900 dark:text-white"
                placeholder="my-etl-pipeline"
                autofocus
              />
              {error() && <p class="mt-1 text-sm text-red-600">{error()}</p>}
            </div>
            <Button onClick={create} class="w-full">Create</Button>
          </div>
        </div>
      </div>
    </div>
  );
}
