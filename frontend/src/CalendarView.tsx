import React from 'react';
import { ChevronLeft, ChevronRight } from 'lucide-react';
import { WorkDay } from './types';
import { getDaysInMonth, isWeekday, formatHours } from './utils';

interface CalendarViewProps {
  darkMode: boolean;
  currentMonth: Date;
  workDays: WorkDay[];
  onMonthChange: (date: Date) => void;
  onUpdateWorkDay: (dateStr: string, field: string, value: any) => void;
  onDeleteWorkDay: (id: number) => void;
}

export default function CalendarView({
  darkMode,
  currentMonth,
  workDays,
  onMonthChange,
  onUpdateWorkDay,
  onDeleteWorkDay,
}: CalendarViewProps) {
  if (!currentMonth) {
    return <div>Loading...</div>;
  }

  const getWorkDayForDate = (dateStr: string) => {
    return workDays.find(wd => wd.date === dateStr);
  };

  const renderCalendar = () => {
    const { daysInMonth, startingDayOfWeek, year, month } = getDaysInMonth(currentMonth);
    const days = [];
    
    for (let i = 0; i < startingDayOfWeek; i++) {
      days.push(
        <div key={`empty-${i}`} className={`min-h-32 ${darkMode ? 'bg-gray-800' : 'bg-gray-50'}`}></div>
      );
    }
    
    for (let day = 1; day <= daysInMonth; day++) {
      const date = new Date(year, month, day);
      const dateStr = date.toISOString().split('T')[0];
      const isWeekDay = isWeekday(date);
      const workDay = getWorkDayForDate(dateStr);
      const isToday = new Date().toDateString() === date.toDateString();
      
      days.push(
        <div
          key={day}
          className={`min-h-32 border p-2 transition-colors ${
            darkMode 
              ? `${!isWeekDay ? 'bg-gray-800 border-gray-700' : 'bg-gray-900 border-gray-700'}`
              : `${!isWeekDay ? 'bg-gray-100 border-gray-200' : 'bg-white border-gray-200'}`
          } ${isToday ? 'ring-2 ring-blue-500' : ''}`}
        >
          <div className="flex justify-between items-start mb-2">
            <div className={`font-semibold text-sm ${darkMode ? 'text-gray-300' : 'text-gray-900'}`}>
              {day}
            </div>
            {isWeekDay && workDay?.id && (
              <button
                onClick={() => onDeleteWorkDay(workDay.id!)}
                className={`text-xs px-1 hover:text-red-500 ${darkMode ? 'text-gray-500' : 'text-gray-400'}`}
                title="Clear"
              >
                ✕
              </button>
            )}
          </div>
          
          {isWeekDay && (
            <div className="space-y-1">
              <input
                type="time"
                id={`start-${dateStr}`}
                name={`start-${dateStr}`}
                value={workDay?.start_time ? new Date(workDay.start_time).toTimeString().slice(0, 5) : ''}
                onChange={(e) => onUpdateWorkDay(dateStr, 'start_time', e.target.value)}
                placeholder="Start"
                className={`w-full text-xs px-1 py-1 border rounded ${
                  darkMode 
                    ? 'bg-gray-800 border-gray-600 text-gray-300 placeholder-gray-600' 
                    : 'bg-white border-gray-300 text-gray-900 placeholder-gray-400'
                } focus:ring-1 focus:ring-blue-500 focus:border-blue-500`}
              />
              
              <input
                type="time"
                id={`end-${dateStr}`}
                name={`end-${dateStr}`}
                value={workDay?.end_time ? new Date(workDay.end_time).toTimeString().slice(0, 5) : ''}
                onChange={(e) => onUpdateWorkDay(dateStr, 'end_time', e.target.value)}
                placeholder="End"
                className={`w-full text-xs px-1 py-1 border rounded ${
                  darkMode 
                    ? 'bg-gray-800 border-gray-600 text-gray-300 placeholder-gray-600' 
                    : 'bg-white border-gray-300 text-gray-900 placeholder-gray-400'
                } focus:ring-1 focus:ring-blue-500 focus:border-blue-500`}
              />

              <input
                type="number"
                id={`break-${dateStr}`}
                name={`break-${dateStr}`}
                step="0.25"
                min="0"
                max="8"
                value={workDay?.break_hours || ''}
                onChange={(e) => onUpdateWorkDay(dateStr, 'break_hours', e.target.value)}
                placeholder="Break (hrs)"
                className={`w-full text-xs px-1 py-1 border rounded ${
                  darkMode 
                    ? 'bg-gray-800 border-gray-600 text-orange-400 placeholder-gray-600' 
                    : 'bg-white border-gray-300 text-orange-600 placeholder-gray-400'
                } focus:ring-1 focus:ring-blue-500 focus:border-blue-500`}
              />

              {workDay?.start_time && workDay?.end_time && (
                <div className={`text-xs font-bold text-center py-1 rounded ${
                  darkMode ? 'bg-green-900 text-green-400' : 'bg-green-50 text-green-600'
                }`}>
                  {formatHours((new Date(workDay.end_time).getTime() - new Date(workDay.start_time).getTime()) / 3600000 - (workDay.break_hours || 0))}
                </div>
              )}
            </div>
          )}
        </div>
      );
    }
    
    return days;
  };

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <button
          onClick={() => onMonthChange(new Date(currentMonth.getFullYear(), currentMonth.getMonth() - 1))}
          className={`p-2 rounded-lg transition-colors ${
            darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-100'
          }`}
        >
          <ChevronLeft className={`w-6 h-6 ${darkMode ? 'text-gray-300' : 'text-gray-700'}`} />
        </button>
        <h2 className={`text-2xl font-bold ${darkMode ? 'text-white' : 'text-gray-900'}`}>
          {currentMonth.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })}
        </h2>
        <button
          onClick={() => onMonthChange(new Date(currentMonth.getFullYear(), currentMonth.getMonth() + 1))}
          className={`p-2 rounded-lg transition-colors ${
            darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-100'
          }`}
        >
          <ChevronRight className={`w-6 h-6 ${darkMode ? 'text-gray-300' : 'text-gray-700'}`} />
        </button>
      </div>

      <div className="grid grid-cols-7 gap-1 mb-2">
        {['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].map(day => (
          <div key={day} className={`text-center font-semibold py-2 ${darkMode ? 'text-gray-400' : 'text-gray-600'}`}>
            {day}
          </div>
        ))}
      </div>

      <div className="grid grid-cols-7 gap-1">
        {renderCalendar()}
      </div>

      <div className={`mt-6 p-4 rounded-lg ${darkMode ? 'bg-blue-900' : 'bg-blue-50'}`}>
        <p className={`text-sm ${darkMode ? 'text-gray-300' : 'text-gray-600'}`}>
          💡 Type directly into the time fields. Changes save automatically. Total hours appear after entering start and end times.
        </p>
      </div>
    </div>
  );
}