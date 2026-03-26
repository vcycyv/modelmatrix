import { useState, useEffect, useCallback } from 'react';
import { performanceApi, PerformanceThresholdDefault } from '../lib/api';

const TASK_TYPES = ['classification', 'regression'] as const;
type TaskType = (typeof TASK_TYPES)[number];

export default function GlobalThresholdDefaultsPanel() {
  const [activeTaskType, setActiveTaskType] = useState<TaskType>('classification');
  const [defaults, setDefaults] = useState<PerformanceThresholdDefault[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [editingMetric, setEditingMetric] = useState<string | null>(null);
  const [editWarning, setEditWarning] = useState('');
  const [editCritical, setEditCritical] = useState('');
  const [saving, setSaving] = useState(false);

  const load = useCallback(async (taskType: TaskType) => {
    setLoading(true);
    setError(null);
    try {
      const res = await performanceApi.getThresholdDefaults(taskType);
      setDefaults(res.defaults || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load defaults');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load(activeTaskType);
  }, [activeTaskType, load]);

  const startEdit = (d: PerformanceThresholdDefault) => {
    setEditingMetric(d.metric_name);
    setEditWarning(String(d.warning_threshold));
    setEditCritical(String(d.critical_threshold));
  };

  const cancelEdit = () => {
    setEditingMetric(null);
    setEditWarning('');
    setEditCritical('');
  };

  const saveEdit = async (metricName: string) => {
    const w = parseFloat(editWarning);
    const c = parseFloat(editCritical);
    if (Number.isNaN(w) || Number.isNaN(c) || w <= 0 || c <= 0) {
      alert('Thresholds must be positive numbers.');
      return;
    }
    if (w > c) {
      alert('Warning must be ≤ critical.');
      return;
    }
    setSaving(true);
    try {
      const updated = await performanceApi.upsertThresholdDefault({
        task_type: activeTaskType,
        metric_name: metricName,
        warning_threshold: w,
        critical_threshold: c,
      });
      setDefaults((prev) =>
        prev.map((d) => (d.metric_name === metricName ? { ...d, ...updated } : d))
      );
      cancelEdit();
    } catch (err) {
      alert(err instanceof Error ? err.message : 'Failed to save');
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="space-y-4">
      <div>
        <h3 className="text-base font-semibold text-slate-800">Default Alert Thresholds</h3>
        <p className="mt-1 text-sm text-slate-500">
          These values are used when a new model baseline is created. Existing per-model thresholds are
          not affected by changes here.
        </p>
      </div>

      {/* Task type tabs */}
      <div className="flex space-x-1 bg-slate-100 p-1 rounded-lg w-fit">
        {TASK_TYPES.map((t) => (
          <button
            key={t}
            onClick={() => { setActiveTaskType(t); cancelEdit(); }}
            className={`px-4 py-1.5 text-sm font-medium rounded-md transition-colors ${
              activeTaskType === t
                ? 'bg-white text-slate-800 shadow-sm'
                : 'text-slate-500 hover:text-slate-700'
            }`}
          >
            {t.charAt(0).toUpperCase() + t.slice(1)}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="flex items-center gap-2 py-6 text-slate-500 text-sm">
          <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600" />
          Loading…
        </div>
      ) : error ? (
        <div className="p-4 bg-red-50 border border-red-200 rounded-lg text-sm text-red-700">
          {error}
          <button onClick={() => load(activeTaskType)} className="ml-3 underline">Retry</button>
        </div>
      ) : (
        <div className="bg-white border border-slate-200 rounded-lg overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="bg-slate-50">
                <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                  Metric
                </th>
                <th className="px-4 py-3 text-center text-xs font-medium text-slate-500 uppercase tracking-wider">
                  Warning default
                </th>
                <th className="px-4 py-3 text-center text-xs font-medium text-slate-500 uppercase tracking-wider">
                  Critical default
                </th>
                <th className="px-4 py-3 text-center text-xs font-medium text-slate-500 uppercase tracking-wider">
                  Direction
                </th>
                <th className="px-4 py-3 text-right text-xs font-medium text-slate-500 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-200">
              {defaults.map((d) => {
                const isEditing = editingMetric === d.metric_name;
                return (
                  <tr key={d.metric_name} className="hover:bg-slate-50">
                    <td className="px-4 py-3 font-medium text-slate-700">
                      {d.metric_name.replace(/_/g, ' ').toUpperCase()}
                    </td>
                    <td className="px-4 py-3 text-center">
                      {isEditing ? (
                        <div className="flex items-center justify-center gap-1">
                          <input
                            type="number"
                            min={0}
                            step={0.1}
                            value={editWarning}
                            onChange={(e) => setEditWarning(e.target.value)}
                            className="w-20 px-2 py-1 border border-slate-300 rounded text-sm text-center focus:outline-none focus:ring-2 focus:ring-amber-500"
                          />
                          <span className="text-slate-400 text-xs">%</span>
                        </div>
                      ) : (
                        <span className="px-2 py-1 bg-amber-100 text-amber-700 rounded text-xs font-medium">
                          {d.warning_threshold}%
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-center">
                      {isEditing ? (
                        <div className="flex items-center justify-center gap-1">
                          <input
                            type="number"
                            min={0}
                            step={0.1}
                            value={editCritical}
                            onChange={(e) => setEditCritical(e.target.value)}
                            className="w-20 px-2 py-1 border border-slate-300 rounded text-sm text-center focus:outline-none focus:ring-2 focus:ring-red-500"
                          />
                          <span className="text-slate-400 text-xs">%</span>
                        </div>
                      ) : (
                        <span className="px-2 py-1 bg-red-100 text-red-700 rounded text-xs font-medium">
                          {d.critical_threshold}%
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-3 text-center text-slate-600">
                      {d.direction === 'lower' ? '↓ Lower is bad' : '↑ Higher is bad'}
                    </td>
                    <td className="px-4 py-3 text-right whitespace-nowrap">
                      {isEditing ? (
                        <div className="flex justify-end gap-2">
                          <button
                            onClick={cancelEdit}
                            disabled={saving}
                            className="px-2 py-1 text-xs font-medium text-slate-600 bg-slate-100 rounded hover:bg-slate-200 disabled:opacity-50"
                          >
                            Cancel
                          </button>
                          <button
                            onClick={() => saveEdit(d.metric_name)}
                            disabled={saving}
                            className="px-2 py-1 text-xs font-medium text-white bg-blue-600 rounded hover:bg-blue-700 disabled:opacity-50"
                          >
                            {saving ? 'Saving…' : 'Save'}
                          </button>
                        </div>
                      ) : (
                        <button
                          onClick={() => startEdit(d)}
                          className="px-2 py-1 text-xs font-medium text-blue-700 bg-blue-50 rounded hover:bg-blue-100"
                        >
                          Edit
                        </button>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      <p className="text-xs text-slate-400">
        Only admins can change these defaults. Changes apply to new baselines only.
      </p>
    </div>
  );
}
