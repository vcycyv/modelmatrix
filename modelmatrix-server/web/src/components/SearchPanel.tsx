import { useState, useEffect, useRef, useCallback } from 'react';
import {
  searchApi,
  SearchResultItem,
  SearchResourceType,
} from '../lib/api';

interface Props {
  isOpen: boolean;
  onClose: () => void;
  /** When user clicks a result, caller can navigate to/select it */
  onSelectResult: (item: SearchResultItem) => void;
  /** Optional: scope search to this folder (and descendants) */
  scopeFolderId?: string;
  scopeFolderName?: string;
}

const TYPE_OPTIONS: { value: SearchResourceType; label: string }[] = [
  { value: 'all',     label: 'All types' },
  { value: 'build',   label: 'Builds'    },
  { value: 'model',   label: 'Models'    },
  { value: 'project', label: 'Projects'  },
  { value: 'folder',  label: 'Folders'   },
];

const TYPE_ICON: Record<string, string> = {
  build:   'bg-purple-100 text-purple-700',
  model:   'bg-emerald-100 text-emerald-700',
  project: 'bg-blue-100 text-blue-700',
  folder:  'bg-amber-100 text-amber-700',
};

const STATUS_COLOR: Record<string, string> = {
  completed: 'bg-emerald-100 text-emerald-700',
  active:    'bg-emerald-100 text-emerald-700',
  running:   'bg-blue-100 text-blue-700',
  pending:   'bg-slate-100 text-slate-600',
  failed:    'bg-red-100 text-red-700',
  cancelled: 'bg-slate-100 text-slate-600',
  draft:     'bg-slate-100 text-slate-600',
  inactive:  'bg-slate-100 text-slate-600',
  archived:  'bg-slate-100 text-slate-600',
};

export default function SearchPanel({
  isOpen,
  onClose,
  onSelectResult,
  scopeFolderId,
  scopeFolderName,
}: Props) {
  const [query, setQuery] = useState('');
  const [typeFilter, setTypeFilter] = useState<SearchResourceType>('all');
  const [useScopeFolder, setUseScopeFolder] = useState(false);
  const [results, setResults] = useState<SearchResultItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [searched, setSearched] = useState(false);

  const inputRef = useRef<HTMLInputElement>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Focus input when panel opens
  useEffect(() => {
    if (isOpen) {
      setTimeout(() => inputRef.current?.focus(), 50);
    } else {
      setQuery('');
      setResults([]);
      setSearched(false);
      setError(null);
    }
  }, [isOpen]);

  const runSearch = useCallback(async (q: string, type: SearchResourceType, useScope: boolean) => {
    if (!q.trim()) {
      setResults([]);
      setSearched(false);
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const res = await searchApi.search(
        q.trim(),
        type,
        useScope && scopeFolderId ? scopeFolderId : undefined
      );
      setResults(res.results);
      setTotal(res.total);
      setSearched(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Search failed');
    } finally {
      setLoading(false);
    }
  }, [scopeFolderId]);

  const handleQueryChange = (value: string) => {
    setQuery(value);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      runSearch(value, typeFilter, useScopeFolder);
    }, 300);
  };

  const handleTypeChange = (type: SearchResourceType) => {
    setTypeFilter(type);
    runSearch(query, type, useScopeFolder);
  };

  const handleScopeToggle = (checked: boolean) => {
    setUseScopeFolder(checked);
    runSearch(query, typeFilter, checked);
  };

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Escape') onClose();
    if (e.key === 'Enter' && query.trim()) {
      if (debounceRef.current) clearTimeout(debounceRef.current);
      runSearch(query, typeFilter, useScopeFolder);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-start justify-center pt-20 px-4">
      {/* Backdrop */}
      <div className="fixed inset-0 bg-slate-900/50" onClick={onClose} />

      {/* Panel */}
      <div className="relative bg-white rounded-xl shadow-2xl w-full max-w-2xl flex flex-col max-h-[70vh]">
        {/* Search input row */}
        <div className="flex items-center gap-3 px-4 pt-4 pb-3 border-b border-slate-200">
          <svg className="w-5 h-5 text-slate-400 flex-shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-4.35-4.35M17 11A6 6 0 115 11a6 6 0 0112 0z" />
          </svg>
          <input
            ref={inputRef}
            type="text"
            value={query}
            onChange={(e) => handleQueryChange(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Search by name or description…"
            className="flex-1 text-sm bg-transparent outline-none placeholder-slate-400 text-slate-800"
          />
          {loading && (
            <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600 flex-shrink-0" />
          )}
          <button onClick={onClose} className="text-slate-400 hover:text-slate-600 flex-shrink-0">
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Filters row */}
        <div className="flex items-center gap-3 px-4 py-2 border-b border-slate-100 bg-slate-50/80 flex-wrap">
          {/* Type filter chips */}
          <div className="flex items-center gap-1">
            {TYPE_OPTIONS.map((opt) => (
              <button
                key={opt.value}
                onClick={() => handleTypeChange(opt.value)}
                className={`px-2.5 py-1 rounded-full text-xs font-medium transition-colors ${
                  typeFilter === opt.value
                    ? 'bg-blue-600 text-white'
                    : 'bg-white text-slate-600 border border-slate-200 hover:border-slate-300'
                }`}
              >
                {opt.label}
              </button>
            ))}
          </div>

          {/* Folder scope toggle */}
          {scopeFolderId && (
            <label className="flex items-center gap-1.5 ml-auto text-xs text-slate-600 cursor-pointer select-none">
              <input
                type="checkbox"
                checked={useScopeFolder}
                onChange={(e) => handleScopeToggle(e.target.checked)}
                className="rounded border-slate-300 text-blue-600"
              />
              In "{scopeFolderName || 'selected folder'}" and subfolders
            </label>
          )}
        </div>

        {/* Results */}
        <div className="overflow-y-auto flex-1">
          {error && (
            <div className="px-4 py-3 text-sm text-red-600 bg-red-50 border-b border-red-100">{error}</div>
          )}

          {!searched && !loading && (
            <div className="px-4 py-8 text-center text-sm text-slate-400">
              Type to search across builds, models, projects, and folders.
            </div>
          )}

          {searched && results.length === 0 && !loading && (
            <div className="px-4 py-8 text-center text-sm text-slate-500">
              No results found for <strong>"{query}"</strong>.
            </div>
          )}

          {results.length > 0 && (
            <>
              <div className="px-4 py-2 text-xs text-slate-500 border-b border-slate-100">
                {total} result{total !== 1 ? 's' : ''}
              </div>
              <ul className="divide-y divide-slate-100">
                {results.map((item) => (
                  <li key={`${item.type}-${item.id}`}>
                    <button
                      className="w-full text-left px-4 py-3 hover:bg-slate-50 transition-colors flex items-start gap-3"
                      onClick={() => { onSelectResult(item); onClose(); }}
                    >
                      {/* Type badge */}
                      <span className={`mt-0.5 flex-shrink-0 inline-flex items-center px-2 py-0.5 rounded text-xs font-medium capitalize ${TYPE_ICON[item.type] ?? 'bg-slate-100 text-slate-600'}`}>
                        {item.type}
                      </span>

                      {/* Main info */}
                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 flex-wrap">
                          <span className="text-sm font-medium text-slate-800 truncate">{item.name}</span>
                          {item.status && (
                            <span className={`inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium capitalize ${STATUS_COLOR[item.status] ?? 'bg-slate-100 text-slate-600'}`}>
                              {item.status}
                            </span>
                          )}
                          {item.model_type && (
                            <span className="text-xs text-slate-500">{item.model_type}</span>
                          )}
                        </div>
                        {item.description && (
                          <p className="text-xs text-slate-500 mt-0.5 line-clamp-1">{item.description}</p>
                        )}
                        {item.breadcrumb && (
                          <p className="text-xs text-slate-400 mt-0.5 flex items-center gap-1">
                            <svg className="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z" />
                            </svg>
                            {item.breadcrumb}
                          </p>
                        )}
                      </div>

                      {/* Date */}
                      <span className="flex-shrink-0 text-xs text-slate-400 mt-0.5">
                        {new Date(item.created_at).toLocaleDateString()}
                      </span>
                    </button>
                  </li>
                ))}
              </ul>
            </>
          )}
        </div>

        {/* Footer hint */}
        <div className="px-4 py-2 border-t border-slate-100 text-xs text-slate-400 flex items-center gap-4">
          <span><kbd className="font-mono bg-slate-100 px-1 rounded">↵</kbd> search</span>
          <span><kbd className="font-mono bg-slate-100 px-1 rounded">Esc</kbd> close</span>
          <span className="ml-auto">Searches name &amp; description</span>
        </div>
      </div>
    </div>
  );
}
