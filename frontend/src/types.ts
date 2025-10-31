export interface TimeEntry {
  id: number;
  project_name: string;
  description: string;
  start_time: string;
  end_time: string | null;
  duration: number | null;
  created_at: string;
}

export interface WorkDay {
  id?: number;
  date: string;
  start_time: string | null;
  end_time: string | null;
  break_hours: number;
  total_hours: number | null;
}

export interface ProjectStat {
  project_name: string;
  entry_count: number;
  total_hours: number;
}
