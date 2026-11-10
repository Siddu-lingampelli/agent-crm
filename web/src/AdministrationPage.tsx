import { Link } from 'react-router-dom';
import { Activity, ArrowRight, ArrowUpRight, Bot, Clock3, Database, KeyRound, RefreshCw, ScrollText, Server, Settings2, ShieldCheck, Webhook } from 'lucide-react';
import { api } from './api';
import { adminSections } from './admin-navigation';
import { AdminHeader, AdminPanel, AdminLink, AdminLoading, AdminError, AdminEmpty, AdminStatus } from './AdminArea';
import { useAdminData, loadHealth, healthState, adminDate, bytes, uptime, type AdminPage, type AuditRecord, type RuntimeMetrics } from './admin-data';

type Counted = AdminPage<{ id: string }>;
export function AdministrationPage() {
  const health = useAdminData(loadHealth, 'admin-overview-health');
  const metrics = useAdminData(() => api.get<RuntimeMetrics>('/admin/metrics'), 'admin-overview-metrics');
  const agents = useAdminData(() => api.get<Counted>('/agents', { params: { limit: 1 } }), 'admin-overview-agents');
  const keys = useAdminData(() => api.get<Counted>('/api-keys', { params: { limit: 1 } }), 'admin-overview-keys');
  const webhooks = useAdminData(() => api.get<Counted>('/webhooks', { params: { limit: 1 } }), 'admin-overview-webhooks');
  const audit = useAdminData(() => api.get<AdminPage<AuditRecord>>('/admin/audit', { params: { limit: 5, sort: 'created_at', order: 'desc' } }), 'admin-overview-audit');
  const resources = [health, metrics, agents, keys, webhooks, audit];
  const loading = resources.some(resource => resource.loading);
  const refresh = () => resources.forEach(resource => resource.refresh());
  const state = healthState(health.data?.checks);
  const cards = [
    { label: 'Agents', value: agents.data?.total, note: 'Workspace automation access', href: '/agents?tab=agents', icon: Bot, error: agents.error },
    { label: 'API keys', value: keys.data?.total, note: 'Issued workspace credentials', href: '/agents', icon: KeyRound, error: keys.error },
    { label: 'Webhooks', value: webhooks.data?.total, note: 'Configured event connections', href: '/webhooks', icon: Webhook, error: webhooks.error },
  ];
  return <div className="ad-page"><AdminHeader title="Administration" description="A clear view of your workspace, access, and system operations." actions={<><Link className="ad-button" to="/settings"><Settings2 size={14} />Settings</Link><button className="ad-button ad-button-primary" onClick={refresh} disabled={loading}><RefreshCw size={14} className={loading ? 'loading-spinner' : ''} />Refresh</button></>} />
    {resources.some(resource => resource.error) && <AdminError message="Some administration data could not be refreshed. Available data is kept below." onRetry={refresh} />}
    <div className="ad-overview-metrics"><Link to="/system" className="ad-overview-stat"><div><span>System status</span><Activity size={15} /></div><strong className="ad-health-value">{health.loading && !health.data ? 'Checking...' : state === 'healthy' ? 'Operational' : state === 'unknown' ? 'Unavailable' : 'Needs attention'}</strong><span><i className={state === 'healthy' ? 'ad-live-dot' : 'ad-neutral-dot'} />{health.data ? `${health.data.checks.length} service checks` : 'Waiting for health data'}<ArrowUpRight size={12} /></span></Link>{cards.map(card => <Link key={card.label} to={card.href} className="ad-overview-stat"><div><span>{card.label}</span><card.icon size={15} /></div><strong>{card.value === undefined ? '—' : card.value.toLocaleString()}</strong><span>{card.error ? 'Could not refresh count' : card.note}<ArrowUpRight size={12} /></span></Link>)}</div>
    <div className="ad-overview-grid"><div className="ad-overview-main"><AdminPanel title="Manage your workspace" description="The controls you need, organized by purpose."><div className="ad-destination-grid">{adminSections.filter(section => section.href !== '/admin').map(section => <Link key={section.href} to={section.href} className="ad-destination"><span className="ad-destination-icon"><section.icon size={18} /></span><div><h3>{section.title}</h3><p>{section.description}</p></div><ArrowUpRight size={14} /></Link>)}</div></AdminPanel>
    <AdminPanel title="Recent recorded changes" description="Latest audit events in this workspace" action={<AdminLink to="/audit">View audit log</AdminLink>}>{audit.loading && !audit.data ? <AdminLoading /> : audit.error && !audit.data ? <AdminError message={audit.error} onRetry={audit.refresh} /> : !audit.data?.items.length ? <AdminEmpty icon={<ScrollText size={24} />} title="No recorded changes yet.">Workspace activity will appear here as records are changed.</AdminEmpty> : <div className="ad-recent-events">{audit.data.items.map(record => <Link className="ad-recent-event" key={record.id} to="/audit"><span className="ad-event-icon"><ScrollText size={14} /></span><div><strong><code>{record.action.toUpperCase()}</code>{record.resource_type.replace(/_/g, ' ')}</strong><p>{record.actor_type === 'agent' ? 'Agent' : 'User'} <code>{record.actor_id.slice(0, 12)}...</code></p></div><time dateTime={record.created_at}>{adminDate(record.created_at)}</time><ArrowUpRight size={12} /></Link>)}</div>}</AdminPanel></div>
    <div className="ad-overview-side"><AdminPanel title="Service health" action={<AdminLink to="/system">Details</AdminLink>}>{health.loading && !health.data ? <AdminLoading /> : health.error && !health.data ? <AdminError message={health.error} onRetry={health.refresh} /> : <div className="ad-service-list">{health.data?.checks.map(check => <div key={check.id || check.name}><span className="ad-service-icon">{check.name === 'database' ? <Database size={16} /> : <Server size={16} />}</span><div><strong>{check.name === 'jobs' ? 'Background jobs' : check.name === 'database' ? 'Database' : check.name}</strong><small>{check.message}</small></div><AdminStatus status={check.status} /></div>)}</div>}<div className="ad-panel-footer">{health.updatedAt ? `Checked ${health.updatedAt.toLocaleTimeString('en-US', {hour:'numeric',minute:'2-digit'})}` : 'Health check pending'}</div></AdminPanel>
    <AdminPanel title="Runtime snapshot" description="Current application process"><dl className="ad-key-values"><div><dt><Clock3 size={13} />Uptime</dt><dd>{uptime(metrics.data?.uptime_seconds)}</dd></div><div><dt>Heap allocated</dt><dd>{bytes(metrics.data?.memory_usage)}</dd></div><div><dt>Database connections</dt><dd>{metrics.data?.db_connections ?? '—'}</dd></div><div><dt>Goroutines</dt><dd>{metrics.data?.goroutines ?? '—'}</dd></div></dl>{metrics.error && <AdminError message={metrics.error} onRetry={metrics.refresh} />}</AdminPanel>
    <div className="ad-admin-note"><ShieldCheck size={20} /><h3>Keep access intentional.</h3><p>Review agent scopes and API keys as your workspace evolves.</p><AdminLink to="/agents">Review access</AdminLink></div></div></div>
    <footer className="ad-footer"><span><ShieldCheck size={12} />Administration for your current workspace</span><Link to="/agent-console">Open Agent Console<ArrowRight size={12} /></Link></footer>
  </div>;
}
