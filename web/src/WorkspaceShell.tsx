import { useEffect, useRef, useState, type ReactNode } from 'react';
import { Link, NavLink, useLocation } from 'react-router-dom';
import { ScrollText, ArrowDownLeft, ArrowUpRight, Building2, CheckSquare, ChevronDown, ChevronRight, CircleAlert, Contact, DollarSign, Home, Loader2, LogOut, Menu, Plus, Search, Settings, Users, X } from 'lucide-react';
import { api, type SearchResult } from './api';
import { recordUrl } from './dashboard-data';
import './Workspace.css';

type NavigationItem = { name: string; href: string; icon: ReactNode };
const createItems = [
  { name: 'Deal', href: '/deals?create=1', icon: DollarSign, note: 'Track a sales opportunity' },
  { name: 'Lead', href: '/leads?create=1', icon: Users, note: 'Start a new relationship' },
  { name: 'Contact', href: '/contacts?create=1', icon: Contact, note: 'Keep someone in your CRM' },
  { name: 'Company', href: '/companies?create=1', icon: Building2, note: 'Add a company or account' },
  { name: 'Task', href: '/tasks?create=1', icon: CheckSquare, note: 'Plan your next follow-up' },
];

export function WorkspaceShell({ children, navigation, adminNavigation }: { children: ReactNode; navigation: NavigationItem[]; adminNavigation: NavigationItem[] }) {
  const location = useLocation();
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [searching, setSearching] = useState(false);
  const [searchError, setSearchError] = useState('');
  const [user, setUser] = useState<{ name?: string; email?: string } | null>(null);
  const [signingOut, setSigningOut] = useState(false);
  const [error, setError] = useState('');
  const dialog = useRef<HTMLDialogElement>(null);
  const menuButton = useRef<HTMLButtonElement>(null);
  const closeButton = useRef<HTMLButtonElement>(null);
  const sidebar = useRef<HTMLElement>(null);
  const previouslyOpen = useRef(false);
  const pages = [...navigation, ...adminNavigation];
  const current = pages.find(page => page.href === location.pathname)?.name ?? 'Workspace';
  const accountName = user?.name || user?.email?.split('@')[0] || 'Your account';
  const initials = accountName.slice(0, 2).toUpperCase();
  const groups = [
    { label: '', items: navigation.filter(item => item.href === '/dashboard') },
    { label: 'CRM', items: navigation.filter(item => ['/leads', '/deals', '/contacts', '/companies'].includes(item.href)) },
    { label: 'WORK', items: navigation.filter(item => ['/tasks', '/projects', '/inbox', '/calendar'].includes(item.href)) },
    { label: 'AUTOMATION', items: adminNavigation.filter(item => ['/workflows', '/integrations', '/agent-console'].includes(item.href)) },
  ];
  const tools = adminNavigation.filter(item => !['/settings', '/workflows', '/integrations', '/agent-console'].includes(item.href));
  const matchingPages = pages.filter(page => `${page.name}${page.href === '/audit' ? ' activity' : ''}`.toLowerCase().includes(query.toLowerCase().trim()));

  useEffect(() => {
    let active = true;
    api.get<{ name?: string; email?: string }>('/auth/me').then(data => { if (active) setUser(data); }).catch(() => {});
    return () => { active = false; };
  }, []);
  useEffect(() => {
    const title = document.title; document.title = `${current} — TORQ`;
    return () => { document.title = title; };
  }, [current]);
  useEffect(() => {
    if (query.trim().length < 2) return;
    let active = true;
    const timeout = window.setTimeout(() => {
      api.search(query.trim(), 8).then(data => { if (active) { setResults(data.results ?? []); setSearchError(''); } })
        .catch(() => { if (active) { setSearchError('Could not search records. Page navigation is still available.'); setResults([]); } })
        .finally(() => { if (active) setSearching(false); });
    }, 220);
    return () => { active = false; window.clearTimeout(timeout); };
  }, [query]);
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        setQuery(''); setResults([]); setSearchError(''); setSearching(false);
        if (!sidebarOpen && !document.querySelector('[role="dialog"][aria-modal="true"]')) dialog.current?.showModal();
      }
      if (event.key === 'Escape' && sidebarOpen) setSidebarOpen(false);
      if (event.key === 'Tab' && sidebarOpen && !dialog.current?.open) {
        const elements = Array.from(sidebar.current?.querySelectorAll<HTMLElement>('a[href], button:not([disabled]), summary') ?? []).filter(element => element.getClientRects().length > 0);
        const first = elements[0], last = elements[elements.length - 1];
        if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last?.focus(); }
        else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first?.focus(); }
      }
    };
    window.addEventListener('keydown', onKey); return () => window.removeEventListener('keydown', onKey);
  }, [sidebarOpen]);
  useEffect(() => {
    if (sidebarOpen) closeButton.current?.focus();
    else if (previouslyOpen.current) menuButton.current?.focus();
    previouslyOpen.current = sidebarOpen;
  }, [sidebarOpen]);
  useEffect(() => {
    const desktop = window.matchMedia('(min-width: 901px)');
    const onResize = () => { if (desktop.matches) setSidebarOpen(false); };
    desktop.addEventListener('change', onResize); return () => desktop.removeEventListener('change', onResize);
  }, []);
  const navigateAway = () => { setSidebarOpen(false); dialog.current?.close(); };
  const openSearch = () => { setQuery(''); setResults([]); setSearching(false); setSearchError(''); dialog.current?.showModal(); };
  const signOut = async () => {
    setSigningOut(true); setError('');
    try { await api.post('/auth/logout', {}); window.location.assign('/login'); }
    catch { setError('Could not sign out. Please try again.'); setSigningOut(false); }
  };
  const navLink = (item: NavigationItem) => <NavLink key={item.href} to={item.href} onClick={navigateAway} className={({ isActive }) => `ws-nav-link${isActive ? ' is-active' : ''}`}>{item.icon}<span>{item.name}</span></NavLink>;

  return <div className="workspace">
    <a className="ws-skip" href="#workspace-content">Skip to content</a>
    {sidebarOpen && <button className="ws-backdrop" aria-label="Close navigation" onClick={() => setSidebarOpen(false)} />}
    <aside ref={sidebar} id="workspace-navigation" className={`ws-sidebar${sidebarOpen ? ' is-open' : ''}`} aria-label="Workspace navigation">
      <div className="ws-brand-row"><Link to="/dashboard" className="ws-brand" aria-label="TORQ dashboard"><svg viewBox="0 0 32 32" fill="none" aria-hidden="true"><path d="M3 7h20v6H3zM9 19h20v6H9z" fill="currentColor" transform="rotate(-28 16 16)" /></svg>torq<span>.</span></Link><span className="ws-edition">CRM</span><button ref={closeButton} className="ws-icon-button ws-close" onClick={() => setSidebarOpen(false)} aria-label="Close navigation"><X size={19} /></button></div>
      <Link to="/settings" className="ws-workspace-label"><span className="ws-workspace-symbol">T</span><div>Your workspace<small>CRM workspace</small></div><ChevronDown size={13} /></Link>
      <details className="ws-create-menu"><summary><Plus size={16} /><span>Create new</span><ChevronDown size={13} /></summary><div className="ws-create-options">{createItems.map(item => <Link key={item.name} to={item.href} onClick={event => { const details = event.currentTarget.closest('details'); if (details) details.open = false; navigateAway(); }}><item.icon size={16} /><span>{item.name}<small>{item.note}</small></span><Plus size={12} /></Link>)}</div></details>
      <nav className="ws-nav" aria-label="Main navigation">{groups.map((group, index) => <div className="ws-nav-group" key={group.label || index}>{group.label && <span className="ws-nav-label">{group.label}</span>}{group.items.map(navLink)}</div>)}<details className="ws-tools" key={current} open={tools.some(item => item.href === location.pathname) || undefined}><summary><Settings size={14} />Administration<ChevronDown size={12} /></summary><div>{tools.map(item => navLink(item.href === '/admin' ? { ...item, name: 'Overview' } : item))}</div></details></nav>
      <div className="ws-sidebar-bottom"><NavLink to="/settings" className={({ isActive }) => `ws-nav-link ws-settings${isActive ? ' is-active' : ''}`} onClick={navigateAway}><Settings size={16} /><span>Settings</span></NavLink><div className="ws-user"><span className="ws-avatar">{initials}</span><div><strong>{accountName}</strong><small>{user?.email || 'TORQ account'}</small></div><button className="ws-icon-button" aria-label="Sign out" title="Sign out" disabled={signingOut} onClick={signOut}><LogOut size={15} /></button></div>{error && <p className="ws-signout-error" role="alert">{error}</p>}</div>
    </aside>
    <div className="ws-body" inert={sidebarOpen}>
      <header className="ws-header"><nav className="ws-header-location" aria-label="Breadcrumb"><button ref={menuButton} className="ws-icon-button ws-menu" aria-label="Open navigation" aria-expanded={sidebarOpen} aria-controls="workspace-navigation" onClick={() => setSidebarOpen(true)}><Menu size={21} /></button><Link to="/dashboard" aria-label="Workspace home"><Home size={14} /></Link><ChevronRight size={12} /><strong aria-current="page">{current}</strong></nav><div className="ws-header-actions"><button className="ws-search" aria-label="Search CRM records and pages" onClick={openSearch}><Search size={14} /><span>Search anything...</span><kbd>Ctrl K</kbd></button><span className="ws-header-divider" /><Link className="ws-icon-button" to="/audit" aria-label="Audit log" title="Audit log"><ScrollText size={17} /></Link><Link className="ws-avatar" to="/settings" aria-label="Your profile">{initials}</Link></div></header>
      <main id="workspace-content" tabIndex={-1} className="ws-content">{children}</main>
    </div>
    <dialog ref={dialog} className="ws-command" aria-labelledby="ws-command-title" onClick={event => { if (event.target === event.currentTarget) dialog.current?.close(); }}>
      <div className="ws-command-heading"><Search size={18} /><h2 id="ws-command-title">Search your CRM</h2><button className="ws-icon-button" aria-label="Close search" onClick={() => dialog.current?.close()}><X size={18} /></button></div>
      <input autoFocus value={query} onChange={event => { const value = event.target.value; setQuery(value); setResults([]); setSearchError(''); setSearching(value.trim().length >= 2); }} placeholder="Search contacts, companies, deals, and pages..." aria-label="Search CRM" />
      <div className="ws-command-results"><span className="ws-result-label">PAGES</span>{matchingPages.slice(0, query ? 20 : 6).map(page => <Link key={page.href} to={page.href} onClick={navigateAway}>{page.icon}<span>{page.name}</span><ArrowDownLeft size={12} /></Link>)}
        {query.trim().length >= 2 && <><span className="ws-result-label">RECORDS</span>{searching ? <p className="ws-search-status" role="status"><Loader2 size={14} className="loading-spinner" />Searching your workspace...</p> : searchError ? <p className="ws-search-status" role="alert"><CircleAlert size={14} />{searchError}</p> : results.length === 0 ? <p className="ws-search-status">No matching records.</p> : results.map(result => <Link key={`${result.entity_type}-${result.entity_id}`} to={recordUrl(result.entity_type, result.entity_id) || `/${result.entity_type}`} onClick={navigateAway}><Search size={14} /><span>{result.title}<small>{result.subtitle || result.entity_type}</small></span><ArrowUpRight size={13} /></Link>)}</>}
        {query && matchingPages.length === 0 && query.trim().length < 2 && <p className="ws-search-status">Type at least two characters to search records.</p>}
      </div><div className="ws-command-footer"><span>Tab to navigate · Enter to open</span><span>Esc to close</span></div>
    </dialog>
  </div>;
}
