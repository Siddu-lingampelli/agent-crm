import type { ReactNode } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { ArrowUpRight, CircleAlert, Loader2 } from 'lucide-react';
import { adminSections } from './admin-navigation';
import './Administration.css';

export function AdminArea({ children }: { children: ReactNode }) {
  const { pathname } = useLocation();
  return <div className="admin-area"><nav className="ad-navigation" aria-label="Administration sections">{adminSections.map(section => {
    const active = pathname === section.href;
    return <Link key={section.href} to={section.href} className={active ? 'is-active' : ''} aria-current={active ? 'page' : undefined}><section.icon size={14} /><span>{section.title}</span></Link>;
  })}</nav>{children}</div>;
}
export function AdminTitle({ title, description }: { title: string; description: string }) {
  return <div className="ad-title"><span className="ad-eyebrow">WORKSPACE ADMINISTRATION</span><h1>{title}</h1><p>{description}</p></div>;
}
export function AdminHeader({ title, description, actions }: { title: string; description: string; actions?: ReactNode }) {
  return <header className="ad-page-header"><AdminTitle title={title} description={description} />{actions && <div className="ad-header-actions">{actions}</div>}</header>;
}
export function AdminPanel({ title, description, action, children, className = '' }: { title: string; description?: string; action?: ReactNode; children: ReactNode; className?: string }) {
  return <section className={`ad-panel ${className}`}><div className="ad-panel-header"><div><h2>{title}</h2>{description && <p>{description}</p>}</div>{action}</div>{children}</section>;
}
export function AdminLink({ to, children }: { to: string; children: ReactNode }) { return <Link className="ad-text-link" to={to}>{children}<ArrowUpRight size={12} /></Link>; }
export function AdminStatus({ status }: { status: string }) {
  const names: Record<string, string> = { healthy: 'Healthy', down: 'Unavailable', degraded: 'Needs attention', unknown: 'Not available', enabled: 'Enabled', disabled: 'Disabled' };
  return <span className={`ad-status ad-status-${status}`}><i />{names[status] || status}</span>;
}
export function AdminError({ message, onRetry }: { message: string; onRetry: () => void }) {
  return <div className="ad-error" role="alert"><CircleAlert size={16} /><span>{message}</span><button onClick={onRetry}>Try again</button></div>;
}
export function AdminLoading({ label = 'Loading administration data' }: { label?: string }) {
  return <div className="ad-loading" role="status"><Loader2 size={19} className="loading-spinner" />{label}</div>;
}
export function AdminEmpty({ icon, title, children }: { icon?: ReactNode; title: string; children: ReactNode }) {
  return <div className="ad-empty">{icon && <span>{icon}</span>}<h3>{title}</h3><p>{children}</p></div>;
}
