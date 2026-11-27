import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { ArrowLeft, Eye, EyeOff, Inbox, Link2, MailPlus, Plug, RefreshCw, Send, Server, Trash2, MousePointerClick } from 'lucide-react';
import { api, type InboundMessage, type MailConnection, type MailConnectionCreateInput } from './api';
import { useAdminData } from './admin-data';
import { FormField, Input } from './components';
import { AutomationButton as Button, AutomationDialog as Dialog, AutomationMessage as Message, AutomationLoading as Loading, AutomationSearch as Search, AutomationPagination as Pagination, RefreshButton } from './AutomationUI';
import { errorMessage, useDebouncedSearch, automationDate } from './automation-data';
import { CommunicationsHeader, CommunicationsEmpty } from './CommunicationsUI';
import { sandboxEmailHTML } from './communications-data';

type EditorState = { connection?: MailConnection } | null;
type OAuthProvider = { id: string; name: string; configured: boolean };

// One-click connect (HubSpot/Salesforce style): clicking a provider immediately POSTs
// /mail/oauth/connect and redirects the whole tab to the provider's consent URL.
// The provider redirects back to /api/v1/mail/oauth/callback, which lands on /inbox?mail_connected=...
function useOAuthConnect() {
  const [providers, setProviders] = useState<OAuthProvider[] | null>(null);
  const [busyProvider, setBusyProvider] = useState('');
  const [error, setError] = useState('');
  useEffect(() => {
    api.mail.oauthProviders().then(d => setProviders(d.providers)).catch(err => setError(errorMessage(err)));
  }, []);
  async function connect(provider: string) {
    if (busyProvider) return;
    setBusyProvider(provider); setError('');
    try {
      const res = await api.mail.oauthConnect(provider);
      window.location.href = res.authorization_url;
    } catch (err) { setError(errorMessage(err)); setBusyProvider(''); }
  }
  return { providers, busyProvider, error, connect, clearError: () => setError('') };
}

function ConnectionEditor({ editor, onClose, onSaved }: { editor: NonNullable<EditorState>; onClose: () => void; onSaved: () => void }) {
  const [fields, setFields] = useState({
    name: editor.connection?.name ?? '',
    email_address: editor.connection?.email_address ?? '',
    imap_host: editor.connection?.imap_host ?? '',
    imap_port: String(editor.connection?.imap_port ?? 993),
    use_tls: editor.connection?.use_tls ?? true,
    username: editor.connection?.username ?? '',
    password: '',
    mailbox: editor.connection?.mailbox ?? 'INBOX',
    poll_interval_seconds: String(editor.connection?.poll_interval_seconds ?? 60),
  });
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);
  const update = (key: keyof typeof fields, value: string | boolean) => { setFields(p => ({ ...p, [key]: value })); setErrors(p => ({ ...p, [key]: '' })); };
  async function save(e: React.FormEvent) {
    e.preventDefault();
    if (saving) return;
    const validation: Record<string, string> = {};
    if (!fields.name.trim()) validation.name = 'Enter a name.';
    if (!fields.email_address.trim() || !fields.email_address.includes('@')) validation.email_address = 'Enter a valid email address.';
    if (!fields.imap_host.trim()) validation.imap_host = 'Enter the IMAP host.';
    if (!editor.connection && !fields.password) validation.password = 'Enter the IMAP password.';
    const port = Number(fields.imap_port);
    if (!Number.isInteger(port) || port < 1 || port > 65535) validation.imap_port = 'Enter a valid port.';
    const interval = Number(fields.poll_interval_seconds);
    if (!Number.isInteger(interval) || interval < 15) validation.poll_interval_seconds = 'Minimum 15 seconds.';
    setErrors(validation);
    if (Object.keys(validation).length) return;
    setSaving(true); setError('');
    try {
      const base = { name: fields.name.trim(), email_address: fields.email_address.trim(), imap_host: fields.imap_host.trim(), imap_port: port, use_tls: fields.use_tls, username: fields.username.trim(), mailbox: fields.mailbox.trim() || 'INBOX', poll_interval_seconds: interval };
      if (editor.connection) {
        const data: Parameters<typeof api.mail.updateConnection>[1] = { ...base };
        if (fields.password) data.password = fields.password;
        await api.mail.updateConnection(editor.connection.id, data);
      } else {
        const data: MailConnectionCreateInput = { ...base, password: fields.password };
        await api.mail.createConnection(data);
      }
      onSaved();
    } catch (err) { setError(errorMessage(err)); } finally { setSaving(false); }
  }
  return <Dialog title={editor.connection ? 'Edit inbox connection' : 'Connect an inbox'} description="IMAP credentials are encrypted at rest (AES-256-GCM) and only decrypted while polling." onClose={onClose} busy={saving} wide>
    <form onSubmit={save}><div className="au-dialog-body">{error && <Message message={error} error />}
      <div className="au-form-grid">
        <FormField label="Name" error={errors.name} required><Input value={fields.name} onChange={e => update('name', e.target.value)} placeholder="Work inbox" disabled={saving} /></FormField>
        <FormField label="Email address" error={errors.email_address} required><Input value={fields.email_address} onChange={e => update('email_address', e.target.value)} placeholder="me@company.com" disabled={saving} /></FormField>
        <FormField label="IMAP host" error={errors.imap_host} required><Input value={fields.imap_host} onChange={e => update('imap_host', e.target.value)} placeholder="imap.gmail.com" disabled={saving} /></FormField>
        <FormField label="Port" error={errors.imap_port} required><Input value={fields.imap_port} onChange={e => update('imap_port', e.target.value)} inputMode="numeric" disabled={saving} /></FormField>
        <FormField label="Username"><Input value={fields.username} onChange={e => update('username', e.target.value)} placeholder="Defaults to email address" disabled={saving} /></FormField>
        <FormField label="Mailbox"><Input value={fields.mailbox} onChange={e => update('mailbox', e.target.value)} placeholder="INBOX" disabled={saving} /></FormField>
        <FormField label="Poll every (seconds)" error={errors.poll_interval_seconds} required><Input value={fields.poll_interval_seconds} onChange={e => update('poll_interval_seconds', e.target.value)} inputMode="numeric" disabled={saving} /></FormField>
        <FormField label="Password" error={errors.password} required={!editor.connection} helpText={editor.connection ? 'Leave blank to keep the stored password.' : 'Encrypted before storage.'}><Input type="password" value={fields.password} onChange={e => update('password', e.target.value)} autoComplete="new-password" disabled={saving} /></FormField>
      </div>
      <label className="au-checkbox"><input type="checkbox" checked={fields.use_tls} onChange={e => update('use_tls', e.target.checked)} disabled={saving} /><span>Use TLS (recommended; forced for port 993)</span></label>
    </div><div className="au-dialog-footer"><Button disabled={saving} onClick={onClose}>Cancel</Button><Button primary type="submit" disabled={saving}>{saving ? 'Saving...' : editor.connection ? 'Save connection' : 'Connect inbox'}</Button></div></form></Dialog>;
}

function InboundReader({ message, onBack, onChanged }: { message: InboundMessage; onBack: () => void; onChanged: () => void }) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  async function patch(data: Parameters<typeof api.inbox.update>[1]) {
    if (busy) return;
    setBusy(true); setError('');
    try { await api.inbox.update(message.id, data); onChanged(); } catch (err) { setError(errorMessage(err)); } finally { setBusy(false); }
  }
  return <section className="cm-mail-reader" aria-label="Message reading pane">
    <div className="cm-reader-actions"><button className="cm-reader-back" onClick={onBack}><ArrowLeft size={16} />Inbox</button>
      <div>
        <Button onClick={() => patch({ is_read: !message.is_read })} disabled={busy}>{message.is_read ? <EyeOff size={14} /> : <Eye size={14} />}{message.is_read ? 'Mark unread' : 'Mark read'}</Button>
        <Button onClick={() => patch({ is_archived: !message.is_archived })} disabled={busy}>{message.is_archived ? 'Unarchive' : 'Archive'}</Button>
      </div>
    </div>
    {error && <Message message={error} error />}
    <div className="cm-message-heading"><h2>{message.subject || '(no subject)'}</h2>
      <div className="cm-message-address"><span className="cm-recipient-avatar" aria-hidden="true">{(message.from_name || message.from_addr).slice(0, 1).toUpperCase()}</span>
        <div><span><strong>From</strong> {message.from_name ? `${message.from_name} <${message.from_addr}>` : message.from_addr}</span><small>Received {automationDate(message.received_at)} · To {message.to_addr}</small></div>
      </div>
      {message.related_id && <dl className="cm-recipient-details"><div><dt>Linked</dt><dd>{message.related_type} / {message.related_id}</dd></div></dl>}
    </div>
    <div className="cm-message-body">{message.body_html ? <iframe title="Message HTML preview" sandbox="" referrerPolicy="no-referrer" srcDoc={sandboxEmailHTML(message.body_html)} /> : <p>{message.body_text || 'This message has no readable body.'}</p>}</div>
    <div className="cm-message-footer"><Inbox size={13} /><span>Inbound message via {message.connection_id.slice(0, 8)}…</span></div>
  </section>;
}

export function MailInboxPage() {
  const [folder, setFolder] = useState<'all' | 'unread' | 'archived'>('all');
  const [query, setQuery] = useState('');
  const search = useDebouncedSearch(query);
  const [page, setPage] = useState(1);
  const [size, setSize] = useState(12);
  const [selectedId, setSelectedId] = useState('');
  const [editor, setEditor] = useState<EditorState>(null);
  const oauth = useOAuthConnect();
  const [deleting, setDeleting] = useState<MailConnection | null>(null);
  const [showConnections, setShowConnections] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [searchParams, setSearchParams] = useSearchParams();

  const connections = useAdminData(() => api.mail.listConnections(), 'mail-connections');
  useEffect(() => {
    const connected = searchParams.get('mail_connected');
    const mailError = searchParams.get('mail_error');
    if (connected) { setNotice(`Connected ${connected}. Syncing inbox…`); connections.refresh(); messages.refresh(); setSearchParams({}, { replace: true }); }
    if (mailError) { setError(`Mail connect failed: ${mailError}`); setSearchParams({}, { replace: true }); }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);
  const isRead = folder === 'unread' ? false : folder === 'all' ? undefined : undefined;
  const isArchived = folder === 'archived' ? true : undefined;
  const messages = useAdminData(
    () => api.inbox.list({ q: search || undefined, is_read: isRead, is_archived: isArchived, limit: size, offset: (page - 1) * size }),
    `inbox:${search}:${folder}:${page}:${size}`
  );
  const items = messages.data?.items ?? [];
  const conns = connections.data?.items ?? [];
  const oauthProviders = oauth.providers ?? [];
  const selected = items.find(m => m.id === selectedId);

  async function syncNow() {
    if (busy) return;
    setBusy(true); setError('');
    try { await api.mail.sync(); setNotice('Inbox sync started — new messages appear below.'); messages.refresh(); connections.refresh(); }
    catch (err) { setError(errorMessage(err)); } finally { setBusy(false); }
  }
  async function testConn(id: string) {
    if (busy) return;
    setBusy(true); setError('');
    try { await api.mail.testConnection(id); setNotice('IMAP login and mailbox select succeeded.'); connections.refresh(); }
    catch (err) { setError(errorMessage(err)); connections.refresh(); } finally { setBusy(false); }
  }
  async function toggleConn(c: MailConnection) {
    if (busy) return;
    setBusy(true); setError('');
    try { await api.mail.updateConnection(c.id, { enabled: !c.enabled }); connections.refresh(); }
    catch (err) { setError(errorMessage(err)); } finally { setBusy(false); }
  }
  async function removeConn() {
    if (!deleting || busy) return;
    setBusy(true); setError('');
    try { await api.mail.deleteConnection(deleting.id); setDeleting(null); setNotice('Inbox connection removed.'); connections.refresh(); messages.refresh(); }
    catch (err) { setError(errorMessage(err)); } finally { setBusy(false); }
  }

  return <div className="automation-area communications-area mail-client mail-client-inbox">
    <CommunicationsHeader title="Inbox" description="Email conversations from your connected accounts.">
      <RefreshButton loading={messages.loading || connections.loading} onClick={() => { messages.refresh(); connections.refresh(); }} />
      <Button onClick={syncNow} disabled={busy}><RefreshCw size={15} className={busy ? 'au-spin' : ''} />Sync now</Button>
      {conns.length > 0 && <Button onClick={() => setShowConnections(value => !value)}><Server size={15} />{showConnections ? 'Hide mailboxes' : 'Mailboxes'}</Button>}
      {conns.length > 0 && oauthProviders.map(p => (
        <Button key={p.id} onClick={() => oauth.connect(p.id)} disabled={!p.configured || !!oauth.busyProvider}
          title={p.configured ? `Connect ${p.name} — you'll authorize on their site` : 'Not configured on this server (set CRM_MAIL_OAUTH_*_CLIENT_ID/SECRET)'}
          className={`cm-oauth-connect${oauth.busyProvider === p.id ? ' au-spin-wrap' : ''}`}>
          <span className={`cm-oauth-mark is-${p.id}`} aria-hidden="true">{p.id === 'google' ? 'G' : 'M'}</span>{p.name.replace(' (', ' · ').replace(')', '')}{oauth.busyProvider === p.id ? '…' : ''}
        </Button>
      ))}
      <Button primary onClick={() => setEditor({})}><MailPlus size={16} />IMAP manually</Button>
    </CommunicationsHeader>
    {notice && <Message message={notice} onClose={() => setNotice('')} />}
    {(error || oauth.error) && <Message message={error || oauth.error} error onClose={() => { setError(''); oauth.clearError(); }} />}

    {conns.length > 0 && showConnections && <section className="au-panel cm-connections-panel" aria-label="Connected inboxes">
      <header className="cm-connections-header"><div><h2>Connected inboxes</h2><p>Accounts that sync inbound mail into your workspace.</p></div><span>{conns.length} {conns.length === 1 ? 'mailbox' : 'mailboxes'}</span></header>
      <div className="cm-mail-connections">
        {conns.map(c => <div className="cm-connection-row" key={c.id}>
          <span className="cm-connection-icon"><Server size={18} /></span>
          <div className="cm-connection-info"><strong>{c.name}</strong><span>{c.email_address} · {c.imap_host}:{c.imap_port} · {c.mailbox}</span>
            <small>{c.enabled ? 'Enabled' : 'Disabled'} · poll {c.poll_interval_seconds}s · {c.last_sync_at ? `last sync ${automationDate(c.last_sync_at)}` : 'never synced'}{c.last_error ? ` · error: ${c.last_error}` : ''}</small></div>
          <div className="cm-connection-actions">
            <span className="cm-auth-badge">{c.auth_mode === 'oauth' ? `OAuth · ${c.oauth_provider ?? ''}` : 'Password'}</span>
            <Button onClick={() => testConn(c.id)} disabled={busy}><Plug size={14} />Test</Button>
            <Button onClick={() => toggleConn(c)} disabled={busy}>{c.enabled ? 'Disable' : 'Enable'}</Button>
            <Button onClick={() => setEditor({ connection: c })} disabled={busy}>Edit</Button>
            <button className="au-icon-button" aria-label={`Delete ${c.name}`} onClick={() => setDeleting(c)}><Trash2 size={15} /></button>
          </div>
        </div>)}
      </div>
    </section>}

    <section className={`au-panel cm-inbox-shell${selectedId ? ' is-reading' : ''}`}>
      <aside className="cm-mail-folders">
        <span className="au-eyebrow">INBOUND MAIL</span>
        <nav aria-label="Inbox folders">
          {[ { id: 'all', label: 'All', icon: Inbox }, { id: 'unread', label: 'Unread', icon: Eye }, { id: 'archived', label: 'Archived', icon: Link2 } ].map(f => (
            <button key={f.id} className={folder === f.id ? 'is-active' : ''} aria-pressed={folder === f.id} onClick={() => { setFolder(f.id as typeof folder); setPage(1); setSelectedId(''); }}>
              <f.icon size={16} /><span>{f.label}</span>{folder === f.id && <small>{messages.data?.total ?? '—'}</small>}
            </button>
          ))}
        </nav>
        <div className="cm-mail-rail-note"><Send size={17} /><p>Replies from known contacts or leads are linked automatically.</p></div>
      </aside>
      <div className="cm-mail-workspace">
        <div className="au-toolbar cm-inbox-toolbar">
          <Search value={query} onChange={v => { setQuery(v); setPage(1); setSelectedId(''); }} placeholder="Search inbox" />
          {query && <button className="au-text-button" onClick={() => { setQuery(''); setPage(1); }}>Clear</button>}
        </div>
        {messages.error && <Message message={messages.error} error onRetry={messages.refresh} />}
        {messages.loading && !messages.data ? <Loading /> : messages.error && !messages.data ? null : selected ? <InboundReader key={selected.id} message={selected} onBack={() => setSelectedId('')} onChanged={() => { messages.refresh(); }} /> : !items.length ? (
          <CommunicationsEmpty icon={<Inbox size={29} />} title={query ? 'No matching messages' : conns.length ? 'No messages yet' : 'Connect your inbox to receive mail'} description={query ? 'Try another search or clear your filters.' : conns.length ? 'Run "Sync now" to fetch recent messages from your connected mailboxes.' : oauthProviders.some(provider => provider.configured) ? 'Choose an email provider below, or connect a mailbox manually.' : 'OAuth sign-in is not configured on this server. Connect an inbox manually with IMAP, or ask an administrator to configure Google or Microsoft OAuth.'}>
            {!conns.length && oauthProviders.length > 0 && <div className="cm-oauth-providers">
              {oauthProviders.map(p => (
                <button key={p.id} className="cm-oauth-provider" disabled={!p.configured || !!oauth.busyProvider} onClick={() => oauth.connect(p.id)}>
                  <strong><span className={`cm-oauth-mark is-${p.id}`} aria-hidden="true">{p.id === 'google' ? 'G' : 'M'}</span>{p.name}</strong>
                  <span>{p.configured ? 'Click to connect — you will authorize on the provider site' : 'Not configured on this server'}</span>
                </button>
              ))}
            </div>}
          </CommunicationsEmpty>
        ) : <div className="cm-message-list" role="region" aria-label="Inbound messages">
            <div className="cm-message-list-heading"><span>{folder === 'all' ? 'All messages' : folder === 'unread' ? 'Unread' : 'Archived'}</span><small>{messages.data?.total} messages</small></div>
            {items.map(m => <button className={`cm-message-item${m.is_read ? '' : ' is-unread'}`} aria-pressed={false} key={m.id} onClick={() => setSelectedId(m.id)}>
              <span className="cm-list-avatar" aria-hidden="true">{(m.from_name || m.from_addr).slice(0, 1).toUpperCase()}</span>
              <span className="cm-message-item-to">{m.from_name || m.from_addr}</span>
              <span className="cm-message-summary"><strong>{m.subject || '(no subject)'}</strong><span className="cm-message-snippet">{m.snippet || 'No preview'}</span></span>
              <span className="cm-message-item-meta">{!m.is_read && <StatusDot />}{m.related_id && <span className="cm-linked"><MousePointerClick size={11} />{m.related_type}</span>}<time>{automationDate(m.received_at)}</time></span>
            </button>)}
          </div>}
        {!selected && !!messages.data?.total && <Pagination page={page} size={size} total={messages.data.total} onPage={v => { setPage(v); setSelectedId(''); }} onSize={v => { setSize(v); setPage(1); setSelectedId(''); }} />}
      </div>
    </section>

    {editor && <ConnectionEditor editor={editor} onClose={() => setEditor(null)} onSaved={() => { setEditor(null); setNotice('Inbox connection saved.'); connections.refresh(); messages.refresh(); }} />}
    {deleting && <Dialog title="Delete inbox connection?" description={deleting.name} onClose={() => setDeleting(null)} busy={busy}>
      <div className="au-dialog-body"><p className="au-dialog-copy">This removes the connection and all ingested messages from this workspace. Stored credentials are deleted.</p></div>
      <div className="au-dialog-footer"><Button disabled={busy} onClick={() => setDeleting(null)}>Cancel</Button><Button primary disabled={busy} onClick={removeConn}>{busy ? 'Deleting...' : 'Delete connection'}</Button></div>
    </Dialog>}
  </div>;
}

function StatusDot() { return <span className="cm-unread-dot" role="img" aria-label="Unread" title="Unread" />; }
