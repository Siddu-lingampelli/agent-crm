import { useEffect, useId, useRef, type ReactNode, type ButtonHTMLAttributes } from 'react';
import { Link } from 'react-router-dom';
import { ArrowRight, ChevronLeft, ChevronRight, CircleAlert, GitBranch, Loader2, Plug, RefreshCw, Search, X } from 'lucide-react';
import './Automation.css';

export function AutomationHeader({ section, description, children }: { section: 'Workflows' | 'Integrations'; description: string; children?: ReactNode }) {
  return <><nav className="au-section-nav" aria-label="Automation pages"><Link to="/workflows" aria-current={section === 'Workflows' ? 'page' : undefined}><GitBranch size={15} />Workflows</Link><Link to="/integrations" aria-current={section === 'Integrations' ? 'page' : undefined}><Plug size={15} />Integrations</Link></nav><header className="au-header"><div><span className="au-eyebrow">AUTOMATION</span><h1>{section}</h1><p>{description}</p></div><div className="au-header-actions">{children}</div></header></>;
}
export function AutomationButton({ primary, children, className = '', ...props }: ButtonHTMLAttributes<HTMLButtonElement> & { primary?: boolean }) {
  return <button type="button" className={`au-button${primary ? ' au-button-primary' : ''} ${className}`} {...props}>{children}</button>;
}
export function RefreshButton({ loading, onClick }: { loading: boolean; onClick: () => void }) { return <AutomationButton onClick={onClick} disabled={loading} aria-label="Refresh records"><RefreshCw size={15} className={loading ? 'au-spin' : ''} /><span>Refresh</span></AutomationButton>; }
export function AutomationStatus({ status }: { status: string }) {
  return <span className={`au-status au-status-${status}`}><i />{status.replaceAll('_', ' ')}</span>;
}
export function AutomationMessage({ message, error = false, onClose, onRetry }: { message: string; error?: boolean; onClose?: () => void; onRetry?: () => void }) {
  return <div className={`au-message${error ? ' is-error' : ''}`} role={error ? 'alert' : 'status'}><CircleAlert size={16} /><span>{message}</span>{onRetry && <button onClick={onRetry}>Try again</button>}{onClose && <button onClick={onClose} aria-label="Dismiss message"><X size={15} /></button>}</div>;
}
export function AutomationLoading() { return <div className="au-loading" role="status"><Loader2 size={20} className="au-spin" />Loading records</div>; }
export function AutomationEmpty({ title, description, children }: { title: string; description: string; children?: ReactNode }) {
  return <div className="au-empty"><svg viewBox="0 0 210 65" fill="none" aria-hidden="true"><rect x="1" y="10" width="49" height="45" fill="#ff4f00" stroke="#000" /><path d="M16 24h19M16 32h19M16 40h12M51 32h29m-6-6 6 6-6 6" stroke="#000" /><rect x="81" y="10" width="49" height="45" stroke="#000" /><path d="M97 25h17v15H97zM131 32h29m-6-6 6 6-6 6" stroke="#000" /><circle cx="185" cy="32" r="23" stroke="#000" /><path d="m175 32 7 7 12-14" stroke="#000" /></svg><h3>{title}</h3><p>{description}</p>{children}</div>;
}
export function AutomationSearch({ value, onChange, placeholder }: { value: string; onChange: (value: string) => void; placeholder: string }) {
  return <label className="au-search"><Search size={16} /><span className="au-sr-only">{placeholder}</span><input type="search" value={value} onChange={event => onChange(event.target.value)} placeholder={placeholder} /></label>;
}
export function AutomationPagination({ page, size, total, onPage, onSize }: { page: number; size: number; total: number; onPage: (page: number) => void; onSize: (size: number) => void }) {
  const pages = Math.max(1, Math.ceil(total / size));
  return <div className="au-pagination"><span>{total ? (page - 1) * size >= total ? `0 of ${total}` : `${(page - 1) * size + 1}\u2013${Math.min(page * size, total)} of ${total}` : '0 records'}</span><div><label>Rows<select value={size} onChange={event => onSize(Number(event.target.value))}>{[12,25,50].map(value => <option key={value}>{value}</option>)}</select></label><button onClick={() => onPage(page - 1)} disabled={page <= 1} aria-label="Previous page"><ChevronLeft size={16} /></button><span>{page} / {pages}</span><button onClick={() => onPage(page + 1)} disabled={page >= pages} aria-label="Next page"><ChevronRight size={16} /></button></div></div>;
}
export function AutomationDialog({ title, description, onClose, busy = false, children, wide = false }: { title: string; description?: string; onClose: () => void; busy?: boolean; children: ReactNode; wide?: boolean }) {
  const ref = useRef<HTMLDialogElement>(null);
  const id = useId();
  useEffect(() => { const dialog = ref.current; dialog?.showModal(); return () => dialog?.close(); }, []);
  return <dialog ref={ref} className={`au-dialog${wide ? ' au-dialog-wide' : ''}`} aria-labelledby={id} onCancel={event => { event.preventDefault(); if (!busy) onClose(); }}><div className="au-dialog-heading"><div><h2 id={id}>{title}</h2>{description && <p>{description}</p>}</div><button onClick={onClose} disabled={busy} aria-label="Close dialog"><X size={19} /></button></div>{children}</dialog>;
}
export function AutomationLink({ to, children }: { to: string; children: ReactNode }) { return <Link className="au-text-link" to={to}>{children}<ArrowRight size={14} /></Link>; }
