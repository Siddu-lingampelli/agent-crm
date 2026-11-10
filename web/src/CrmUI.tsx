import { useEffect, useId, useRef, type ReactNode } from 'react';
import { NavLink } from 'react-router-dom';
import { ArrowDownToLine, ArrowRight, Building2, ChevronLeft, ChevronRight, Contact, DollarSign, Loader2, Plus, Users, X } from 'lucide-react';
import './Crm.css';
import type { Company, Contact as ContactRecord } from './api';
import { Badge, getStatusBadge } from './components';
import { Briefcase, Globe, Mail, Pencil, Phone, Trash2 } from 'lucide-react';

export function CrmHeader({ title, description, onCreate, onExport, busy }: { title: string; description: string; onCreate: () => void; onExport: () => void; busy: boolean }) {
  const singular = title === 'Companies' ? 'company' : title.slice(0, -1).toLowerCase();
  return <><nav className="crm-nav" aria-label="Sales navigation">{[{ to: '/leads', name: 'Leads', icon: Users }, { to: '/deals', name: 'Deals', icon: DollarSign }, { to: '/contacts', name: 'Contacts', icon: Contact }, { to: '/companies', name: 'Companies', icon: Building2 }].map(({ to, name, icon: Icon }) => <NavLink key={to} to={to}><Icon size={14} />{name}</NavLink>)}</nav><header className="crm-heading"><div><span className="crm-eyebrow">WORKSPACE / RELATIONSHIPS</span><h1>{title}</h1><p>{description}</p></div><div className="crm-heading-actions"><button className="crm-button" onClick={onExport} disabled={busy}><ArrowDownToLine size={14} />Export page</button><button className="crm-button crm-primary" onClick={onCreate}><Plus size={15} />New {singular}</button></div></header></>;
}

export function CrmSummary({ items, loading }: { items: { label: string; value: ReactNode; note: string }[]; loading: boolean }) {
  return <div className="crm-summary" aria-busy={loading}>{items.map(item => <div key={item.label}><span>{item.label}</span><strong>{loading ? '\u2014' : item.value}</strong><small>{item.note}</small></div>)}</div>;
}

export function CrmState({ loading, error, empty, onRetry, onCreate, noun, filtered, children }: { loading: boolean; error: string; empty: boolean; onRetry: () => void; onCreate: () => void; noun: string; filtered: boolean; children: ReactNode }) {
  if (loading) return <div className="crm-state" role="status"><Loader2 size={22} className="loading-spinner" /><p>Loading {noun}...</p></div>;
  if (error) return <div className="crm-state" role="alert"><Building2 size={27} /><h2>Could not load {noun}</h2><p>Your records could not be retrieved. Try connecting again.</p><button className="crm-button" onClick={onRetry}>Try again<ArrowRight size={14} /></button></div>;
  if (empty) return <div className="crm-state"><span className="crm-empty-mark"><Users size={23} /></span><h2>{filtered ? 'No matching ' : 'Make room for new '}{noun}</h2><p>{filtered ? 'Try another search or clear your filters to see more records.' : 'Keep your relationships, details, and next opportunities together.'}</p>{!filtered && <button className="crm-button crm-primary" onClick={onCreate}><Plus size={14} />Add your first {noun === 'companies' ? 'company' : noun.slice(0, -1)}</button>}</div>;
  return <>{children}</>;
}

export function CrmPagination({ page, pageSize, total, onPage, onSize, busy }: { page: number; pageSize: number; total: number; onPage: (page: number) => void; onSize: (size: number) => void; busy: boolean }) {
  const pages = Math.max(1, Math.ceil(total / pageSize));
  return <footer className="crm-pagination"><span>{total ? `${(page - 1) * pageSize + 1}\u2013${Math.min(page * pageSize, total)}` : '0'} of {total} records</span><label><span className="sr-only">Records per page</span><select aria-label="Records per page" value={pageSize} onChange={event => onSize(Number(event.target.value))} disabled={busy}>{[10, 25, 50, 100].map(size => <option key={size} value={size}>{size} per page</option>)}</select></label><div><button onClick={() => onPage(page - 1)} disabled={busy || page <= 1} aria-label="Previous page"><ChevronLeft size={16} /></button><span>Page {page} of {pages}</span><button onClick={() => onPage(page + 1)} disabled={busy || page >= pages} aria-label="Next page"><ChevronRight size={16} /></button></div></footer>;
}

export function CrmDialog({ isOpen, onClose, title, children, busy = false }: { isOpen: boolean; onClose: () => void; title: string; children: ReactNode; size?: string; busy?: boolean }) {
  const ref = useRef<HTMLDialogElement>(null);
  const titleId = useId();
  useEffect(() => { const dialog = ref.current; if (isOpen && !dialog?.open) dialog?.showModal(); else if (!isOpen && dialog?.open) dialog.close(); }, [isOpen]);
  return <dialog ref={ref} className="crm-dialog crm-area" aria-labelledby={titleId} onCancel={event => { event.preventDefault(); if (!busy) onClose(); }}><div className="crm-dialog-heading"><div><span className="crm-eyebrow">RELATIONSHIP DETAILS</span><h2 id={titleId}>{title}</h2></div><button aria-label="Close dialog" onClick={onClose} disabled={busy}><X size={19} /></button></div>{isOpen && <div className="crm-dialog-body">{children}</div>}</dialog>;
}

type CardActions<T> = { items: T[]; selected: Set<string>; onSelect: (ids: Set<string>) => void; onEdit: (item: T) => void; onArchive: (item: T) => void };
function selectRecord(selected: Set<string>, id: string) { const next = new Set(selected); if (next.has(id)) next.delete(id); else next.add(id); return next; }
function companyWebsite(value: string) { try { const url = new URL(value); return ['https:', 'http:'].includes(url.protocol) ? url.href : null; } catch { return null; } }
export function CrmCompanyCards({ items, selected, onSelect, onEdit, onArchive }: CardActions<Company>) {
  return <div className="crm-card-grid">{items.map(company => <article className="crm-card" key={company.id}><div className="crm-card-top"><span className="crm-card-mark"><Building2 size={19} /></span><div className="crm-card-tags"><Badge size="sm" variant={getStatusBadge(company.status)} dot>{company.status}</Badge><input type="checkbox" aria-label={`Select ${company.name}`} checked={selected.has(company.id)} onChange={() => onSelect(selectRecord(selected, company.id))} /></div></div><button className="crm-card-title" onClick={() => onEdit(company)}>{company.name}</button><p className="crm-card-subtitle">{company.domain || 'No domain recorded'}</p><div className="crm-card-details"><span><Briefcase />{company.industry || 'Industry not specified'}</span><span><Users />{company.size_band ? `${company.size_band} employees` : 'Company size not specified'}</span>{company.website && companyWebsite(company.website) && <a href={companyWebsite(company.website)!} target="_blank" rel="noopener noreferrer"><Globe />{company.domain || 'Visit website'}</a>}</div><div className="crm-card-footer"><span>{company.is_client ? 'Client account' : 'Prospective account'}</span><div><button aria-label={`Edit ${company.name}`} onClick={() => onEdit(company)}><Pencil size={14} /></button><button aria-label={`Archive ${company.name}`} onClick={() => onArchive(company)}><Trash2 size={14} /></button></div></div></article>)}</div>;
}
export function CrmContactCards({ items, companies, selected, onSelect, onEdit, onArchive }: CardActions<ContactRecord> & { companies: Company[] }) {
  return <div className="crm-card-grid">{items.map(contact => { const name = `${contact.first_name} ${contact.last_name}`.trim(); const company = companies.find(item => item.id === contact.company_id); return <article className="crm-card" key={contact.id}><div className="crm-card-top"><span className="crm-card-mark">{contact.first_name?.[0]}{contact.last_name?.[0]}</span><div className="crm-card-tags"><Badge size="sm" variant={getStatusBadge(contact.status)} dot>{contact.status}</Badge><input type="checkbox" aria-label={`Select ${name}`} checked={selected.has(contact.id)} onChange={() => onSelect(selectRecord(selected, contact.id))} /></div></div><button className="crm-card-title" onClick={() => onEdit(contact)}>{name || contact.email}</button><p className="crm-card-subtitle">{contact.job_title || 'Contact'}</p><div className="crm-card-details">{contact.email && <a href={`mailto:${contact.email}`}><Mail />{contact.email}</a>}{contact.phone && <a href={`tel:${contact.phone}`}><Phone />{contact.phone}</a>}{contact.company_id ? <a href={`/companies?record=${encodeURIComponent(contact.company_id)}`}><Building2 />{company?.name || 'View linked company'}</a> : <span><Building2 />Independent contact</span>}</div><div className="crm-card-footer"><span>Relationship details</span><div><button aria-label={`Edit ${name}`} onClick={() => onEdit(contact)}><Pencil size={14} /></button><button aria-label={`Archive ${name}`} onClick={() => onArchive(contact)}><Trash2 size={14} /></button></div></div></article>; })}</div>;
}

