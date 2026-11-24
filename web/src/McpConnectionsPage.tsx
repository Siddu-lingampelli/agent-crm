import { useEffect, useState, useCallback } from 'react';
import { Link } from 'react-router-dom';
import {
  ArrowRight,
  ArrowUpRight,
  Bot,
  CircleAlert,
  Copy,
  ExternalLink,
  History,
  Key,
  Loader2,
  Network,
  RefreshCw,
  ShieldCheck,
  X,
} from 'lucide-react';
import { api, type McpClient, type McpToken, type McpToolStat, type McpToolCall, ApiError } from './api';
import { Agent } from './AgentsPage';
import {
  AdminHeader,
  AdminPanel,
  AdminLoading,
  AdminError,
  AdminEmpty,
  AdminStatus,
} from './AdminArea';
import styles from './McpConnectionsPage.module.css';

export function McpConnectionsPage() {
  const [clients, setClients] = useState<McpClient[]>([]);
  const [tokens, setTokens] = useState<McpToken[]>([]);
  const [stats, setStats] = useState<McpToolStat[]>([]);
  const [calls, setCalls] = useState<McpToolCall[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [sinceHours, setSinceHours] = useState(24);
  const [genOpen, setGenOpen] = useState(false);
  const [genAgent, setGenAgent] = useState('');
  const [genName, setGenName] = useState('');
  const [genKey, setGenKey] = useState<string | null>(null);
  const [genLoading, setGenLoading] = useState(false);
  const [genError, setGenError] = useState<string | null>(null);

  const fetchAll = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [clientsRes, tokensRes, statsRes, callsRes, agentsRes] = await Promise.all([
        api.mcp.clients(),
        api.mcp.tokens(),
        api.mcp.stats(sinceHours),
        api.mcp.calls(),
        api.get<{ items: Agent[] }>('/agents'),
      ]);
      setClients(clientsRes.items ?? []);
      setTokens(tokensRes.items ?? []);
      setStats(statsRes.items ?? []);
      setCalls(callsRes.items ?? []);
      setAgents(agentsRes.items ?? []);
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        setError('MCP observability not enabled on this server.');
      } else {
        setError(err instanceof Error ? err.message : 'Failed to load MCP connections');
      }
    } finally {
      setLoading(false);
    }
  }, [sinceHours]);

  useEffect(() => { fetchAll(); }, [fetchAll]);

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text);
  };

  const formatDuration = (us: number) => {
    if (us < 1000) return `${us} µs`;
    if (us < 1_000_000) return `${(us / 1000).toFixed(1)} ms`;
    return `${(us / 1_000_000).toFixed(2)} s`;
  };

  const formatTime = (iso: string) => new Date(iso).toLocaleString('en-US', {
    hour: 'numeric',
    minute: '2-digit',
    second: '2-digit',
    month: 'short',
    day: 'numeric',
  });

  const isTokenAlive = (token: McpToken) => {
    if (token.revoked_at) return false;
    return new Date(token.expires_at) > new Date();
  };

  const statusBadge = (status: 'ok' | 'error') => (
    <AdminStatus status={status === 'ok' ? 'enabled' : 'disabled'} />
  );
  const activeTokenCount = tokens.filter(isTokenAlive).length;
  const totalCalls = stats.reduce((sum, item) => sum + item.calls, 0);
  const totalErrors = stats.reduce((sum, item) => sum + item.errors, 0);

  if (error && !loading) {
    return (
      <div className={`ad-page ${styles.page}`}>
        <AdminHeader
          title="MCP connections"
          description="OAuth clients, issued tokens, and per-tool activity for the MCP server."
        />
        <AdminError message={error} onRetry={fetchAll} />
      </div>
    );
  }

  return (
    <div className={`ad-page ${styles.page}`}>
      <AdminHeader
        title="MCP connections"
        description="OAuth clients, issued tokens, and per-tool activity for the MCP server."
        actions={
          <>
            <button
              className="ad-button ad-button-primary"
              onClick={() => { setGenOpen(true); setGenError(null); setGenKey(null); setGenAgent(agents.find(a => a.enabled)?.id || ''); setGenName('MCP agent key'); }}
              disabled={agents.length === 0}
            >
              <Key size={14} /> Generate API key
            </button>
            <Link className="ad-button" to="/agents?tab=agents"><Bot size={14} /> Agents</Link>
            <button
              className="ad-button"
              onClick={fetchAll}
              disabled={loading}
              aria-label="Refresh"
            >
              <RefreshCw size={14} className={loading ? 'loading-spinner' : ''} />
              Refresh
            </button>
          </>
        }
      />

      <section className={styles.connectionSummary} aria-label="MCP connection details">
        <div className={styles.endpointBlock}>
          <span>Workspace endpoint</span>
          <div><code>/api/v1/mcp</code><button className="ad-icon-button" onClick={() => copyToClipboard('/api/v1/mcp')} aria-label="Copy MCP endpoint" title="Copy endpoint"><Copy size={14} /></button></div>
        </div>
        <dl className={styles.connectionFacts}>
          <div><dt>Transport</dt><dd><i /> Stateless HTTP</dd></div>
          <div><dt>Authentication</dt><dd>Bearer token</dd><small>OAuth JWT or crm_ API key</small></div>
          <div><dt>Protocol</dt><dd>2024-11-05</dd></div>
        </dl>
        <div className={styles.connectionLinks}>
          <a href="/.well-known/oauth-protected-resource" target="_blank" rel="noopener noreferrer">Discovery document <ExternalLink size={12} /></a>
          <Link to="/agent-console">Agent console <ArrowUpRight size={12} /></Link>
        </div>
      </section>

      <section className={styles.metrics} aria-label="MCP connection summary" aria-busy={loading}>
        <div className={styles.metric}><span>OAuth clients</span><strong>{loading ? '—' : clients.length}</strong></div>
        <div className={styles.metric}><span>Active tokens</span><strong>{loading ? '—' : activeTokenCount}</strong></div>
        <div className={styles.metric}><span>Tool calls</span><strong>{loading ? '—' : totalCalls.toLocaleString()}</strong></div>
        <div className={`${styles.metric} ${totalErrors ? styles.metricWarning : ''}`}><span>Errors</span><strong>{loading ? '—' : totalErrors.toLocaleString()}</strong></div>
      </section>

      {/* OAuth Clients */}
      <AdminPanel
        title="OAuth clients"
        className={styles.dataPanel}
        description="Registered OAuth 2.1 clients and their redirect URIs."
        action={
          <Link className="ad-button" to="/oauth/register" target="_blank" rel="noopener">
            Register client <ExternalLink size={12} />
          </Link>
        }
      >
        {loading && !clients.length ? (
          <AdminLoading label="Loading OAuth clients…" />
        ) : clients.length === 0 ? (
          <AdminEmpty
            icon={<Network size={24} />}
            title="No OAuth clients registered yet."
            children={
              <>
                Register a client to issue tokens for AI agents.
                <a href="/oauth/register" target="_blank" rel="noopener" className="ad-text-link">
                  Register a client
                </a>
              </>
            }
          />
        ) : (
          <div className={styles.tableWrapper}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th scope="col">Client ID</th>
                  <th scope="col">Name</th>
                  <th scope="col">Type</th>
                  <th scope="col">Scopes</th>
                  <th scope="col">Redirect URIs</th>
                  <th scope="col">Created</th>
                </tr>
              </thead>
              <tbody>
                {clients.map((c) => (
                  <tr key={c.id}>
                    <td>
                      <code>{c.client_id}</code>
                      <button
                        className="ad-icon-button"
                        onClick={() => copyToClipboard(c.client_id)}
                        aria-label="Copy client ID"
                        title="Copy client ID"
                      >
                        <Copy size={12} />
                      </button>
                    </td>
                    <td>{c.name}</td>
                    <td>
                      <AdminStatus
                        status={c.token_endpoint_auth_method === 'none' ? 'enabled' : 'disabled'}
                      />
                    </td>
                    <td>
                      {c.scopes.map((s) => (
                        <span key={s} className={styles.scopeTag}>
                          {s}
                        </span>
                      ))}
                    </td>
                    <td>
                      <div className={styles.uris}>
                        {c.redirect_uris.map((u, i) => (
                          <code key={i}>{u}</code>
                        ))}
                      </div>
                    </td>
                    <td>{formatTime(c.created_at)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </AdminPanel>

      {/* Issued Access Tokens */}
      <AdminPanel
        title="Issued access tokens"
        className={styles.dataPanel}
        description="Issued credentials, scopes, and expiration status."
      >
        {loading && !tokens.length ? (
          <AdminLoading label="Loading tokens…" />
        ) : tokens.length === 0 ? (
          <AdminEmpty
            icon={<ShieldCheck size={24} />}
            title="No access tokens issued yet."
            children="Tokens appear here after an OAuth client completes the authorization code flow (with PKCE)."
          />
        ) : (
          <div className={styles.tableWrapper}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th scope="col">Token (prefix)</th>
                  <th scope="col">Client ID</th>
                  <th scope="col">User</th>
                  <th scope="col">Scopes</th>
                  <th scope="col">Status</th>
                  <th scope="col">Expires</th>
                  <th scope="col">Created</th>
                </tr>
              </thead>
              <tbody>
                {tokens.map((t) => {
                  const alive = isTokenAlive(t);
                  return (
                    <tr key={t.token_hash}>
                      <td>
                        <code>{t.token_hash}</code>
                      </td>
                      <td>
                        <code>{t.client_id}</code>
                      </td>
                      <td>
                        <code>{t.user_id.slice(0, 12)}…</code>
                      </td>
                      <td>
                        {t.scopes.map((s) => (
                          <span key={s} className={styles.scopeTag}>
                            {s}
                          </span>
                        ))}
                      </td>
                      <td>
                        {alive ? (
                          <AdminStatus status="enabled" />
                        ) : t.revoked_at ? (
                          <AdminStatus status="disabled" />
                        ) : (
                          <AdminStatus status="degraded" />
                        )}
                      </td>
                      <td>{formatTime(t.expires_at)}</td>
                      <td>{formatTime(t.created_at)}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </AdminPanel>

      {/* Per-tool stats */}
      <AdminPanel
        title={`Per-tool activity · last ${sinceHours}h`}
        className={styles.dataPanel}
        description="See which tools your agents are using and spot errors or slow calls."
      >
        <div className={styles.statsFilter}>
          <label htmlFor="since-hours" className={styles.filterLabel}>
            Time window
            <select
              id="since-hours"
              value={sinceHours}
              onChange={(e) => setSinceHours(Number(e.target.value))}
              className={styles.filterSelect}
            >
              <option value={1}>1 hour</option>
              <option value={6}>6 hours</option>
              <option value={24}>24 hours</option>
              <option value={168}>7 days</option>
            </select>
          </label>
          <button
            className="ad-button"
            onClick={fetchAll}
            disabled={loading}
            aria-label="Refresh stats"
          >
            <RefreshCw size={14} className={loading ? 'loading-spinner' : ''} />
          </button>
        </div>

        {loading && !stats.length ? (
          <AdminLoading label="Loading tool stats…" />
        ) : stats.length === 0 ? (
          <AdminEmpty
            icon={<History size={24} />}
            title="No tool calls recorded in this window."
            children="Activity appears here when agents call MCP tools."
          />
        ) : (
          <div className={styles.tableWrapper}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th scope="col">Tool</th>
                  <th scope="col">Calls</th>
                  <th scope="col">Errors</th>
                  <th scope="col">Error rate</th>
                  <th scope="col">Total time</th>
                  <th scope="col">Avg time</th>
                </tr>
              </thead>
              <tbody>
                {stats.map((s) => (
                  <tr key={s.tool}>
                    <td><code>{s.tool}</code></td>
                    <td>{s.calls.toLocaleString()}</td>
                    <td>{s.errors.toLocaleString()}</td>
                    <td>
                      {s.calls > 0
                        ? `${((s.errors / s.calls) * 100).toFixed(1)}%`
                        : '—'}
                    </td>
                    <td>{formatDuration(s.total_duration_us)}</td>
                    <td>{formatDuration(s.avg_duration_us)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </AdminPanel>

      {/* Recent tool calls */}
      <AdminPanel
        title="Recent tool calls"
        className={styles.dataPanel}
        description="Individual MCP tool invocations (newest first). Shows the calling actor, tool name, status, latency, and any error."
      >
        {loading && !calls.length ? (
          <AdminLoading label="Loading recent calls…" />
        ) : calls.length === 0 ? (
          <AdminEmpty
            icon={<History size={24} />}
            title="No tool calls recorded yet."
            children="Individual calls appear here as agents invoke MCP tools."
          />
        ) : (
          <div className={styles.tableWrapper}>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th scope="col">Time</th>
                  <th scope="col">Actor</th>
                  <th scope="col">Tool</th>
                  <th scope="col">Status</th>
                  <th scope="col">Duration</th>
                  <th scope="col">Error</th>
                  <th scope="col">IP</th>
                </tr>
              </thead>
              <tbody>
                {calls.map((c) => (
                  <tr key={c.id}>
                    <td>{formatTime(c.created_at)}</td>
                    <td>
                      <code>{c.actor_type}:{c.actor_id.slice(0, 12)}…</code>
                    </td>
                    <td><code>{c.tool}</code></td>
                    <td>{statusBadge(c.status)}</td>
                    <td>{formatDuration(c.duration_us)}</td>
                    <td>
                      {c.error && (
                        <span className={styles.errorText} title={c.error}>
                          {c.error.length > 60 ? c.error.slice(0, 60) + '…' : c.error}
                        </span>
                      )}
                    </td>
                    <td><code>{c.ip_address || '—'}</code></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </AdminPanel>

      <footer className="ad-footer">
        <span>
          <ShieldCheck size={12} /> MCP connections for your current workspace
        </span>
        <Link to="/agent-console">
          Open Agent Console <ArrowRight size={12} />
        </Link>
      </footer>

      {genOpen && (
        <div className="modal-overlay" onClick={() => { setGenOpen(false); setGenKey(null); setGenError(null); }}>
          <div className="modal-content modal-md mcp-key-modal" onClick={e => e.stopPropagation()}>
            <div className="modal-header">
              <div><span className="ad-eyebrow">MCP ACCESS</span><h3>Generate API key</h3><p className="mcp-key-subtitle">Create a credential for an enabled agent to connect to this workspace.</p></div>
              <button className="modal-close" onClick={() => { setGenOpen(false); setGenKey(null); setGenError(null); }}><X size={16} /></button>
            </div>
            {genKey ? (
              <div className="modal-body">
                <p className="text-sm text-neutral-600 dark:text-neutral-400 mb-4">Key created successfully. Copy it now — you won't see it again.</p>
                <div className="flex gap-2 mb-4 mt-4">
                  <code className="mcp-key-value">{genKey}</code>
                  <button onClick={() => navigator.clipboard.writeText(genKey)} className="ad-button ad-button-primary"><Copy size={14} />Copy</button>
                </div>
                <button className="ad-button ad-button-primary mcp-key-done" onClick={() => { setGenOpen(false); setGenKey(null); setGenError(null); }}>Done</button>
              </div>
            ) : (
              <div className="modal-body">
                <div className="mcp-key-form">
                  <div>
                    <label className="block text-sm font-medium mb-1">Agent</label>
                    <select value={genAgent} onChange={e => setGenAgent(e.target.value)} className="w-full px-3 py-2 border border-neutral-300 rounded bg-white text-sm">
                      {agents.filter(a => a.enabled).map(a => <option key={a.id} value={a.id}>{a.name} ({a.provider})</option>)}
                    </select>
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">Key name</label>
                    <input value={genName} onChange={e => setGenName(e.target.value)} className="w-full px-3 py-2 border border-neutral-300 rounded bg-white text-sm" placeholder="MCP agent key" />
                  </div>
                  {genError && <p className="text-sm text-red-600 dark:text-red-400">{genError}</p>}
                </div>
              </div>
            )}
            {!genKey && (
              <div className="modal-footer">
                <button onClick={() => { setGenOpen(false); setGenError(null); }} className="ad-button">Cancel</button>
                <button onClick={async () => {
                  setGenLoading(true);
                  setGenError(null);
                  try {
                    const res = await api.apiKeys.createMCPKey(genAgent, genName);
                    setGenKey(res.key);
                  } catch (e: any) {
                    setGenError(e.message || 'Failed to generate key');
                  } finally {
                    setGenLoading(false);
                  }
                }} disabled={genLoading || !genAgent} className="ad-button ad-button-primary">
                  {genLoading ? <Loader2 size={14} className="loading-spinner" /> : 'Generate'}
                </button>
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
