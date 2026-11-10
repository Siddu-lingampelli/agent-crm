import type { Deal, PipelineStageMetric, Task } from './api';

export const stageColors = ['#ff4f00', '#ff9c66', '#d6c3ad', '#6e9b75', '#b7ab9e'];

export function money(value: number | null | undefined, currency = 'USD') {
  try { return new Intl.NumberFormat('en-US', { style: 'currency', currency, maximumFractionDigits: 0 }).format(value ?? 0); }
  catch { return `${Number(value ?? 0).toLocaleString('en-US')} ${currency}`; }
}

export function dayKey(value: string) { return /^\d{4}-\d{2}-\d{2}/.test(value) ? value.slice(0, 10) : ''; }

export function shortDate(value: string) {
  const key = dayKey(value);
  if (!key) return 'No date set';
  const date = new Date(`${key}T12:00:00`);
  return Number.isNaN(date.getTime()) ? 'No date set' : date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
}

export function openTasks(tasks: Task[]) {
  return tasks.filter(task => !['done', 'completed', 'cancelled'].includes(task.status) && !task.archived_at)
    .sort((a, b) => (dayKey(a.due_date) || '9999').localeCompare(dayKey(b.due_date) || '9999') || a.title.localeCompare(b.title));
}

export function stageFor(deal: Deal, stages: PipelineStageMetric[]) {
  const stage = stages.find(item => item.stage_id === deal.stage_id);
  return stage?.stage_name || deal.stage_id.replace(/[_-]/g, ' ') || 'Unassigned';
}

export function recordUrl(type: string, id: string) {
  const paths: Record<string, string> = { company: 'companies', companies: 'companies', contact: 'contacts', contacts: 'contacts', lead: 'leads', leads: 'leads', deal: 'deals', deals: 'deals', task: 'tasks', tasks: 'tasks', project: 'projects', projects: 'projects' };
  return paths[type] ? `/${paths[type]}?record=${encodeURIComponent(id)}` : null;
}
