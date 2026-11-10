import { useCallback, useEffect, useEffectEvent, useState } from 'react';

export interface AdminPage<T> { items: T[]; total: number; limit?: number; offset?: number }
export interface HealthCheck { id: string; name: string; status: 'healthy' | 'degraded' | 'down'; message: string; latency_ms: number; timestamp: string }
export interface ProcessMemory { pid: number; resident_bytes: number; peak_resident_bytes: number; sampled_at: string }
export interface RuntimeMetrics { memory_usage: number; memory_total: number; goroutines: number; db_connections: number; db_idle: number; uptime_seconds: number; process_memory?: ProcessMemory | null }
export interface QueueStats { name: string; workers: number; queue_size: number; closed: boolean; pending: number; processing: number; completed: number; failed: number; throughput_per_min?: number }
export interface AuditRecord { id: string; actor_type: string; actor_id: string; action: string; resource_type: string; resource_id: string; old_values: string; new_values: string; ip_address: string; user_agent: string; created_at: string }
export interface SchemaColumn { name: string; type: string; nullable: boolean; default_value?: string; primary_key: boolean }
export interface SchemaTable { id: string; name: string; type: string; columns: SchemaColumn[]; indexes: { name: string }[] }
export interface Migration { id: string; name: string; applied_at: string }
export interface AccountIdentity { id: string; email: string; workspace_id: string }

export function useAdminData<T>(load: () => Promise<T>, key: string) {
  const [revision, setRevision] = useState(0);
  const requestKey = `${key}:${revision}`;
  const [state, setState] = useState<{ key: string; baseKey: string; data: T | null; error: string; updatedAt: Date | null }>({ key: '', baseKey: '', data: null, error: '', updatedAt: null });
  const loadCurrent = useEffectEvent(load);
  useEffect(() => {
    const controller = new AbortController();
    loadCurrent().then(data => {
      if (!controller.signal.aborted) setState({ key: requestKey, baseKey: key, data, error: '', updatedAt: new Date() });
    }).catch(error => {
      if (!controller.signal.aborted) setState(previous => ({ key: requestKey, baseKey: key, data: previous.baseKey === key ? previous.data : null, updatedAt: previous.baseKey === key ? previous.updatedAt : null, error: error instanceof Error ? error.message : 'Could not load this section.' }));
    });
    return () => controller.abort();
  }, [requestKey, key]);
  const refresh = useCallback(() => setRevision(value => value + 1), []);
  return { data: state.baseKey === key ? state.data : null, error: state.key === requestKey ? state.error : '', loading: state.key !== requestKey, updatedAt: state.baseKey === key ? state.updatedAt : null, refresh };
}

export async function loadHealth(): Promise<{ checks: HealthCheck[] }> {
  const response = await fetch('/api/v1/admin/health', { credentials: 'include' });
  const data = await response.json();
  if ((response.ok || response.status === 503) && Array.isArray(data.checks)) return data;
  throw new Error(response.status === 401 ? 'Your session has expired. Sign in again.' : 'Could not read system health.');
}
export function bytes(value: number | undefined) {
  if (typeof value !== 'number' || !Number.isFinite(value) || value < 0) return '—';
  if (value === 0) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB', 'TB'];
  const exponent = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  return `${(value / 1024 ** exponent).toFixed(exponent === 0 ? 0 : 1)} ${units[exponent]}`;
}
export function uptime(seconds: number | undefined) {
  if (typeof seconds !== 'number' || !Number.isFinite(seconds)) return '—';
  const minutes = Math.floor(Math.max(0, seconds) / 60);
  if (minutes < 1) return '< 1 min';
  if (minutes < 60) return `${minutes} min`;
  const hours = Math.floor(minutes / 60);
  return hours < 24 ? `${hours}h ${minutes % 60}m` : `${Math.floor(hours / 24)}d ${hours % 24}h`;
}
export function adminDate(value: string | null | undefined) {
  if (!value) return '—';
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? '—' : date.toLocaleString('en-US', { month: 'short', day: 'numeric', year: 'numeric', hour: 'numeric', minute: '2-digit' });
}
export function healthState(checks: HealthCheck[] | undefined) {
  if (!checks?.length) return 'unknown';
  return checks.some(check => check.status === 'down') ? 'down' : checks.some(check => check.status === 'degraded') ? 'degraded' : checks.every(check => check.status === 'healthy') ? 'healthy' : 'unknown';
}
export function auditJson(value: string | null | undefined) {
  if (!value) return 'No values recorded';
  try { return JSON.stringify(JSON.parse(value), null, 2); } catch { return value; }
}
export function auditCsv(records: AuditRecord[]) {
  const cell = (value: unknown) => {
    let text = String(value ?? '');
    const start = text.trimStart();
    if (/^[=+@-]/.test(start) || start.length > 0 && start.charCodeAt(0) < 32) text = `'${text}`;
    return `"${text.replaceAll('"', '""')}"`;
  };
  const rows = [['Timestamp', 'Event ID', 'Action', 'Resource type', 'Resource ID', 'Actor type', 'Actor ID', 'IP address'], ...records.map(record => [record.created_at, record.id, record.action, record.resource_type, record.resource_id, record.actor_type, record.actor_id, record.ip_address])];
  return '\ufeff' + rows.map(row => row.map(cell).join(',')).join('\r\n');
}
export function downloadText(name: string, text: string, type: string) {
  const url = URL.createObjectURL(new Blob([text], { type }));
  const link = document.createElement('a'); link.href = url; link.download = name;
  document.body.append(link); link.click(); link.remove();
  window.setTimeout(() => URL.revokeObjectURL(url), 1000);
}
