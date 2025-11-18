import React, { useState, useEffect } from 'react';
import { Clock, Play, Trash2, BarChart3, Calendar, ChevronLeft, ChevronRight, Moon, Sun, LogOut } from 'lucide-react';

const API_URL = '/api';

interface TimeEntry {
  id: number;
  project_name: string;
  description: string;
  start_time: string;
  end_time: string | null;
  duration: number | null;
  created_at: string;
}

interface WorkDay {
  id?: number;
  date: string;
  project_id?: number;
  project_name?: string;
  start_time: string | null;
  end_time: string | null;
  break_hours: number;
  total_hours: number | null;
}

interface ProjectStat {
  project_name: string;
  entry_count: number;
  total_hours: number;
}

interface Project {
  id: number;
  name: string;
  description: string;
  created_at: string;
}

export default function TimesheetTracker() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [authToken, setAuthToken] = useState<string | null>(null);
  const [currentUser, setCurrentUser] = useState<{ username: string; user_id: number } | null>(null);
  const [authView, setAuthView] = useState<string>('login');
  const [authError, setAuthError] = useState<string>('');
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');

  const [darkMode, setDarkMode] = useState(() => {
    const saved = localStorage.getItem('darkMode');
    return saved ? JSON.parse(saved) : false;
  });
  const [view, setView] = useState<'home' | 'calendar' | 'report'>('home');
  const [stats, setStats] = useState<ProjectStat[]>([]);
  const [projects, setProjects] = useState<Project[]>([]);
  const [activeProject, setActiveProject] = useState<Project | null>(null);
  const [showAddProject, setShowAddProject] = useState(false);
  const [newProjectName, setNewProjectName] = useState('');
  const [newProjectDesc, setNewProjectDesc] = useState('');
  
  // Calendar state
  const [currentMonth, setCurrentMonth] = useState(new Date());
  const [workDays, setWorkDays] = useState<WorkDay[]>([]);
  const [selectedDay, setSelectedDay] = useState<string | null>(null);
  const [editingDay, setEditingDay] = useState<WorkDay | null>(null);

  // Save dark mode preference
  useEffect(() => {
    localStorage.setItem('darkMode', JSON.stringify(darkMode));
  }, [darkMode]);

  // Check authentication on mount
  useEffect(() => {
    checkAuth();
  }, []);

  // Load data after authentication
  useEffect(() => {
    if (isAuthenticated && authToken) {
      loadProjects();
      fetchStats();
    }
  }, [isAuthenticated, authToken]);

  // Fetch work days when view/month/project changes
  useEffect(() => {
    if (isAuthenticated && authToken && (view === 'calendar' || view === 'report')) {
      fetchWorkDays();
    }
  }, [view, currentMonth, activeProject, isAuthenticated, authToken]);

  const checkAuth = async () => {
    const token = localStorage.getItem('authToken');
    if (!token) {
      setIsLoading(false);
      return;
    }

    try {
      const res = await fetch(`${API_URL}/auth/verify`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });

      if (res.ok) {
        const data = await res.json();
        setAuthToken(token);
        setCurrentUser({ username: data.username, user_id: data.user_id });
        setIsAuthenticated(true);
      } else {
        localStorage.removeItem('authToken');
      }
    } catch (error) {
      console.error('Auth check failed:', error);
      localStorage.removeItem('authToken');
    } finally {
      setIsLoading(false);
    }
  };

  const handleLogin = async (username: string, password: string) => {
    setAuthError('');
    try {
      const res = await fetch(`${API_URL}/auth/login`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ username, password })
      });

      const data = await res.json();

      if (res.ok) {
        localStorage.setItem('authToken', data.token);
        setAuthToken(data.token);
        setCurrentUser({ username: data.username, user_id: data.user_id });
        setIsAuthenticated(true);
        // Clear form
        setUsername('');
        setPassword('');
      } else {
        setAuthError(data.error || 'Login failed');
      }
    } catch (error) {
      setAuthError('Network error. Please try again.');
    }
  };

  // const handleRegister = async (username: string, password: string) => {
  //   setAuthError('');
  //   try {
  //     const res = await fetch(`${API_URL}/auth/register`, {
  //       method: 'POST',
  //       headers: { 'Content-Type': 'application/json' },
  //       body: JSON.stringify({ username, password })
  //     });

  //     const data = await res.json();

  //     if (res.ok) {
  //       localStorage.setItem('authToken', data.token);
  //       setAuthToken(data.token);
  //       setCurrentUser({ username: data.username, user_id: data.user_id });
  //       setIsAuthenticated(true);
  //       // Clear form
  //       setUsername('');
  //       setPassword('');
  //     } else {
  //       setAuthError(data.error || 'Registration failed');
  //     }
  //   } catch (error) {
  //     setAuthError('Network error. Please try again.');
  //   }
  // };

  const handleLogout = () => {
    localStorage.removeItem('authToken');
    setAuthToken(null);
    setCurrentUser(null);
    setIsAuthenticated(false);
  };

const loadProjects = async () => {
  try {
    // Try fetching from backend
    const response = await fetch('/api/projects');
    if (!response.ok) {
      throw new Error('Failed to fetch projects from server');
    }

    const backendProjects: Project[] = await response.json();

    if (backendProjects.length > 0) {
      setProjects(backendProjects);
      localStorage.setItem('projects', JSON.stringify(backendProjects));

      // Set active project if not already in localStorage
      const savedActiveProject = localStorage.getItem('activeProject');
      if (savedActiveProject) {
        setActiveProject(JSON.parse(savedActiveProject));
      } else {
        setActiveProject(backendProjects[0]);
        localStorage.setItem('activeProject', JSON.stringify(backendProjects[0]));
      }
    } else {
      // fallback to localStorage if backend has no projects
      const savedProjects = localStorage.getItem('projects');
      const savedActiveProject = localStorage.getItem('activeProject');

      if (savedProjects) {
        setProjects(JSON.parse(savedProjects));
      }

      if (savedActiveProject) {
        setActiveProject(JSON.parse(savedActiveProject));
      }
    }
  } catch (error) {
    console.error('Error loading projects:', error);

    // fallback to localStorage on error
    const savedProjects = localStorage.getItem('projects');
    const savedActiveProject = localStorage.getItem('activeProject');

    if (savedProjects) {
      setProjects(JSON.parse(savedProjects));
    }

    if (savedActiveProject) {
      setActiveProject(JSON.parse(savedActiveProject));
    }
  }
};


const addProject = async () => {
  if (!newProjectName.trim()) return;

  const newProject: Project = {
    id: projects.length > 0 ? Math.max(...projects.map(p => p.id)) + 1 : 1,
    name: newProjectName.trim(),
    description: newProjectDesc.trim(),
    created_at: new Date().toISOString(),
  };

  try {
    // POST to backend
    const response = await fetch(`${API_URL}/api/projects`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify(newProject),
    });

    if (!response.ok) {
      throw new Error('Failed to add project on server');
    }

    const savedProject = await response.json(); // backend might return the project with its ID

    // Update frontend state
    const updatedProjects = [...projects, savedProject];
    setProjects(updatedProjects);
    localStorage.setItem('projects', JSON.stringify(updatedProjects));

    // Set as active if first project
    if (projects.length === 0) {
      setActiveProject(savedProject);
      localStorage.setItem('activeProject', JSON.stringify(savedProject));
    }

    setNewProjectName('');
    setNewProjectDesc('');
    setShowAddProject(false);
  } catch (error) {
    console.error('Error adding project:', error);
    // optionally show an error notification to the user
  }
};


const deleteProject = async (projectId: number) => {
  try {
    const response = await fetch(`/api/projects/${projectId}`, {
      method: 'DELETE',
    });

    if (!response.ok) {
      throw new Error('Failed to delete project on server');
    }

    // Update frontend state
    const updatedProjects = projects.filter(p => p.id !== projectId);
    setProjects(updatedProjects);
    localStorage.setItem('projects', JSON.stringify(updatedProjects));

    // Clear active if deleted
    if (activeProject?.id === projectId) {
      setActiveProject(null);
      localStorage.removeItem('activeProject');
    }
  } catch (error) {
    console.error('Error deleting project:', error);
    // Optionally show user feedback
  }
};

  const setAsActive = (project: Project) => {
    setActiveProject(project);
    localStorage.setItem('activeProject', JSON.stringify(project));
  };

  const fetchStats = async () => {
    if (!authToken) return;
    
    try {
      const res = await fetch(`${API_URL}/stats`, {
        headers: {
          'Authorization': `Bearer ${authToken}`
        }
      });
      const data = await res.json();
      setStats(data || []);
    } catch (error) {
      console.error('Error fetching stats:', error);
    }
  };

  const fetchWorkDays = async () => {
    if (!authToken) return;
    
    const year = currentMonth.getFullYear();
    const month = currentMonth.getMonth() + 1;
    
    let url = `${API_URL}/workdays/${year}/${month}`;
    if (activeProject) {
      url += `?project_id=${activeProject.id}`;
    }
    
    try {
      const res = await fetch(url, {
        headers: {
          'Authorization': `Bearer ${authToken}`
        }
      });
      const data = await res.json();
      setWorkDays(data || []);
    } catch (error) {
      console.error('Error fetching work days:', error);
    }
  };

  const saveWorkDay = async (workDay: WorkDay) => {
    if (!authToken) return;
    
    if (!workDay.date && selectedDay) {
      workDay.date = selectedDay;
    }

    if (activeProject) {
      workDay.project_id = activeProject.id;
      workDay.project_name = activeProject.name;
    }

    const method = workDay.id ? 'PUT' : 'POST';
    const url = workDay.id ? `${API_URL}/workdays/${workDay.id}` : `${API_URL}/workdays`;

    try {
      const res = await fetch(url, {
        method,
        headers: { 
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${authToken}`
        },
        body: JSON.stringify(workDay),
      });

      if (res.ok) {
        setSelectedDay(null);
        setEditingDay(null);
        await fetchWorkDays();
      } else {
        const error = await res.json();
        console.error('Error saving work day:', error);
        alert('Failed to save work day: ' + (error.error || 'Unknown error'));
      }
    } catch (error) {
      console.error('Error saving work day:', error);
      alert('Failed to save work day. Please try again.');
    }
  };

  const deleteWorkDay = async (id: number) => {
    if (!authToken) return;
    
    try {
      const res = await fetch(`${API_URL}/workdays/${id}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${authToken}`
        }
      });

      if (res.ok) {
        setSelectedDay(null);
        setEditingDay(null);
        await fetchWorkDays();
      }
    } catch (error) {
      console.error('Error deleting work day:', error);
    }
  };

  const formatDuration = (start: string, end: string | null) => {
    const startTime = new Date(start);
    const endTime = end ? new Date(end) : new Date();
    const diff = endTime.getTime() - startTime.getTime();
    const hours = Math.floor(diff / 3600000);
    const minutes = Math.floor((diff % 3600000) / 60000);
    const seconds = Math.floor((diff % 60000) / 1000);
    return `${hours}h ${minutes}m ${seconds}s`;
  };

  const formatHours = (hours: number) => {
    const h = Math.floor(hours);
    const m = Math.round((hours - h) * 60);
    return `${h}h ${m}m`;
  };

  const getDaysInMonth = (date: Date) => {
    const year = date.getFullYear();
    const month = date.getMonth();
    const firstDay = new Date(year, month, 1);
    const lastDay = new Date(year, month + 1, 0);
    const daysInMonth = lastDay.getDate();
    const startingDayOfWeek = firstDay.getDay();
    
    return { daysInMonth, startingDayOfWeek, year, month };
  };

  const isWeekday = (date: Date) => {
    const day = date.getDay();
    return day !== 0 && day !== 6;
  };

  const getWorkDayForDate = (dateStr: string) => {
    const found = workDays.find(wd => {
      const wdDate = wd.date.split('T')[0];
      const dateMatches = wdDate === dateStr;
      
      if (activeProject) {
        return dateMatches && wd.project_id === activeProject.id;
      }
      
      return dateMatches;
    });
    return found;
  };

  const renderCalendar = () => {
    const { daysInMonth, startingDayOfWeek, year, month } = getDaysInMonth(currentMonth);
    const days = [];
    
    for (let i = 0; i < startingDayOfWeek; i++) {
      days.push(<div key={`empty-${i}`} className={`h-24 ${darkMode ? 'bg-gray-800' : 'bg-gray-50'}`}></div>);
    }
    
    for (let day = 1; day <= daysInMonth; day++) {
      const date = new Date(year, month, day);
      const dateStr = `${year}-${String(month + 1).padStart(2, '0')}-${String(day).padStart(2, '0')}`;
      const isWeekDay = isWeekday(date);
      const workDay = getWorkDayForDate(dateStr);
      const isToday = new Date().toDateString() === date.toDateString();
      
      days.push(
        <div
          key={day}
          className={`h-24 border p-2 cursor-pointer transition-colors ${
            darkMode 
              ? `${!isWeekDay ? 'bg-gray-800 border-gray-700' : 'bg-gray-900 border-gray-700 hover:bg-gray-800'}`
              : `${!isWeekDay ? 'bg-gray-100 border-gray-200' : 'bg-white border-gray-200 hover:bg-blue-50'}`
          } ${isToday ? 'ring-2 ring-blue-500' : ''}`}
          onClick={() => isWeekDay && setSelectedDay(dateStr)}
        >
          <div className={`font-semibold text-sm mb-1 ${darkMode ? 'text-gray-300' : 'text-gray-900'}`}>{day}</div>
          {isWeekDay && workDay && workDay.start_time && workDay.end_time && (
            <div className="text-xs space-y-1">
              <div className={darkMode ? 'text-gray-400' : 'text-gray-600'}>
                {new Date(workDay.start_time).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false })} - 
                {new Date(workDay.end_time).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', hour12: false })}
              </div>
              {workDay.break_hours > 0 && (
                <div className={darkMode ? 'text-orange-400' : 'text-orange-600'}>Break: {workDay.break_hours}h</div>
              )}
              <div className={`font-bold ${darkMode ? 'text-green-400' : 'text-green-600'}`}>
                {workDay.total_hours !== null && formatHours(workDay.total_hours)}
              </div>
            </div>
          )}
        </div>
      );
    }
    
    return days;
  };

  const renderHome = () => {
    return (
      <div>
        {activeProject ? (
          <div className={`rounded-xl p-6 mb-8 border-2 ${
            darkMode 
              ? 'bg-indigo-900 border-indigo-700' 
              : 'bg-indigo-50 border-indigo-200'
          }`}>
            <div className="flex items-center justify-between mb-2">
              <h2 className={`text-sm font-medium ${darkMode ? 'text-indigo-300' : 'text-indigo-600'}`}>
                ACTIVE PROJECT
              </h2>
              <span className={`px-2 py-1 rounded text-xs font-semibold ${
                darkMode ? 'bg-green-900 text-green-300' : 'bg-green-100 text-green-700'
              }`}>
                Active
              </span>
            </div>
            <h3 className={`text-2xl font-bold mb-2 ${darkMode ? 'text-white' : 'text-gray-900'}`}>
              {activeProject.name}
            </h3>
            {activeProject.description && (
              <p className={`${darkMode ? 'text-indigo-200' : 'text-gray-600'}`}>
                {activeProject.description}
              </p>
            )}
          </div>
        ) : (
          <div className={`rounded-xl p-6 mb-8 border-2 border-dashed ${
            darkMode 
              ? 'bg-gray-800 border-gray-600' 
              : 'bg-gray-50 border-gray-300'
          }`}>
            <p className={`text-center ${darkMode ? 'text-gray-400' : 'text-gray-500'}`}>
              No active project. Create a project and set it as active.
            </p>
          </div>
        )}

        <button
          onClick={() => setShowAddProject(true)}
          className={`w-full mb-6 px-4 py-3 rounded-lg flex items-center justify-center gap-2 transition-colors ${
            darkMode 
              ? 'bg-indigo-600 hover:bg-indigo-700 text-white' 
              : 'bg-indigo-600 hover:bg-indigo-700 text-white'
          }`}
        >
          <Play className="w-5 h-5" />
          Create New Project
        </button>

        <div>
          <h3 className={`text-xl font-semibold mb-4 ${darkMode ? 'text-white' : 'text-gray-800'}`}>
            All Projects ({projects.length})
          </h3>
          
          {projects.length === 0 ? (
            <p className={`text-center py-8 ${darkMode ? 'text-gray-400' : 'text-gray-500'}`}>
              No projects yet. Create your first project!
            </p>
          ) : (
            <div className="space-y-3">
              {projects.map((project) => (
                <div
                  key={project.id}
                  className={`rounded-lg p-4 flex items-center justify-between transition-colors ${
                    project.id === activeProject?.id
                      ? darkMode ? 'bg-indigo-900 border-2 border-indigo-600' : 'bg-indigo-50 border-2 border-indigo-300'
                      : darkMode ? 'bg-gray-700 hover:bg-gray-600' : 'bg-gray-50 hover:bg-gray-100'
                  }`}
                >
                  <div className="flex-1">
                    <div className="flex items-center gap-2">
                      <h4 className={`font-semibold ${darkMode ? 'text-white' : 'text-gray-900'}`}>
                        {project.name}
                      </h4>
                      {project.id === activeProject?.id && (
                        <span className={`px-2 py-0.5 rounded text-xs font-semibold ${
                          darkMode ? 'bg-green-900 text-green-300' : 'bg-green-100 text-green-700'
                        }`}>
                          Active
                        </span>
                      )}
                    </div>
                    {project.description && (
                      <p className={`text-sm mt-1 ${darkMode ? 'text-gray-400' : 'text-gray-600'}`}>
                        {project.description}
                      </p>
                    )}
                    <p className={`text-xs mt-1 ${darkMode ? 'text-gray-500' : 'text-gray-400'}`}>
                      Created: {new Date(project.created_at).toLocaleDateString()}
                    </p>
                  </div>
                  
                  <div className="flex gap-2">
                    {project.id !== activeProject?.id && (
                      <button
                        onClick={() => setAsActive(project)}
                        className={`px-3 py-1.5 rounded text-sm transition-colors ${
                          darkMode 
                            ? 'bg-indigo-600 hover:bg-indigo-700 text-white' 
                            : 'bg-indigo-600 hover:bg-indigo-700 text-white'
                        }`}
                      >
                        Set Active
                      </button>
                    )}
                    <button
                      onClick={() => deleteProject(project.id)}
                      className={`p-2 rounded-lg transition-colors ${
                        darkMode 
                          ? 'text-red-400 hover:text-red-300 hover:bg-red-900' 
                          : 'text-red-500 hover:text-red-700 hover:bg-red-50'
                      }`}
                    >
                      <Trash2 className="w-4 h-4" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>
    );
  };

  const renderReport = () => {
    if (!activeProject) {
      return (
        <div className={`rounded-xl p-8 border-2 border-dashed ${
          darkMode ? 'bg-gray-800 border-gray-600' : 'bg-gray-50 border-gray-300'
        }`}>
          <p className={`text-center ${darkMode ? 'text-gray-400' : 'text-gray-500'}`}>
            Please select an active project from the Home screen to view reports.
          </p>
          <button
            onClick={() => setView('home')}
            className="mt-4 mx-auto block px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg transition-colors"
          >
            Go to Home
          </button>
        </div>
      );
    }

    const sortedWorkDays = [...workDays]
      .filter(wd => wd.start_time && wd.end_time && wd.total_hours)
      .sort((a, b) => new Date(a.date).getTime() - new Date(b.date).getTime());

    const totalHours = sortedWorkDays.reduce((sum, wd) => sum + (wd.total_hours || 0), 0);

    return (
      <div>
        <div className={`rounded-lg p-4 mb-6 ${darkMode ? 'bg-indigo-900' : 'bg-indigo-50'}`}>
          <div className="flex items-center justify-between">
            <div>
              <p className={`text-sm ${darkMode ? 'text-indigo-300' : 'text-indigo-600'}`}>
                Report for
              </p>
              <p className={`text-lg font-semibold ${darkMode ? 'text-white' : 'text-gray-900'}`}>
                {activeProject.name}
              </p>
            </div>
          </div>
        </div>

        <div className="flex items-center justify-between mb-6 print:mb-4">
          <div className="flex items-center gap-4">
            <button
              onClick={() => setCurrentMonth(new Date(currentMonth.getFullYear(), currentMonth.getMonth() - 1))}
              className={`p-2 rounded-lg transition-colors print:hidden ${
                darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-100'
              }`}
            >
              <ChevronLeft className={`w-6 h-6 ${darkMode ? 'text-gray-300' : 'text-gray-700'}`} />
            </button>
            <h2 className={`text-2xl font-bold ${darkMode ? 'text-white' : 'text-gray-900'}`}>
              {currentMonth.toLocaleDateString('en-US', { month: 'long', year: 'numeric' })}
            </h2>
            <button
              onClick={() => setCurrentMonth(new Date(currentMonth.getFullYear(), currentMonth.getMonth() + 1))}
              className={`p-2 rounded-lg transition-colors print:hidden ${
                darkMode ? 'hover:bg-gray-700' : 'hover:bg-gray-100'
              }`}
            >
              <ChevronRight className={`w-6 h-6 ${darkMode ? 'text-gray-300' : 'text-gray-700'}`} />
            </button>
          </div>
          <button
            onClick={() => window.print()}
            className={`px-4 py-2 rounded-lg transition-colors print:hidden ${
              darkMode 
                ? 'bg-indigo-600 hover:bg-indigo-700 text-white' 
                : 'bg-indigo-600 hover:bg-indigo-700 text-white'
            }`}
          >
            Print Report
          </button>
        </div>

        <div className={`rounded-lg overflow-hidden border ${
          darkMode ? 'border-gray-700' : 'border-gray-300'
        }`}>
<table
  className={`w-full border border-collapse ${
    darkMode ? 'border-gray-700' : 'border-gray-300'
  }`}
>
  <thead className={darkMode ? 'bg-gray-700' : 'bg-gray-100'}>
    <tr
      className={`border-b ${
        darkMode ? 'border-gray-600' : 'border-gray-300'
      }`}
    >
      <th
        className={`px-4 py-3 pl-8 text-left font-semibold border ${
          darkMode ? 'border-gray-700 text-white' : 'border-gray-300 text-gray-900'
        }`}
      >
        DAY & DATE
      </th>
      <th
        className={`px-4 py-3 text-center font-semibold border ${
          darkMode ? 'border-gray-700 text-white' : 'border-gray-300 text-gray-900'
        }`}
      >
        SHIFTS
      </th>
      <th
        className={`px-4 py-3 text-center font-semibold border ${
          darkMode ? 'border-gray-700 text-white' : 'border-gray-300 text-gray-900'
        }`}
      >
        BREAK
      </th>
      <th
        className={`px-4 py-3 pr-8 text-right font-semibold border ${
          darkMode ? 'border-gray-700 text-white' : 'border-gray-300 text-gray-900'
        }`}
      >
        TOTAL HOURS
      </th>
    </tr>
  </thead>

  <tbody className={darkMode ? 'bg-gray-800' : 'bg-white'}>
    {sortedWorkDays.length === 0 ? (
      <tr>
        <td
          colSpan={4}
          className={`px-4 py-8 text-center border ${
            darkMode ? 'border-gray-700 text-gray-400' : 'border-gray-300 text-gray-500'
          }`}
        >
          No work entries for this month
        </td>
      </tr>
    ) : (
      sortedWorkDays.map((wd) => {
        const date = new Date(wd.date.split('T')[0]);
        const dayName = date.toLocaleDateString('en-GB', { weekday: 'short' });
        const dateStr = date.toLocaleDateString('en-GB');

        return (
          <tr
            key={wd.id}
            className={`border ${
              darkMode ? 'border-gray-700' : 'border-gray-300'
            }`}
          >
            <td
              className={`px-4 py-3 pl-8 border ${
                darkMode ? 'border-gray-700 text-gray-300' : 'border-gray-300 text-gray-900'
              }`}
            >
              {dayName} {dateStr}
            </td>
            <td
              className={`px-4 py-3 text-center font-mono border ${
                darkMode ? 'border-gray-700 text-gray-300' : 'border-gray-300 text-gray-900'
              }`}
            >
              {new Date(wd.start_time!).toLocaleTimeString('en-GB', {
                hour: '2-digit',
                minute: '2-digit',
                hour12: false,
              })}{' '}
              -{' '}
              {new Date(wd.end_time!).toLocaleTimeString('en-GB', {
                hour: '2-digit',
                minute: '2-digit',
                hour12: false,
              })}
            </td>
            <td
              className={`px-4 py-3 text-center border ${
                darkMode ? 'border-gray-700 text-orange-400' : 'border-gray-300 text-orange-600'
              }`}
            >
              {wd.break_hours > 0 ? `${wd.break_hours}h` : '-'}
            </td>
            <td
              className={`px-4 py-3 pr-8 text-right font-semibold border ${
                darkMode ? 'border-gray-700 text-green-400' : 'border-gray-300 text-green-600'
              }`}
            >
              {formatHours(wd.total_hours!)}
            </td>
          </tr>
        );
      })
    )}

    {sortedWorkDays.length > 0 && (
      <tr
        className={`border-t-2 font-bold ${
          darkMode
            ? 'border-gray-600 bg-gray-700'
            : 'border-gray-400 bg-gray-50'
        }`}
      >
        <td
          className={`px-4 py-3 pl-8 border ${
            darkMode ? 'border-gray-700 text-white' : 'border-gray-300 text-gray-900'
          }`}
        >
          TOTAL
        </td>
        <td
          className={`px-4 py-3 text-center border ${
            darkMode ? 'border-gray-700 text-gray-400' : 'border-gray-300 text-gray-600'
          }`}
        >
          {sortedWorkDays.length} days
        </td>
        <td
          className={`px-4 py-3 text-center border ${
            darkMode ? 'border-gray-700 text-gray-400' : 'border-gray-300 text-gray-600'
          }`}
        >
          -
        </td>
        <td
          className={`px-4 py-3 pr-8 text-right text-lg border ${
            darkMode ? 'border-gray-700 text-indigo-400' : 'border-gray-300 text-indigo-600'
          }`}
        >
          {formatHours(totalHours)}
        </td>
      </tr>
    )}
  </tbody>
</table>

        </div>
      </div>
    );
  };

  const renderWorkDayModal = () => {
    if (!selectedDay) return null;

    const existingWorkDay = getWorkDayForDate(selectedDay);
    
    let workDay: WorkDay;
    
    if (editingDay) {
      workDay = editingDay;
    } else if (existingWorkDay) {
      workDay = { ...existingWorkDay };
      setTimeout(() => setEditingDay(workDay), 0);
    } else {
      workDay = {
        date: selectedDay,
        start_time: null,
        end_time: null,
        break_hours: 0,
        total_hours: null,
      };
    }

    const [year, month, day] = selectedDay.split('-').map(Number);
    const displayDate = new Date(year, month - 1, day);

    return (
      <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" onClick={() => { setSelectedDay(null); setEditingDay(null); }}>
        <div className={`${darkMode ? 'bg-gray-800' : 'bg-white'} rounded-xl p-6 max-w-md w-full mx-4`} onClick={(e) => e.stopPropagation()}>
          <h3 className={`text-xl font-bold mb-4 ${darkMode ? 'text-white' : 'text-gray-900'}`}>
            {workDay.id ? 'Edit' : 'Add'} Work Day - {displayDate.toLocaleDateString('en-US', { weekday: 'long', month: 'long', day: 'numeric', year: 'numeric' })}
          </h3>
          
          <div className="space-y-4">
            <div>
              <label className={`block text-sm font-medium mb-1 ${darkMode ? 'text-gray-300' : 'text-gray-700'}`}>Start Time</label>
              <input
                type="time"
                autoFocus
                value={workDay.start_time ? new Date(workDay.start_time).toTimeString().slice(0, 5) : ''}
                onChange={(e) => {
                  const dateTime = new Date(selectedDay + 'T' + e.target.value + ':00');
                  setEditingDay({ 
                    ...workDay, 
                    start_time: dateTime.toISOString(), 
                    date: selectedDay 
                  });
                }}
                className={`w-full px-3 py-2 border rounded-lg ${
                  darkMode 
                    ? 'bg-gray-700 border-gray-600 text-white' 
                    : 'bg-white border-gray-300 text-gray-900'
                }`}
              />
            </div>

            <div>
              <label className={`block text-sm font-medium mb-1 ${darkMode ? 'text-gray-300' : 'text-gray-700'}`}>End Time</label>
              <input
                type="time"
                value={workDay.end_time ? new Date(workDay.end_time).toTimeString().slice(0, 5) : ''}
                onChange={(e) => {
                  const dateTime = new Date(selectedDay + 'T' + e.target.value + ':00');
                  setEditingDay({ 
                    ...workDay, 
                    end_time: dateTime.toISOString(), 
                    date: selectedDay 
                  });
                }}
                className={`w-full px-3 py-2 border rounded-lg ${
                  darkMode 
                    ? 'bg-gray-700 border-gray-600 text-white' 
                    : 'bg-white border-gray-300 text-gray-900'
                }`}
              />
            </div>

            <div>
              <label className={`block text-sm font-medium mb-1 ${darkMode ? 'text-gray-300' : 'text-gray-700'}`}>Break Hours</label>
              <input
                type="number"
                step="0.25"
                min="0"
                value={workDay.break_hours || ''}
                onChange={(e) => setEditingDay({ 
                  ...workDay, 
                  break_hours: parseFloat(e.target.value) || 0, 
                  date: selectedDay 
                })}
                className={`w-full px-3 py-2 border rounded-lg ${
                  darkMode 
                    ? 'bg-gray-700 border-gray-600 text-white' 
                    : 'bg-white border-gray-300 text-gray-900'
                }`}
              />
            </div>

            {workDay.start_time && workDay.end_time && (
              <div className={`${darkMode ? 'bg-blue-900' : 'bg-blue-50'} p-3 rounded-lg`}>
                <div className={`text-sm ${darkMode ? 'text-gray-300' : 'text-gray-600'}`}>Total Work Hours:</div>
                <div className="text-2xl font-bold text-blue-500">
                  {formatHours((new Date(workDay.end_time).getTime() - new Date(workDay.start_time).getTime()) / 3600000 - (workDay.break_hours || 0))}
                </div>
              </div>
            )}

            <div className="flex gap-2">
              <button
                onClick={() => {
                  const dataToSave = editingDay || workDay;
                  saveWorkDay(dataToSave);
                }}
                className="flex-1 bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg transition-colors"
              >
                Save
              </button>
              {workDay.id && (
                <button
                  onClick={() => deleteWorkDay(workDay.id!)}
                  className="bg-red-500 hover:bg-red-600 text-white px-4 py-2 rounded-lg transition-colors"
                >
                  Delete
                </button>
              )}
              <button
                onClick={() => { setSelectedDay(null); setEditingDay(null); }}
                className={`px-4 py-2 rounded-lg transition-colors ${
                  darkMode 
                    ? 'bg-gray-700 hover:bg-gray-600 text-gray-200' 
                    : 'bg-gray-300 hover:bg-gray-400 text-gray-800'
                }`}
              >
                Cancel
              </button>
            </div>
          </div>
        </div>
      </div>
    );
  };

  const renderAuthScreen = () => {
    const handleSubmit = (e: React.FormEvent) => {
      e.preventDefault();
      if (authView === 'login') {
        handleLogin(username, password);
      } else {
        console.log("Invalid View");
      }
    };

    return (
      <div className={`min-h-screen w-full flex items-center justify-center p-4 transition-colors ${
        darkMode 
          ? 'bg-gradient-to-br from-gray-900 to-gray-800' 
          : 'bg-gradient-to-br from-blue-50 to-indigo-100'
      }`}>
        <div className={`w-full max-w-md rounded-2xl shadow-xl p-8 transition-colors ${
          darkMode ? 'bg-gray-800' : 'bg-white'
        }`}>
          <div className="flex items-center justify-between mb-8">
            <div className="flex items-center gap-3">
              <Clock className={`w-8 h-8 ${darkMode ? 'text-indigo-400' : 'text-indigo-600'}`} />
              <h1 className={`text-2xl font-bold ${darkMode ? 'text-white' : 'text-gray-800'}`}>
                Timesheet Tracker Login
              </h1>
            </div>
            <button
              onClick={() => setDarkMode(!darkMode)}
              className={`p-2 rounded-lg transition-colors ${
                darkMode 
                  ? 'bg-gray-700 hover:bg-gray-600 text-yellow-400' 
                  : 'bg-gray-200 hover:bg-gray-300 text-gray-700'
              }`}
            >
              {darkMode ? <Sun className="w-5 h-5" /> : <Moon className="w-5 h-5" />}
            </button>
          </div>

          <div className="mb-6">
            <div className="flex gap-2 mb-6">
              {/* <button
                onClick={() => { setAuthView('login'); setAuthError(''); }}
                className={`flex-1 py-2 rounded-lg font-medium transition-colors ${
                  authView === 'login'
                    ? 'bg-indigo-600 text-white'
                    : darkMode
                      ? 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                      : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
                }`}
              >
                Login
              </button>
              <button
                onClick={() => { setAuthView('register'); setAuthError(''); }}
                className={`flex-1 py-2 rounded-lg font-medium transition-colors ${
                  authView === 'register'
                    ? 'bg-indigo-600 text-white'
                    : darkMode
                      ? 'bg-gray-700 text-gray-300 hover:bg-gray-600'
                      : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
                }`}
              >
                Register
              </button> */}
            </div>

            <form onSubmit={handleSubmit} className="space-y-4">
              <div>
                <label className={`block text-sm font-medium mb-2 ${darkMode ? 'text-gray-300' : 'text-gray-700'}`}>
                  Username
                </label>
                <input
                  type="text"
                  value={username}
                  onChange={(e) => setUsername(e.target.value)}
                  required
                  minLength={authView === 'register' ? 3 : undefined}
                  className={`w-full px-4 py-2 border rounded-lg transition-colors ${
                    darkMode 
                      ? 'bg-gray-700 border-gray-600 text-white placeholder-gray-400 focus:ring-2 focus:ring-indigo-500' 
                      : 'bg-white border-gray-300 text-gray-900 focus:ring-2 focus:ring-indigo-500'
                  }`}
                  placeholder="Enter username"
                />
              </div>

              <div>
                <label className={`block text-sm font-medium mb-2 ${darkMode ? 'text-gray-300' : 'text-gray-700'}`}>
                  Password
                </label>
                <input
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  required
                  minLength={authView === 'register' ? 6 : undefined}
                  className={`w-full px-4 py-2 border rounded-lg transition-colors ${
                    darkMode 
                      ? 'bg-gray-700 border-gray-600 text-white placeholder-gray-400 focus:ring-2 focus:ring-indigo-500' 
                      : 'bg-white border-gray-300 text-gray-900 focus:ring-2 focus:ring-indigo-500'
                  }`}
                  placeholder="Enter password"
                />
                {authView === 'register' && (
                  <p className={`text-xs mt-1 ${darkMode ? 'text-gray-400' : 'text-gray-500'}`}>
                    Must be at least 6 characters
                  </p>
                )}
              </div>

              {authError && (
                <div className="bg-red-500 bg-opacity-10 border border-red-500 text-red-500 px-4 py-2 rounded-lg text-sm">
                  {authError}
                </div>
              )}
                <label className={`block text-sm font-medium mb-2 ${darkMode ? 'text-gray-300' : 'text-gray-700'}`}>
                  
                </label>  
              <button
                type="submit"
                className="w-full bg-indigo-600 hover:bg-indigo-700 text-white font-medium py-2 rounded-lg transition-colors"
              >
                {authView === 'login' ? 'Login' : 'Create Account'}
              </button>
            </form>
          </div>

          <p className={`text-center text-sm ${darkMode ? 'text-gray-400' : 'text-gray-600'}`}>
            {/* {authView === 'login' ? "Don't have an account? " : "Already have an account? "} */}          
            <button
              onClick={() => { 
                setAuthView(authView === 'login' ? 'none' : 'login'); 
                setAuthError('');
              }}
              className="text-indigo-500 hover:text-indigo-600 font-medium"
            >
              {/* {authView === 'login' ? 'None' : 'Login'} */}
            </button>
          </p>
        </div>
      </div>
    );
  };

  if (isLoading) {
    return (
      <div className={`min-h-screen w-full flex items-center justify-center ${
        darkMode 
          ? 'bg-gradient-to-br from-gray-900 to-gray-800' 
          : 'bg-gradient-to-br from-blue-50 to-indigo-100'
      }`}>
        <div className="text-center">
          <Clock className={`w-12 h-12 mx-auto mb-4 animate-spin ${darkMode ? 'text-indigo-400' : 'text-indigo-600'}`} />
          <p className={darkMode ? 'text-gray-300' : 'text-gray-700'}>Loading...</p>
        </div>
      </div>
    );
  }

  if (!isAuthenticated) {
    return renderAuthScreen();
  }

  return (
    <div className={`min-h-screen w-full p-4 md:p-8 transition-colors ${
      darkMode 
        ? 'bg-gradient-to-br from-gray-900 to-gray-800' 
        : 'bg-gradient-to-br from-blue-50 to-indigo-100'
    }`}>
      <div className="w-full max-w-7xl mx-auto">
        <div className={`w-full rounded-2xl shadow-xl p-4 md:p-8 mb-8 transition-colors ${
          darkMode ? 'bg-gray-800' : 'bg-white'
        }`}>
          <div className="flex items-center justify-between mb-8">
            <div className="flex items-center gap-3">
              <Clock className={`w-8 h-8 ${darkMode ? 'text-indigo-400' : 'text-indigo-600'}`} />
              <h1 className={`text-3xl font-bold ${darkMode ? 'text-white' : 'text-gray-800'}`}>Timesheet Tracker</h1>
            </div>
            
            <div className="flex gap-2 items-center">
              {currentUser && (
                <div className={`px-3 py-1.5 rounded-lg ${darkMode ? 'bg-gray-700 text-gray-300' : 'bg-gray-200 text-gray-700'}`}>
                  <span className="text-sm font-medium">{currentUser.username}</span>
                </div>
              )}
              <button
                onClick={handleLogout}
                className={`p-2 rounded-lg transition-colors ${
                  darkMode 
                    ? 'bg-gray-700 hover:bg-gray-600 text-red-400' 
                    : 'bg-gray-200 hover:bg-gray-300 text-red-600'
                }`}
                title="Logout"
              >
                <LogOut className="w-5 h-5" />
              </button>
              <button
                onClick={() => setDarkMode(!darkMode)}
                className={`p-2 rounded-lg transition-colors ${
                  darkMode 
                    ? 'bg-gray-700 hover:bg-gray-600 text-yellow-400' 
                    : 'bg-gray-200 hover:bg-gray-300 text-gray-700'
                }`}
                title={darkMode ? 'Light Mode' : 'Dark Mode'}
              >
                {darkMode ? <Sun className="w-5 h-5" /> : <Moon className="w-5 h-5" />}
              </button>
              <button
                onClick={() => setView('home')}
                className={`px-4 py-2 rounded-lg flex items-center gap-2 transition-colors ${
                  view === 'home' 
                    ? 'bg-indigo-600 text-white' 
                    : darkMode 
                      ? 'bg-gray-700 text-gray-300 hover:bg-gray-600' 
                      : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
                }`}
              >
                <Clock className="w-4 h-4" />
                Home
              </button>
              <button
                onClick={() => setView('calendar')}
                className={`px-4 py-2 rounded-lg flex items-center gap-2 transition-colors ${
                  view === 'calendar' 
                    ? 'bg-indigo-600 text-white' 
                    : darkMode 
                      ? 'bg-gray-700 text-gray-300 hover:bg-gray-600' 
                      : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
                }`}
              >
                <Calendar className="w-4 h-4" />
                Calendar
              </button>
              <button
                onClick={() => setView('report')}
                className={`px-4 py-2 rounded-lg flex items-center gap-2 transition-colors ${
                  view === 'report' 
                    ? 'bg-indigo-600 text-white' 
                    : darkMode 
                      ? 'bg-gray-700 text-gray-300 hover:bg-gray-600' 
                      : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
                }`}
              >
                <BarChart3 className="w-4 h-4" />
                Report
              </button>
            </div>
          </div>

          {view === 'home' && renderHome()}

          {view === 'calendar' && (
            <div>
              {!activeProject ? (
                <div className={`rounded-xl p-8 mb-6 border-2 border-dashed ${
                  darkMode ? 'bg-gray-800 border-gray-600' : 'bg-gray-50 border-gray-300'
                }`}>
                  <p className={`text-center ${darkMode ? 'text-gray-400' : 'text-gray-500'}`}>
                    Please select an active project from the Home screen to track work days.
                  </p>
                  <button
                    onClick={() => setView('home')}
                    className="mt-4 mx-auto block px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg transition-colors"
                  >
                    Go to Home
                  </button>
                </div>
              ) : (
                <>
                  <div className={`rounded-lg p-4 mb-6 ${darkMode ? 'bg-indigo-900' : 'bg-indigo-50'}`}>
                    <div className="flex items-center justify-between">
                      <div>
                        <p className={`text-sm ${darkMode ? 'text-indigo-300' : 'text-indigo-600'}`}>
                          Tracking for
                        </p>
                        <p className={`text-lg font-semibold ${darkMode ? 'text-white' : 'text-gray-900'}`}>
                          {activeProject.name}
                        </p>
                      </div>
                      <span className={`px-2 py-1 rounded text-xs font-semibold ${
                        darkMode ? 'bg-green-900 text-green-300' : 'bg-green-100 text-green-700'
                      }`}>
                        Active
                      </span>
                    </div>
                  </div>
              <div className="flex items-center justify-between mb-6">
                <button
                  onClick={() => setCurrentMonth(new Date(currentMonth.getFullYear(), currentMonth.getMonth() - 1))}
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
                  onClick={() => setCurrentMonth(new Date(currentMonth.getFullYear(), currentMonth.getMonth() + 1))}
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
                  💡 Click on any weekday to add or edit work hours. Total hours are displayed after saving.
                </p>
              </div>
                </>
              )}
            </div>
          )}

          {view === 'report' && renderReport()}
        </div>
      </div>

      {renderWorkDayModal()}

      {showAddProject && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50" onClick={() => setShowAddProject(false)}>
          <div className={`${darkMode ? 'bg-gray-800' : 'bg-white'} rounded-xl p-6 max-w-md w-full mx-4`} onClick={(e) => e.stopPropagation()}>
            <h3 className={`text-xl font-bold mb-4 ${darkMode ? 'text-white' : 'text-gray-900'}`}>
              Create New Project
            </h3>
            
            <div className="space-y-4">
              <div>
                <label className={`block text-sm font-medium mb-1 ${darkMode ? 'text-gray-300' : 'text-gray-700'}`}>
                  Project Name *
                </label>
                <input
                  type="text"
                  placeholder="e.g., Website Redesign"
                  value={newProjectName}
                  onChange={(e) => setNewProjectName(e.target.value)}
                  onKeyPress={(e) => e.key === 'Enter' && addProject()}
                  autoFocus
                  className={`w-full px-3 py-2 border rounded-lg ${
                    darkMode 
                      ? 'bg-gray-700 border-gray-600 text-white placeholder-gray-400' 
                      : 'bg-white border-gray-300 text-gray-900'
                  }`}
                />
              </div>

              <div>
                <label className={`block text-sm font-medium mb-1 ${darkMode ? 'text-gray-300' : 'text-gray-700'}`}>
                  Description (optional)
                </label>
                <textarea
                  placeholder="Brief description of the project"
                  value={newProjectDesc}
                  onChange={(e) => setNewProjectDesc(e.target.value)}
                  rows={3}
                  className={`w-full px-3 py-2 border rounded-lg ${
                    darkMode 
                      ? 'bg-gray-700 border-gray-600 text-white placeholder-gray-400' 
                      : 'bg-white border-gray-300 text-gray-900'
                  }`}
                />
              </div>

              <div className="flex gap-2">
                <button
                  onClick={addProject}
                  disabled={!newProjectName.trim()}
                  className="flex-1 bg-indigo-600 hover:bg-indigo-700 disabled:bg-gray-600 disabled:cursor-not-allowed text-white px-4 py-2 rounded-lg transition-colors"
                >
                  Create Project
                </button>
                <button
                  onClick={() => {
                    setShowAddProject(false);
                    setNewProjectName('');
                    setNewProjectDesc('');
                  }}
                  className={`px-4 py-2 rounded-lg transition-colors ${
                    darkMode 
                      ? 'bg-gray-700 hover:bg-gray-600 text-gray-200' 
                      : 'bg-gray-300 hover:bg-gray-400 text-gray-800'
                  }`}
                >
                  Cancel
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}