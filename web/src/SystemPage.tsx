import { useEffect, useState } from 'react';
import { Activity, Clock3, Database, Layers, MemoryStick, RefreshCw, Server, ShieldCheck } from 'lucide-react';
import { api } from './api';
import { AdminHeader, AdminPanel, AdminLoading, AdminError, AdminStatus, AdminEmpty } from './AdminArea';
import { useAdminData, loadHealth, bytes, uptime, adminDate, healthState, type RuntimeMetrics, type QueueStats, type ProcessMemory } from './admin-data';

export function ProcessMemoryPanel({ memory, loading }: { memory?: ProcessMemory | null; loading: boolean }) {
  return <AdminPanel title="Application memory" description="RAM used by the CRM backend process, including runtime and native allocations">
    {loading && !memory ? <AdminLoading label="Reading application memory" /> : !memory ? <AdminEmpty icon={<MemoryStick size={24} />} title="Process RAM data is unavailable.">Refresh to try again. Go runtime memory is available below.</AdminEmpty> : <div className="ad-ram-panel">
      <div className="ad-ram-summary"><div><strong>{bytes(memory.resident_bytes)}</strong><span>current process RAM</span></div><span className="ad-ram-total">CRM backend</span></div>
      <dl className="ad-ram-values"><div><dt>Current RAM</dt><dd>{bytes(memory.resident_bytes)}</dd></div><div><dt>Peak RAM</dt><dd>{bytes(memory.peak_resident_bytes)}</dd></div><div><dt>Process ID</dt><dd>{memory.pid}</dd></div></dl>
      <div className="ad-ram-sampled">Sampled {adminDate(memory.sampled_at)}. Peak is measured since this process started.</div>
    </div>}
  </AdminPanel>;
}

export function SystemPage() {
  const health = useAdminData(loadHealth, 'system-health');
  const metrics = useAdminData(() => api.get<RuntimeMetrics>('/admin/metrics'), 'system-metrics');
  const queues = useAdminData(() => api.get<{ queues: QueueStats[] }>('/admin/queues'), 'system-queues');
  const [autoRefresh, setAutoRefresh] = useState(true);
  const loading = health.loading || metrics.loading || queues.loading;
  const refreshHealth = health.refresh, refreshMetrics = metrics.refresh, refreshQueues = queues.refresh;
  const refresh = () => { refreshHealth(); refreshMetrics(); refreshQueues(); };
  useEffect(() => {
    if (!autoRefresh) return;
    const timer = window.setInterval(() => { if (!document.hidden) { refreshHealth(); refreshMetrics(); refreshQueues(); } }, 30000);
    return () => window.clearInterval(timer);
  }, [autoRefresh, refreshHealth, refreshMetrics, refreshQueues]);
  const values = [
    { label: 'Application RAM', value: bytes(metrics.data?.process_memory?.resident_bytes), note: metrics.data?.process_memory ? 'CRM backend process' : metrics.data ? 'Process RAM unavailable' : 'Reading process memory', icon: MemoryStick },
    { label: 'Database connections', value: metrics.data?.db_connections ?? '—', note: `${metrics.data?.db_idle ?? '—'} idle connections`, icon: Database },
    { label: 'Goroutines', value: metrics.data?.goroutines ?? '—', note: 'Active runtime goroutines', icon: Layers },
    { label: 'Uptime', value: uptime(metrics.data?.uptime_seconds), note: 'Since the process started', icon: Clock3 },
  ];
  return <div className="ad-page"><AdminHeader title="System health" description="Monitor the services and runtime behind your CRM." actions={<><label className="ad-checkbox"><input type="checkbox" checked={autoRefresh} onChange={event => setAutoRefresh(event.target.checked)} />Auto-refresh · 30s</label><button className="ad-button ad-button-primary" onClick={refresh} disabled={loading}><RefreshCw size={14} className={loading ? 'loading-spinner' : ''} />Refresh</button></>} />
    <div className="ad-health-banner"><span><ShieldCheck size={17} /><strong>Service status</strong></span><AdminStatus status={healthState(health.data?.checks)} /><small>{health.updatedAt ? `Checked ${health.updatedAt.toLocaleTimeString()}` : 'Checking services...'}</small></div>
    {[health,metrics,queues].filter(resource => resource.error).map((resource,index) => <AdminError key={index} message={resource.error} onRetry={resource.refresh} />)}
    <div className="ad-overview-metrics">{values.map(value => <div className="ad-overview-stat" key={value.label}><div><span>{value.label}</span><value.icon size={15} /></div><strong>{value.value}</strong><span>{value.note}</span></div>)}</div>
    <div className="ad-ram-section"><ProcessMemoryPanel memory={metrics.data?.process_memory} loading={metrics.loading} /></div>
    <div className="ad-two-columns"><AdminPanel title="Service checks" description="Availability reported by the application">{health.loading && !health.data ? <AdminLoading /> : <div className="ad-service-list">{health.data?.checks.map(check => <div key={check.id || check.name}><span className="ad-service-icon">{check.name === 'database' ? <Database size={17} /> : <Server size={17} />}</span><div><strong>{check.name === 'database' ? 'Database' : check.name === 'jobs' ? 'Background jobs' : check.name}</strong><small>{check.message}</small><small>{adminDate(check.timestamp)} · {check.latency_ms.toFixed(2)} ms</small></div><AdminStatus status={check.status} /></div>)}</div>}</AdminPanel><AdminPanel title="Runtime & database" description="Measured process values"><dl className="ad-key-values"><div><dt>Memory obtained by runtime</dt><dd>{bytes(metrics.data?.memory_total)}</dd></div><div><dt>Heap allocated</dt><dd>{bytes(metrics.data?.memory_usage)}</dd></div><div><dt>Open database connections</dt><dd>{metrics.data?.db_connections ?? '—'}</dd></div><div><dt>Connections in use</dt><dd>{metrics.data ? Math.max(0, metrics.data.db_connections - metrics.data.db_idle) : '—'}</dd></div><div><dt>Idle connections</dt><dd>{metrics.data?.db_idle ?? '—'}</dd></div></dl></AdminPanel></div>
    <AdminPanel title="Background jobs" description="Queue capacity, activity, and processing outcomes">{queues.loading && !queues.data ? <AdminLoading /> : !queues.data?.queues.length ? <AdminEmpty icon={<Activity size={24} />} title="No queue data available.">Refresh this page to check for background job information.</AdminEmpty> : <div className="ad-table-scroll"><table className="ad-table"><thead><tr><th scope="col">Queue</th><th scope="col">Status</th><th scope="col">Workers</th><th scope="col">Pending</th><th scope="col">Processing</th><th scope="col">Completed</th><th scope="col">Failed</th><th scope="col">Throughput / min</th></tr></thead><tbody>{queues.data.queues.map(queue => <tr key={queue.name}><td><strong>{queue.name}</strong><small>Capacity: {queue.queue_size}</small></td><td><AdminStatus status={queue.closed ? 'down' : 'healthy'} /></td><td>{queue.workers}</td><td>{queue.pending}</td><td>{queue.processing}</td><td>{queue.completed}</td><td><span className={queue.failed > 0 ? 'ad-failure-count' : ''}>{queue.failed}</span></td><td>{typeof queue.throughput_per_min === 'number' ? queue.throughput_per_min.toFixed(1) : '—'}</td></tr>)}</tbody></table></div>}</AdminPanel>
    <p className="ad-footnote">Memory figures cover the CRM backend process. Go runtime memory is a separate view of the same process and should not be added to process RAM. Automatic refresh pauses while this tab is hidden.</p>
  </div>;
}
