import React from 'react';
import { Trash2 } from 'lucide-react';
import { TimeEntry } from './types';
import { formatDuration } from './utils';

interface EntriesListProps {
  darkMode: boolean;
  entries: TimeEntry[];
  onDeleteEntry: (id: number) => void;
}

export default function EntriesList({
  darkMode,
  entries,
  onDeleteEntry,
}: EntriesListProps) {
  return (
    <div className="space-y-3">
      <h2 className={`text-xl font-semibold mb-4 ${darkMode ? 'text-white' : 'text-gray-800'}`}>
        Recent Entries
      </h2>
      {entries.length === 0 ? (
        <p className={`text-center py-8 ${darkMode ? 'text-gray-400' : 'text-gray-500'}`}>
          No entries yet. Start tracking time!
        </p>
      ) : (
        entries.map((entry) => (
          <div
            key={entry.id}
            className={`rounded-lg p-4 flex items-center justify-between transition-colors ${
              darkMode 
                ? 'bg-gray-700 hover:bg-gray-600' 
                : 'bg-gray-50 hover:bg-gray-100'
            }`}
          >
            <div className="flex-1">
              <h3 className={`font-semibold ${darkMode ? 'text-white' : 'text-gray-800'}`}>
                {entry.project_name}
              </h3>
              {entry.description && (
                <p className={`text-sm mt-1 ${darkMode ? 'text-gray-400' : 'text-gray-600'}`}>
                  {entry.description}
                </p>
              )}
              <div className={`flex gap-4 mt-2 text-sm ${darkMode ? 'text-gray-400' : 'text-gray-500'}`}>
                <span>{new Date(entry.start_time).toLocaleString()}</span>
                {entry.end_time && (
                  <span className={`font-semibold ${darkMode ? 'text-indigo-400' : 'text-indigo-600'}`}>
                    {formatDuration(entry.start_time, entry.end_time, new Date())}
                  </span>
                )}
              </div>
            </div>
            <button
              onClick={() => onDeleteEntry(entry.id)}
              className={`p-2 rounded-lg transition-colors ${
                darkMode 
                  ? 'text-red-400 hover:text-red-300 hover:bg-red-900' 
                  : 'text-red-500 hover:text-red-700 hover:bg-red-50'
              }`}
            >
              <Trash2 className="w-5 h-5" />
            </button>
          </div>
        ))
      )}
    </div>
  );
}