import { useState, useEffect } from 'react';
import {
  performanceApi,
  PerformanceSummary,
  PerformanceRecord,
  PerformanceAlert,
  PerformanceBaseline,
  PerformanceThreshold,
  PerformanceEvaluation,
  MetricTimeSeries,
  Datasource,
  collectionApi,
  datasourceApi,
  Collection,
} from '../lib/api';

interface Props {
  modelId: string;
  modelType: string;
  modelMetrics?: Record<string, number>; // Training metrics from the model
}

type TabType = 'overview' | 'history' | 'alerts' | 'thresholds';

export default function PerformanceMonitorPanel({ modelId, modelType, modelMetrics }: Props) {
  const [activeTab, setActiveTab] = useState<TabType>('overview');
  const [summary, setSummary] = useState<PerformanceSummary | null>(null);
  const [baselines, setBaselines] = useState<PerformanceBaseline[]>([]);
  const [records, setRecords] = useState<PerformanceRecord[]>([]);
  const [alerts, setAlerts] = useState<PerformanceAlert[]>([]);
  const [thresholds, setThresholds] = useState<PerformanceThreshold[]>([]);
  const [evaluations, setEvaluations] = useState<PerformanceEvaluation[]>([]);
  const [selectedMetric, setSelectedMetric] = useState<string>('');
  const [timeSeries, setTimeSeries] = useState<MetricTimeSeries | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Dialog states
  const [showBaselineDialog, setShowBaselineDialog] = useState(false);
  const [showEvaluationDialog, setShowEvaluationDialog] = useState(false);
  const [showRecordDialog, setShowRecordDialog] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  // Baseline form state
  const [baselineMetrics, setBaselineMetrics] = useState<Record<string, string>>({});
  const [baselineDescription, setBaselineDescription] = useState('');

  // Evaluation form state
  const [collections, setCollections] = useState<Collection[]>([]);
  const [datasources, setDatasources] = useState<Datasource[]>([]);
  const [selectedCollectionId, setSelectedCollectionId] = useState('');
  const [selectedDatasourceId, setSelectedDatasourceId] = useState('');
  const [actualColumn, setActualColumn] = useState('');
  const [predictionColumn, setPredictionColumn] = useState('');

  // Manual record form state
  const [recordMetrics, setRecordMetrics] = useState<Record<string, string>>({});
  const [recordDatasourceId, setRecordDatasourceId] = useState('');

  useEffect(() => {
    loadData();
  }, [modelId]);

  useEffect(() => {
    if (selectedMetric) {
      loadTimeSeries(selectedMetric);
    }
  }, [selectedMetric]);

  useEffect(() => {
    // Initialize baseline metrics from model training metrics
    if (modelMetrics && Object.keys(baselineMetrics).length === 0) {
      const initialMetrics: Record<string, string> = {};
      Object.entries(modelMetrics).forEach(([key, value]) => {
        if (typeof value === 'number') {
          initialMetrics[key] = value.toString();
        }
      });
      setBaselineMetrics(initialMetrics);
    }
  }, [modelMetrics]);

  const loadData = async () => {
    setLoading(true);
    setError(null);
    try {
      const [summaryRes, baselinesRes, historyRes, alertsRes, thresholdsRes, evaluationsRes] = await Promise.all([
        performanceApi.getSummary(modelId),
        performanceApi.getBaselines(modelId),
        performanceApi.getHistory(modelId, { limit: 50 }),
        performanceApi.getAlerts(modelId),
        performanceApi.getThresholds(modelId),
        performanceApi.getEvaluations(modelId, 10),
      ]);

      // Ensure summary has initialized objects to prevent null access
      const normalizedSummary: PerformanceSummary = {
        ...summaryRes,
        baseline_metrics: summaryRes.baseline_metrics || {},
        latest_metrics: summaryRes.latest_metrics || {},
        drift_percentages: summaryRes.drift_percentages || {},
      };
      setSummary(normalizedSummary);
      setBaselines(baselinesRes.baselines || []);
      setRecords(historyRes.records || []);
      setAlerts(alertsRes.alerts || []);
      setThresholds(thresholdsRes.thresholds || []);
      setEvaluations(evaluationsRes.evaluations || []);

      // Set default selected metric
      if (baselinesRes.baselines?.length > 0 && !selectedMetric) {
        setSelectedMetric(baselinesRes.baselines[0].metric_name);
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load performance data');
    } finally {
      setLoading(false);
    }
  };

  const loadCollections = async () => {
    try {
      const cols = await collectionApi.list();
      setCollections(cols);
    } catch (err) {
      console.error('Failed to load collections:', err);
    }
  };

  const loadDatasources = async (collectionId: string) => {
    try {
      const ds = await datasourceApi.list(collectionId);
      setDatasources(ds);
    } catch (err) {
      console.error('Failed to load datasources:', err);
    }
  };

  const loadTimeSeries = async (metricName: string) => {
    try {
      const series = await performanceApi.getMetricTimeSeries(modelId, metricName, 50);
      setTimeSeries(series);
    } catch (err) {
      console.error('Failed to load time series:', err);
    }
  };

  const handleAcknowledgeAlert = async (alertId: string) => {
    try {
      await performanceApi.updateAlert(modelId, alertId, { status: 'acknowledged' });
      loadData();
    } catch (err) {
      console.error('Failed to acknowledge alert:', err);
    }
  };

  const handleResolveAlert = async (alertId: string) => {
    try {
      await performanceApi.updateAlert(modelId, alertId, { status: 'resolved' });
      loadData();
    } catch (err) {
      console.error('Failed to resolve alert:', err);
    }
  };

  const handleSetBaseline = async () => {
    setIsSubmitting(true);
    try {
      const metrics: Record<string, number> = {};
      Object.entries(baselineMetrics).forEach(([key, value]) => {
        const num = parseFloat(value);
        if (!isNaN(num)) {
          metrics[key] = num;
        }
      });

      if (Object.keys(metrics).length === 0) {
        alert('Please enter at least one metric value');
        return;
      }

      await performanceApi.createBaseline(modelId, {
        metrics,
        description: baselineDescription || `Baseline set on ${new Date().toLocaleDateString()}`,
      });

      setShowBaselineDialog(false);
      setBaselineDescription('');
      loadData();
    } catch (err) {
      console.error('Failed to set baseline:', err);
      alert(err instanceof Error ? err.message : 'Failed to set baseline');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleStartEvaluation = async () => {
    if (!selectedDatasourceId || !actualColumn) {
      alert('Please select a datasource and specify the actual target column');
      return;
    }

    setIsSubmitting(true);
    try {
      await performanceApi.startEvaluation(modelId, {
        datasource_id: selectedDatasourceId,
        actual_column: actualColumn,
        prediction_column: predictionColumn || undefined,
      });

      setShowEvaluationDialog(false);
      setSelectedCollectionId('');
      setSelectedDatasourceId('');
      setActualColumn('');
      setPredictionColumn('');
      loadData();
      alert('Evaluation started! Check the history tab for results.');
    } catch (err) {
      console.error('Failed to start evaluation:', err);
      alert(err instanceof Error ? err.message : 'Failed to start evaluation');
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleRecordMetrics = async () => {
    if (!recordDatasourceId) {
      alert('Please select a datasource');
      return;
    }

    setIsSubmitting(true);
    try {
      const metrics: Record<string, number> = {};
      Object.entries(recordMetrics).forEach(([key, value]) => {
        const num = parseFloat(value);
        if (!isNaN(num)) {
          metrics[key] = num;
        }
      });

      if (Object.keys(metrics).length === 0) {
        alert('Please enter at least one metric value');
        return;
      }

      await performanceApi.recordPerformance(modelId, {
        datasource_id: recordDatasourceId,
        metrics,
      });

      setShowRecordDialog(false);
      setRecordMetrics({});
      setRecordDatasourceId('');
      loadData();
    } catch (err) {
      console.error('Failed to record metrics:', err);
      alert(err instanceof Error ? err.message : 'Failed to record metrics');
    } finally {
      setIsSubmitting(false);
    }
  };

  const openEvaluationDialog = () => {
    loadCollections();
    setShowEvaluationDialog(true);
  };

  const openRecordDialog = () => {
    loadCollections();
    // Initialize with baseline metric names
    const initialMetrics: Record<string, string> = {};
    baselines.forEach(b => {
      initialMetrics[b.metric_name] = '';
    });
    setRecordMetrics(initialMetrics);
    setShowRecordDialog(true);
  };

  const getHealthStatusColor = (status: string) => {
    switch (status) {
      case 'healthy':
        return 'text-emerald-600 bg-emerald-50';
      case 'warning':
        return 'text-amber-600 bg-amber-50';
      case 'critical':
        return 'text-red-600 bg-red-50';
      default:
        return 'text-slate-600 bg-slate-50';
    }
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'info':
        return 'text-blue-600 bg-blue-50 border-blue-200';
      case 'warning':
        return 'text-amber-600 bg-amber-50 border-amber-200';
      case 'critical':
        return 'text-red-600 bg-red-50 border-red-200';
      default:
        return 'text-slate-600 bg-slate-50 border-slate-200';
    }
  };

  const formatDrift = (drift: number | undefined) => {
    if (drift === undefined || drift === null) return '—';
    const sign = drift >= 0 ? '+' : '';
    return `${sign}${drift.toFixed(2)}%`;
  };

  const getDriftColor = (drift: number | undefined) => {
    if (drift === undefined || drift === null) return 'text-slate-500';
    if (Math.abs(drift) < 5) return 'text-emerald-600';
    if (Math.abs(drift) < 15) return 'text-amber-600';
    return 'text-red-600';
  };

  const getDefaultMetrics = () => {
    if (modelType === 'classification') {
      return ['accuracy', 'precision', 'recall', 'f1_score'];
    }
    return ['mae', 'mse', 'rmse', 'r2'];
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        <span className="ml-3 text-slate-600">Loading performance data...</span>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 bg-red-50 border border-red-200 rounded-lg">
        <p className="text-red-700">{error}</p>
        <button
          onClick={loadData}
          className="mt-2 text-sm text-red-600 hover:text-red-800 underline"
        >
          Retry
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* Header with Health Status and Actions */}
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-lg font-semibold text-slate-800">Performance Monitor</h3>
          <p className="text-sm text-slate-500">
            Task Type: <span className="font-medium capitalize">{modelType}</span>
          </p>
        </div>
        <div className="flex items-center space-x-3">
          {summary && (
            <div className={`px-4 py-2 rounded-full font-medium ${getHealthStatusColor(summary.overall_health_status)}`}>
              {summary.overall_health_status === 'healthy' && '✓ '}
              {summary.overall_health_status === 'warning' && '⚠ '}
              {summary.overall_health_status === 'critical' && '⚠ '}
              {summary.overall_health_status.charAt(0).toUpperCase() + summary.overall_health_status.slice(1)}
            </div>
          )}
        </div>
      </div>

      {/* Action Buttons */}
      <div className="flex flex-wrap gap-3 p-4 bg-slate-50 rounded-lg border border-slate-200">
        <button
          onClick={() => setShowBaselineDialog(true)}
          className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors flex items-center space-x-2"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
          </svg>
          <span>{baselines.length > 0 ? 'Update Baseline' : 'Set Baseline'}</span>
        </button>

        <button
          onClick={openEvaluationDialog}
          disabled={baselines.length === 0}
          className={`px-4 py-2 text-sm font-medium rounded-lg transition-colors flex items-center space-x-2 ${
            baselines.length > 0
              ? 'text-white bg-emerald-600 hover:bg-emerald-700'
              : 'text-slate-400 bg-slate-200 cursor-not-allowed'
          }`}
          title={baselines.length === 0 ? 'Set a baseline first' : 'Run performance evaluation'}
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
          </svg>
          <span>Run Evaluation</span>
        </button>

        <button
          onClick={openRecordDialog}
          disabled={baselines.length === 0}
          className={`px-4 py-2 text-sm font-medium rounded-lg transition-colors flex items-center space-x-2 ${
            baselines.length > 0
              ? 'text-slate-700 bg-white border border-slate-300 hover:bg-slate-50'
              : 'text-slate-400 bg-slate-100 border border-slate-200 cursor-not-allowed'
          }`}
          title={baselines.length === 0 ? 'Set a baseline first' : 'Manually record metrics'}
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
          </svg>
          <span>Record Metrics</span>
        </button>

        <button
          onClick={loadData}
          className="px-4 py-2 text-sm font-medium text-slate-700 bg-white border border-slate-300 rounded-lg hover:bg-slate-50 transition-colors flex items-center space-x-2"
        >
          <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
          <span>Refresh</span>
        </button>
      </div>

      {/* Quick Stats */}
      {summary && (
        <div className="grid grid-cols-4 gap-4">
          <div className="bg-white border border-slate-200 rounded-lg p-4">
            <div className="text-2xl font-bold text-slate-800">{summary.record_count}</div>
            <div className="text-sm text-slate-500">Performance Records</div>
          </div>
          <div className="bg-white border border-slate-200 rounded-lg p-4">
            <div className={`text-2xl font-bold ${summary.active_alerts > 0 ? 'text-red-600' : 'text-emerald-600'}`}>
              {summary.active_alerts}
            </div>
            <div className="text-sm text-slate-500">Active Alerts</div>
          </div>
          <div className="bg-white border border-slate-200 rounded-lg p-4">
            <div className={`text-2xl font-bold ${summary.warning_alerts > 0 ? 'text-amber-600' : 'text-slate-400'}`}>
              {summary.warning_alerts}
            </div>
            <div className="text-sm text-slate-500">Warnings</div>
          </div>
          <div className="bg-white border border-slate-200 rounded-lg p-4">
            <div className={`text-2xl font-bold ${summary.critical_alerts > 0 ? 'text-red-600' : 'text-slate-400'}`}>
              {summary.critical_alerts}
            </div>
            <div className="text-sm text-slate-500">Critical</div>
          </div>
        </div>
      )}

      {/* Tabs */}
      <div className="border-b border-slate-200">
        <nav className="flex space-x-8">
          {(['overview', 'history', 'alerts', 'thresholds'] as TabType[]).map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={`py-3 px-1 border-b-2 font-medium text-sm transition-colors ${
                activeTab === tab
                  ? 'border-blue-600 text-blue-600'
                  : 'border-transparent text-slate-500 hover:text-slate-700'
              }`}
            >
              {tab.charAt(0).toUpperCase() + tab.slice(1)}
              {tab === 'alerts' && alerts.filter(a => a.status === 'active').length > 0 && (
                <span className="ml-2 px-2 py-0.5 text-xs bg-red-100 text-red-600 rounded-full">
                  {alerts.filter(a => a.status === 'active').length}
                </span>
              )}
            </button>
          ))}
        </nav>
      </div>

      {/* Tab Content */}
      <div className="min-h-[300px]">
        {activeTab === 'overview' && (
          <div className="space-y-6">
            {/* Baseline vs Current Metrics */}
            {(baselines.length > 0 || (summary && summary.has_baseline && summary.baseline_metrics && Object.keys(summary.baseline_metrics).length > 0)) ? (
              <div className="bg-white border border-slate-200 rounded-lg overflow-hidden">
                <div className="px-4 py-3 bg-slate-50 border-b border-slate-200">
                  <h4 className="font-medium text-slate-700">Baseline vs Current Metrics</h4>
                </div>
                <div className="overflow-x-auto">
                  <table className="w-full">
                    <thead>
                      <tr className="bg-slate-50">
                        <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                          Metric
                        </th>
                        <th className="px-4 py-3 text-right text-xs font-medium text-slate-500 uppercase tracking-wider">
                          Baseline
                        </th>
                        <th className="px-4 py-3 text-right text-xs font-medium text-slate-500 uppercase tracking-wider">
                          Current
                        </th>
                        <th className="px-4 py-3 text-right text-xs font-medium text-slate-500 uppercase tracking-wider">
                          Drift
                        </th>
                        <th className="px-4 py-3 text-center text-xs font-medium text-slate-500 uppercase tracking-wider">
                          Status
                        </th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-slate-200">
                      {baselines.map((baselineItem) => {
                        const metricName = baselineItem.metric_name;
                        const baseline = baselineItem.metric_value;
                        const current = summary?.latest_metrics?.[metricName];
                        const drift = summary?.drift_percentages?.[metricName];

                        return (
                          <tr
                            key={metricName}
                            className="hover:bg-slate-50 cursor-pointer"
                            onClick={() => setSelectedMetric(metricName)}
                          >
                            <td className="px-4 py-3 text-sm font-medium text-slate-700">
                              {metricName.replace(/_/g, ' ').toUpperCase()}
                            </td>
                            <td className="px-4 py-3 text-sm text-right text-slate-600">
                              {baseline?.toFixed(4) ?? '—'}
                            </td>
                            <td className="px-4 py-3 text-sm text-right font-medium text-slate-800">
                              {current?.toFixed(4) ?? '—'}
                            </td>
                            <td className={`px-4 py-3 text-sm text-right font-medium ${getDriftColor(drift)}`}>
                              {formatDrift(drift)}
                            </td>
                            <td className="px-4 py-3 text-center">
                              {drift !== undefined && Math.abs(drift) >= 10 ? (
                                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-800">
                                  Drifted
                                </span>
                              ) : drift !== undefined && Math.abs(drift) >= 5 ? (
                                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-amber-100 text-amber-800">
                                  Warning
                                </span>
                              ) : (
                                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-100 text-emerald-800">
                                  Stable
                                </span>
                              )}
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              </div>
            ) : (
              <div className="bg-gradient-to-br from-blue-50 to-indigo-50 border border-blue-200 rounded-lg p-8 text-center">
                <svg className="w-16 h-16 text-blue-500 mx-auto mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
                </svg>
                <h4 className="text-xl font-semibold text-blue-800 mb-2">Set Up Performance Monitoring</h4>
                <p className="text-sm text-blue-700 mb-6 max-w-md mx-auto">
                  Create a baseline from your model's training metrics to enable drift monitoring and alerts.
                </p>
                <button
                  onClick={() => setShowBaselineDialog(true)}
                  className="px-6 py-3 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg transition-colors inline-flex items-center space-x-2"
                >
                  <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                  </svg>
                  <span>Set Baseline Now</span>
                </button>
              </div>
            )}

            {/* Recent Evaluations */}
            {evaluations.length > 0 && (
              <div className="bg-white border border-slate-200 rounded-lg overflow-hidden">
                <div className="px-4 py-3 bg-slate-50 border-b border-slate-200">
                  <h4 className="font-medium text-slate-700">Recent Evaluations</h4>
                </div>
                <div className="divide-y divide-slate-200">
                  {evaluations.slice(0, 5).map((evaluation) => (
                    <div key={evaluation.id} className="px-4 py-3 flex items-center justify-between">
                      <div className="flex items-center space-x-3">
                        <div className={`w-2 h-2 rounded-full ${
                          evaluation.status === 'completed' ? 'bg-emerald-500' :
                          evaluation.status === 'failed' ? 'bg-red-500' :
                          evaluation.status === 'running' ? 'bg-blue-500 animate-pulse' :
                          'bg-slate-400'
                        }`} />
                        <div>
                          <div className="text-sm font-medium text-slate-700">
                            {evaluation.status.charAt(0).toUpperCase() + evaluation.status.slice(1)}
                          </div>
                          <div className="text-xs text-slate-500">
                            {new Date(evaluation.created_at).toLocaleString()}
                          </div>
                        </div>
                      </div>
                      {evaluation.sample_count > 0 && (
                        <div className="text-sm text-slate-500">
                          {evaluation.sample_count.toLocaleString()} samples
                        </div>
                      )}
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Time Series Chart Preview */}
            {timeSeries && timeSeries.data_points.length > 0 && (
              <div className="bg-white border border-slate-200 rounded-lg overflow-hidden">
                <div className="px-4 py-3 bg-slate-50 border-b border-slate-200 flex items-center justify-between">
                  <h4 className="font-medium text-slate-700">
                    {selectedMetric.replace(/_/g, ' ').toUpperCase()} Trend
                  </h4>
                  <select
                    value={selectedMetric}
                    onChange={(e) => setSelectedMetric(e.target.value)}
                    className="text-sm border border-slate-300 rounded px-2 py-1"
                  >
                    {baselines.map((b) => (
                      <option key={b.metric_name} value={b.metric_name}>
                        {b.metric_name.replace(/_/g, ' ').toUpperCase()}
                      </option>
                    ))}
                  </select>
                </div>
                <div className="p-4">
                  <div className="h-48 flex items-end justify-between space-x-1">
                    {timeSeries.data_points.slice(-20).map((point, idx) => {
                      const maxValue = Math.max(...timeSeries.data_points.map(p => p.value));
                      const minValue = Math.min(...timeSeries.data_points.map(p => p.value));
                      const range = maxValue - minValue || 1;
                      const height = ((point.value - minValue) / range) * 100;

                      return (
                        <div
                          key={idx}
                          className="flex-1 bg-blue-500 rounded-t hover:bg-blue-600 transition-colors"
                          style={{ height: `${Math.max(height, 5)}%` }}
                          title={`${new Date(point.timestamp).toLocaleDateString()}: ${point.value.toFixed(4)}`}
                        />
                      );
                    })}
                  </div>
                  {timeSeries.baseline && (
                    <div className="mt-2 text-sm text-slate-500 flex items-center justify-center">
                      <div className="w-4 h-0.5 bg-red-400 mr-2" />
                      Baseline: {timeSeries.baseline.toFixed(4)}
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        )}

        {activeTab === 'history' && (
          <div className="bg-white border border-slate-200 rounded-lg overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="bg-slate-50">
                    <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                      Date
                    </th>
                    <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                      Metric
                    </th>
                    <th className="px-4 py-3 text-right text-xs font-medium text-slate-500 uppercase tracking-wider">
                      Value
                    </th>
                    <th className="px-4 py-3 text-right text-xs font-medium text-slate-500 uppercase tracking-wider">
                      Baseline
                    </th>
                    <th className="px-4 py-3 text-right text-xs font-medium text-slate-500 uppercase tracking-wider">
                      Drift
                    </th>
                    <th className="px-4 py-3 text-right text-xs font-medium text-slate-500 uppercase tracking-wider">
                      Samples
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200">
                  {records.length === 0 ? (
                    <tr>
                      <td colSpan={6} className="px-4 py-8 text-center text-slate-500">
                        No performance records yet. Run an evaluation or record metrics manually.
                      </td>
                    </tr>
                  ) : (
                    records.map((record) => (
                      <tr key={record.id} className="hover:bg-slate-50">
                        <td className="px-4 py-3 text-sm text-slate-600">
                          {new Date(record.window_end).toLocaleString()}
                        </td>
                        <td className="px-4 py-3 text-sm font-medium text-slate-700">
                          {record.metric_name.replace(/_/g, ' ').toUpperCase()}
                        </td>
                        <td className="px-4 py-3 text-sm text-right text-slate-800 font-mono">
                          {record.metric_value.toFixed(4)}
                        </td>
                        <td className="px-4 py-3 text-sm text-right text-slate-500 font-mono">
                          {record.baseline_value?.toFixed(4) ?? '—'}
                        </td>
                        <td className={`px-4 py-3 text-sm text-right font-medium ${getDriftColor(record.drift_percentage)}`}>
                          {formatDrift(record.drift_percentage)}
                        </td>
                        <td className="px-4 py-3 text-sm text-right text-slate-500">
                          {record.sample_count.toLocaleString()}
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}

        {activeTab === 'alerts' && (
          <div className="space-y-4">
            {alerts.length === 0 ? (
              <div className="bg-emerald-50 border border-emerald-200 rounded-lg p-6 text-center">
                <svg className="w-12 h-12 text-emerald-500 mx-auto mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                <h4 className="text-lg font-medium text-emerald-800">No Alerts</h4>
                <p className="text-sm text-emerald-700">All metrics are within acceptable thresholds.</p>
              </div>
            ) : (
              alerts.map((alert) => (
                <div
                  key={alert.id}
                  className={`border rounded-lg p-4 ${getSeverityColor(alert.severity)}`}
                >
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center space-x-2 mb-1">
                        <span className={`px-2 py-0.5 text-xs font-medium rounded-full ${
                          alert.severity === 'critical' ? 'bg-red-200 text-red-800' :
                          alert.severity === 'warning' ? 'bg-amber-200 text-amber-800' :
                          'bg-blue-200 text-blue-800'
                        }`}>
                          {alert.severity.toUpperCase()}
                        </span>
                        <span className={`px-2 py-0.5 text-xs font-medium rounded-full ${
                          alert.status === 'active' ? 'bg-red-100 text-red-700' :
                          alert.status === 'acknowledged' ? 'bg-amber-100 text-amber-700' :
                          'bg-slate-100 text-slate-700'
                        }`}>
                          {alert.status}
                        </span>
                      </div>
                      <h4 className="font-medium text-slate-800 mb-1">{alert.message}</h4>
                      <p className="text-sm text-slate-600">
                        <span className="font-medium">{alert.metric_name}</span>:{' '}
                        {alert.baseline_value.toFixed(4)} → {alert.current_value.toFixed(4)}{' '}
                        <span className={getDriftColor(alert.drift_percentage)}>
                          ({formatDrift(alert.drift_percentage)})
                        </span>
                      </p>
                      <p className="text-xs text-slate-500 mt-1">
                        {new Date(alert.created_at).toLocaleString()}
                      </p>
                    </div>
                    {alert.status === 'active' && (
                      <div className="flex space-x-2 ml-4">
                        <button
                          onClick={() => handleAcknowledgeAlert(alert.id)}
                          className="px-3 py-1 text-xs font-medium text-amber-700 bg-amber-100 rounded hover:bg-amber-200"
                        >
                          Acknowledge
                        </button>
                        <button
                          onClick={() => handleResolveAlert(alert.id)}
                          className="px-3 py-1 text-xs font-medium text-emerald-700 bg-emerald-100 rounded hover:bg-emerald-200"
                        >
                          Resolve
                        </button>
                      </div>
                    )}
                    {alert.status === 'acknowledged' && (
                      <button
                        onClick={() => handleResolveAlert(alert.id)}
                        className="px-3 py-1 text-xs font-medium text-emerald-700 bg-emerald-100 rounded hover:bg-emerald-200"
                      >
                        Resolve
                      </button>
                    )}
                  </div>
                </div>
              ))
            )}
          </div>
        )}

        {activeTab === 'thresholds' && (
          <div className="bg-white border border-slate-200 rounded-lg overflow-hidden">
            <div className="overflow-x-auto">
              <table className="w-full">
                <thead>
                  <tr className="bg-slate-50">
                    <th className="px-4 py-3 text-left text-xs font-medium text-slate-500 uppercase tracking-wider">
                      Metric
                    </th>
                    <th className="px-4 py-3 text-center text-xs font-medium text-slate-500 uppercase tracking-wider">
                      Warning
                    </th>
                    <th className="px-4 py-3 text-center text-xs font-medium text-slate-500 uppercase tracking-wider">
                      Critical
                    </th>
                    <th className="px-4 py-3 text-center text-xs font-medium text-slate-500 uppercase tracking-wider">
                      Direction
                    </th>
                    <th className="px-4 py-3 text-center text-xs font-medium text-slate-500 uppercase tracking-wider">
                      Status
                    </th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-200">
                  {thresholds.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="px-4 py-8 text-center text-slate-500">
                        No thresholds configured. They will be created automatically when you set a baseline.
                      </td>
                    </tr>
                  ) : (
                    thresholds.map((threshold) => (
                      <tr key={threshold.id} className="hover:bg-slate-50">
                        <td className="px-4 py-3 text-sm font-medium text-slate-700">
                          {threshold.metric_name.replace(/_/g, ' ').toUpperCase()}
                        </td>
                        <td className="px-4 py-3 text-sm text-center">
                          <span className="px-2 py-1 bg-amber-100 text-amber-700 rounded">
                            {threshold.warning_threshold}%
                          </span>
                        </td>
                        <td className="px-4 py-3 text-sm text-center">
                          <span className="px-2 py-1 bg-red-100 text-red-700 rounded">
                            {threshold.critical_threshold}%
                          </span>
                        </td>
                        <td className="px-4 py-3 text-sm text-center text-slate-600">
                          {threshold.direction === 'lower' ? '↓ Lower is bad' : '↑ Higher is bad'}
                        </td>
                        <td className="px-4 py-3 text-sm text-center">
                          {threshold.enabled ? (
                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-emerald-100 text-emerald-800">
                              Active
                            </span>
                          ) : (
                            <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-slate-100 text-slate-600">
                              Disabled
                            </span>
                          )}
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>

      {/* Set Baseline Dialog */}
      {showBaselineDialog && (
        <div className="fixed inset-0 z-50 overflow-y-auto">
          <div className="flex min-h-screen items-center justify-center px-4 pt-4 pb-20">
            <div className="fixed inset-0 bg-slate-900/60" onClick={() => setShowBaselineDialog(false)} />
            <div className="relative bg-white rounded-xl shadow-2xl max-w-lg w-full p-6">
              <h3 className="text-lg font-semibold text-slate-800 mb-4">
                {summary?.has_baseline ? 'Update Baseline Metrics' : 'Set Baseline Metrics'}
              </h3>
              <p className="text-sm text-slate-600 mb-4">
                {modelMetrics && Object.keys(modelMetrics).length > 0
                  ? 'Your model training metrics are pre-filled below. Adjust as needed.'
                  : `Enter the baseline metrics for your ${modelType} model.`}
              </p>

              <div className="space-y-4 max-h-80 overflow-y-auto">
                {(modelMetrics && Object.keys(modelMetrics).length > 0
                  ? Object.keys(modelMetrics)
                  : getDefaultMetrics()
                ).map((metric) => (
                  <div key={metric} className="flex items-center space-x-3">
                    <label className="w-28 text-sm font-medium text-slate-700 capitalize">
                      {metric.replace(/_/g, ' ')}
                    </label>
                    <input
                      type="number"
                      step="0.0001"
                      value={baselineMetrics[metric] || ''}
                      onChange={(e) => setBaselineMetrics({ ...baselineMetrics, [metric]: e.target.value })}
                      className="flex-1 px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                      placeholder="0.0000"
                    />
                  </div>
                ))}
              </div>

              <div className="mt-4">
                <label className="block text-sm font-medium text-slate-700 mb-1">
                  Description (optional)
                </label>
                <input
                  type="text"
                  value={baselineDescription}
                  onChange={(e) => setBaselineDescription(e.target.value)}
                  className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="e.g., Baseline from initial deployment"
                />
              </div>

              <div className="mt-6 flex justify-end space-x-3">
                <button
                  onClick={() => setShowBaselineDialog(false)}
                  className="px-4 py-2 text-sm font-medium text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-lg"
                >
                  Cancel
                </button>
                <button
                  onClick={handleSetBaseline}
                  disabled={isSubmitting}
                  className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg disabled:opacity-50"
                >
                  {isSubmitting ? 'Saving...' : 'Save Baseline'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Run Evaluation Dialog */}
      {showEvaluationDialog && (
        <div className="fixed inset-0 z-50 overflow-y-auto">
          <div className="flex min-h-screen items-center justify-center px-4 pt-4 pb-20">
            <div className="fixed inset-0 bg-slate-900/60" onClick={() => setShowEvaluationDialog(false)} />
            <div className="relative bg-white rounded-xl shadow-2xl max-w-lg w-full p-6">
              <h3 className="text-lg font-semibold text-slate-800 mb-4">Run Performance Evaluation</h3>
              <p className="text-sm text-slate-600 mb-4">
                Select a datasource with actual target values to evaluate model performance.
              </p>

              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">Collection</label>
                  <select
                    value={selectedCollectionId}
                    onChange={(e) => {
                      setSelectedCollectionId(e.target.value);
                      setSelectedDatasourceId('');
                      if (e.target.value) {
                        loadDatasources(e.target.value);
                      }
                    }}
                    className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="">Select collection...</option>
                    {collections.map((col) => (
                      <option key={col.id} value={col.id}>{col.name}</option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">Datasource</label>
                  <select
                    value={selectedDatasourceId}
                    onChange={(e) => setSelectedDatasourceId(e.target.value)}
                    disabled={!selectedCollectionId}
                    className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-slate-100"
                  >
                    <option value="">Select datasource...</option>
                    {datasources.map((ds) => (
                      <option key={ds.id} value={ds.id}>{ds.name}</option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Actual Target Column <span className="text-red-500">*</span>
                  </label>
                  <input
                    type="text"
                    value={actualColumn}
                    onChange={(e) => setActualColumn(e.target.value)}
                    className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="e.g., actual_target, y_true"
                  />
                  <p className="text-xs text-slate-500 mt-1">
                    The column containing actual/true values in your evaluation data
                  </p>
                </div>

                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Prediction Column (optional)
                  </label>
                  <input
                    type="text"
                    value={predictionColumn}
                    onChange={(e) => setPredictionColumn(e.target.value)}
                    className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                    placeholder="e.g., prediction, y_pred"
                  />
                  <p className="text-xs text-slate-500 mt-1">
                    Leave empty to generate predictions using the model
                  </p>
                </div>
              </div>

              <div className="mt-6 flex justify-end space-x-3">
                <button
                  onClick={() => setShowEvaluationDialog(false)}
                  className="px-4 py-2 text-sm font-medium text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-lg"
                >
                  Cancel
                </button>
                <button
                  onClick={handleStartEvaluation}
                  disabled={isSubmitting || !selectedDatasourceId || !actualColumn}
                  className="px-4 py-2 text-sm font-medium text-white bg-emerald-600 hover:bg-emerald-700 rounded-lg disabled:opacity-50"
                >
                  {isSubmitting ? 'Starting...' : 'Start Evaluation'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Record Metrics Dialog */}
      {showRecordDialog && (
        <div className="fixed inset-0 z-50 overflow-y-auto">
          <div className="flex min-h-screen items-center justify-center px-4 pt-4 pb-20">
            <div className="fixed inset-0 bg-slate-900/60" onClick={() => setShowRecordDialog(false)} />
            <div className="relative bg-white rounded-xl shadow-2xl max-w-lg w-full p-6">
              <h3 className="text-lg font-semibold text-slate-800 mb-4">Record Performance Metrics</h3>
              <p className="text-sm text-slate-600 mb-4">
                Manually record performance metrics from an external evaluation.
              </p>

              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">Collection</label>
                  <select
                    value={selectedCollectionId}
                    onChange={(e) => {
                      setSelectedCollectionId(e.target.value);
                      setRecordDatasourceId('');
                      if (e.target.value) {
                        loadDatasources(e.target.value);
                      }
                    }}
                    className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                  >
                    <option value="">Select collection...</option>
                    {collections.map((col) => (
                      <option key={col.id} value={col.id}>{col.name}</option>
                    ))}
                  </select>
                </div>

                <div>
                  <label className="block text-sm font-medium text-slate-700 mb-1">
                    Evaluation Datasource <span className="text-red-500">*</span>
                  </label>
                  <select
                    value={recordDatasourceId}
                    onChange={(e) => setRecordDatasourceId(e.target.value)}
                    disabled={!selectedCollectionId}
                    className="w-full px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:bg-slate-100"
                  >
                    <option value="">Select datasource...</option>
                    {datasources.map((ds) => (
                      <option key={ds.id} value={ds.id}>{ds.name}</option>
                    ))}
                  </select>
                </div>

                <div className="border-t pt-4">
                  <label className="block text-sm font-medium text-slate-700 mb-2">Metrics</label>
                  <div className="space-y-3">
                    {Object.keys(recordMetrics).map((metric) => (
                      <div key={metric} className="flex items-center space-x-3">
                        <label className="w-28 text-sm text-slate-600 capitalize">
                          {metric.replace(/_/g, ' ')}
                        </label>
                        <input
                          type="number"
                          step="0.0001"
                          value={recordMetrics[metric]}
                          onChange={(e) => setRecordMetrics({ ...recordMetrics, [metric]: e.target.value })}
                          className="flex-1 px-3 py-2 border border-slate-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
                          placeholder="0.0000"
                        />
                      </div>
                    ))}
                  </div>
                </div>
              </div>

              <div className="mt-6 flex justify-end space-x-3">
                <button
                  onClick={() => setShowRecordDialog(false)}
                  className="px-4 py-2 text-sm font-medium text-slate-700 bg-slate-100 hover:bg-slate-200 rounded-lg"
                >
                  Cancel
                </button>
                <button
                  onClick={handleRecordMetrics}
                  disabled={isSubmitting || !recordDatasourceId}
                  className="px-4 py-2 text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 rounded-lg disabled:opacity-50"
                >
                  {isSubmitting ? 'Recording...' : 'Record Metrics'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
