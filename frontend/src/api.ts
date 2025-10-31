import { TimeEntry, WorkDay, ProjectStat } from "./types";

const API_URL = "http://localhost:8080/api";

export const api = {
  // Time Entries
  async getEntries(): Promise<TimeEntry[]> {
    const res = await fetch(`${API_URL}/entries`);
    return res.json();
  },

  async getActiveEntry(): Promise<TimeEntry | null> {
    const res = await fetch(`${API_URL}/entries/active`);
    return res.json();
  },

  async createEntry(entry: Partial<TimeEntry>): Promise<TimeEntry> {
    const res = await fetch(`${API_URL}/entries`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(entry),
    });
    return res.json();
  },

  async updateEntry(id: number, entry: Partial<TimeEntry>): Promise<TimeEntry> {
    const res = await fetch(`${API_URL}/entries/${id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(entry),
    });
    return res.json();
  },

  async deleteEntry(id: number): Promise<void> {
    await fetch(`${API_URL}/entries/${id}`, {
      method: "DELETE",
    });
  },

  // Statistics
  async getStats(): Promise<ProjectStat[]> {
    const res = await fetch(`${API_URL}/stats`);
    return res.json();
  },

  // Work Days
  async getWorkDays(year: number, month: number): Promise<WorkDay[]> {
    const res = await fetch(`${API_URL}/workdays/${year}/${month}`);
    return res.json();
  },

  async createWorkDay(workDay: WorkDay): Promise<WorkDay> {
    const res = await fetch(`${API_URL}/workdays`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(workDay),
    });
    return res.json();
  },

  async updateWorkDay(id: number, workDay: WorkDay): Promise<WorkDay> {
    const res = await fetch(`${API_URL}/workdays/${id}`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(workDay),
    });
    return res.json();
  },

  async deleteWorkDay(id: number): Promise<void> {
    await fetch(`${API_URL}/workdays/${id}`, {
      method: "DELETE",
    });
  },
};
