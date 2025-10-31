import React from 'react';
import { BarChart3 } from 'lucide-react';
import { ProjectStat } from './types';
import { formatHours } from './utils';

interface ProjectStatsProps {
  darkMode: boolean;
  stats: ProjectStat[];
  showStats: boolean;
  onToggleStats: () => void;
}

export default function ProjectStats({
  darkMode,
  stats,
  showStats,
  onToggleStats,
}: ProjectStatsProps) {
  return (
    <>
      <div className="flex gap-4 mb-6">
        <button
          onClick={onToggleStats}
          className={`px-4 py-2 rounded-lg flex items-center gap-2 transition-colors ${
            showStats 
              ? 'bg-indigo-600 text-white' 
              : darkMode 
                ? 'bg-gray-700 text-gray-300 hover:bg-gray-600' 
                : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
          }`}
        >
          <BarChart3 className="w-4 h-4" />
          {showStats ? 'Hide Stats' : 'Show Stats'}
        </button>
      </div>

      {showStats && stats.length > 0 && (
        <div className={`rounded-xl p-6 mb-8 ${darkMode ? 'bg-gray-700' : 'bg-indigo-50'}`}>
          <h2 className={`text-xl font-semibold mb-4 ${darkMode ? 'text-white' : 'text-gray-800'}`}>
            Project Statistics
          </h2>
          <div className="space-y-3">
            {stats.map((stat) => (
              <div key={stat.project_name} className={`rounded-lg p-4 ${darkMode ? 'bg-gray-800' : 'bg-white'}`}>
                <div className="flex justify-between items-center">
                  <div>
                    <h3 className={`font-semibold ${darkMode ? 'text-white' : 'text-gray-800'}`}>
                      {stat.project_name}
                    </h3>
                    <p className={`text-sm ${darkMode ? 'text-gray-400' : 'text-gray-600'}`}>
                      {stat.entry_count} entries
                    </p>
                  </div>
                  <div className={`text-2xl font-bold ${darkMode ? 'text-indigo-400' : 'text-indigo-600'}`}>
                    {formatHours(stat.total_hours)}
                  </div>
                </div>
              </div>
            ))}
          </div>
        </div>
      )}
    </>
  );
}