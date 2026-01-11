import { useState, useEffect } from 'react';
import { datasourceApi, DataPreview, Datasource } from '../lib/api';

interface DataPreviewPanelProps {
  datasource: Datasource;
}

export default function DataPreviewPanel({ datasource }: DataPreviewPanelProps) {
  const [preview, setPreview] = useState<DataPreview | null>(null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const loadPreview = async () => {
      setIsLoading(true);
      setError(null);
      try {
        const data = await datasourceApi.getPreview(datasource.id, 100);
        setPreview(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load data preview');
      } finally {
        setIsLoading(false);
      }
    };

    loadPreview();
  }, [datasource.id]);

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="flex items-center space-x-2 text-slate-500">
          <svg className="animate-spin h-5 w-5" viewBox="0 0 24 24">
            <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" fill="none" />
            <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z" />
          </svg>
          <span>Loading data preview...</span>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-4 bg-red-50 border border-red-200 rounded-lg">
        <div className="flex items-center space-x-2">
          <svg className="w-5 h-5 text-red-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span className="text-red-700">{error}</span>
        </div>
      </div>
    );
  }

  if (!preview || preview.rows.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-slate-500">
        <div className="text-center">
          <svg className="w-12 h-12 mx-auto mb-2 text-slate-300" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
          </svg>
          <p>No data available</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Info bar */}
      <div className="flex items-center justify-between text-sm text-slate-600 bg-slate-50 px-4 py-2 rounded-lg">
        <span>
          Showing <strong>{preview.rows.length}</strong> of <strong>{preview.total_rows.toLocaleString()}</strong> rows
        </span>
        <span className="text-slate-400">
          {preview.columns.length} columns
        </span>
      </div>

      {/* Data table */}
      <div className="overflow-auto max-h-[600px] border border-slate-200 rounded-lg">
        <table className="min-w-full divide-y divide-slate-200">
          <thead className="bg-slate-50 sticky top-0">
            <tr>
              <th className="px-3 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider bg-slate-100 border-r border-slate-200">
                #
              </th>
              {preview.columns.map((col, idx) => (
                <th
                  key={idx}
                  className="px-3 py-2 text-left text-xs font-medium text-slate-500 uppercase tracking-wider whitespace-nowrap bg-slate-50"
                >
                  {col}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-slate-100">
            {preview.rows.map((row, rowIdx) => (
              <tr key={rowIdx} className="hover:bg-slate-50">
                <td className="px-3 py-2 text-xs text-slate-400 bg-slate-50 border-r border-slate-200 font-mono">
                  {rowIdx + 1}
                </td>
                {preview.columns.map((col, colIdx) => (
                  <td
                    key={colIdx}
                    className="px-3 py-2 text-sm text-slate-700 whitespace-nowrap max-w-xs truncate"
                    title={String(row[col] ?? '')}
                  >
                    {formatCellValue(row[col])}
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {/* Footer note */}
      {preview.rows.length < preview.total_rows && (
        <p className="text-xs text-slate-400 text-center">
          Data preview is limited to first {preview.preview_max} rows for performance
        </p>
      )}
    </div>
  );
}

// Helper to format cell values for display
function formatCellValue(value: unknown): string {
  if (value === null || value === undefined) {
    return '—';
  }
  if (typeof value === 'number') {
    // Format numbers nicely
    if (Number.isInteger(value)) {
      return value.toLocaleString();
    }
    return value.toLocaleString(undefined, { maximumFractionDigits: 4 });
  }
  if (typeof value === 'boolean') {
    return value ? 'true' : 'false';
  }
  return String(value);
}
