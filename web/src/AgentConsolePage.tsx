import { useCallback, useEffect, useRef, useState, type ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { ArrowRight, ArrowUpRight, Bot, Braces, Check, ChevronRight, CircleAlert, Clock3, Code2, Copy, FileJson2, History, Layers, Loader2, Play, Plug, RefreshCw, Search, ShieldCheck, SlidersHorizontal, Terminal, Trash2, Unplug, X } from 'lucide-react';
import { ConsoleTool, ConsoleRequest, RpcRequest, MCP_ENDPOINT, MCP_PROTOCOL, McpRequestError, exampleArguments, isObject, isReadTool, parseArguments, parseRawRequest, resultOutput, sendMcp } from './mcp-console';
import styles from './AgentConsole.module.css';

type ConsoleTab = 'runner' | 'history' | 'raw';
type ServerInfo = { name: string; version: string };
const initialRaw = JSON.stringify({ jsonrpc: '2.0', id: 1000, method: 'tools/list', params: {} }, null, 2);
const pretty = (value: unknown) => JSON.stringify(value, null, 2) ?? '';
const timestamp = (value: string) => new Date(value).toLocaleTimeString('en-US', { hour12: false });

function EmptyState({ icon, title, children, action }: { icon: ReactNode; title: string; children: ReactNode; action?: ReactNode }) {
  return <div className={styles.empty}><span className={styles.emptyIcon}>{icon}</span><h3>{title}</h3><p>{children}</p>{action}</div>;
}
function Status({ status }: { status: ConsoleRequest['status'] }) {
  return <span className={`${styles.status} ${styles[status]}`}>{status === 'pending' ? <Loader2 size={11} className={styles.spinning} /> : status === 'error' ? <CircleAlert size={11} /> : <Check size={11} />}{status === 'pending' ? 'Running' : status === 'error' ? 'Failed' : 'Success'}</span>;
}
export function ConsoleResult({ entry, onCopy }: { entry: ConsoleRequest | null; onCopy: (text: string) => void }) {
  const [view, setView] = useState<'output' | 'response'>('output');
  const output = entry ? view === 'response' ? pretty(entry.response) : resultOutput(entry.response) : '';
  return <section className={styles.resultPanel} aria-label="Execution result"><div className={styles.sectionHeader}><div><span className={styles.sectionIcon}><Terminal size={15} /></span><h2>Result</h2>{entry && <Status status={entry.status} />}</div><div className={styles.resultActions}><div className={styles.segmented} role="group" aria-label="Result format"><button aria-pressed={view === 'output'} className={view === 'output' ? styles.segmentActive : ''} onClick={() => setView('output')}>Output</button><button aria-pressed={view === 'response'} className={view === 'response' ? styles.segmentActive : ''} onClick={() => setView('response')}>JSON</button></div><button className={styles.iconButton} onClick={() => onCopy(output)} disabled={!output} aria-label="Copy result" title="Copy result"><Copy size={14} /></button></div></div>
    {!entry ? <EmptyState icon={<Terminal size={22} />} title="Your result will appear here.">Choose a tool, add its arguments, and run a request.</EmptyState> : entry.status === 'pending' ? <div className={styles.waiting} role="status"><Loader2 size={22} className={styles.spinning} /><strong>Waiting for the server</strong><span>{entry.request.method === 'tools/call' && isObject(entry.request.params) ? String(entry.request.params.name) : entry.request.method}</span></div> : <>
      <div className={styles.resultMeta}><span><Clock3 size={12} />{entry.latencyMs ?? 0} ms</span><span>Request #{entry.request.id ?? 'notification'}</span><span>{timestamp(entry.startedAt)}</span></div>
      {entry.error && <div className={styles.inlineError} role="alert"><CircleAlert size={14} /><span>{entry.error}</span></div>}
      {output && output !== 'null' ? <pre className={styles.resultCode} tabIndex={0} aria-label={view === 'output' ? 'Tool output' : 'JSON-RPC response'}>{output}</pre> : !entry.error && <div className={styles.accepted}><Check size={18} />Notification accepted. No response body was expected.</div>}
    </>}
  </section>;
}
export function ConsoleHistory({ entries, onInspect, onCopy, onClear }: { entries: ConsoleRequest[]; onInspect: (entry: ConsoleRequest) => void; onCopy: (text: string) => void; onClear: () => void }) {
  const [filter, setFilter] = useState<'all' | 'error'>('all');
  const visible = entries.filter(entry => filter === 'all' || entry.status === 'error');
  return <section className={styles.historyPanel}><div className={styles.sectionHeader}><div><History size={16} /><h2>Request history</h2><span className={styles.count}>{entries.length}</span></div><div className={styles.historyActions}><select value={filter} onChange={event => setFilter(event.target.value as 'all' | 'error')} aria-label="Filter request history"><option value="all">All requests</option><option value="error">Failed requests</option></select><button className={styles.secondaryButton} disabled={entries.length === 0 || entries.some(entry => entry.status === 'pending')} onClick={onClear}><Trash2 size={13} />Clear history</button></div></div>
    {visible.length === 0 ? <EmptyState icon={<History size={23} />} title={filter === 'error' ? 'No failed requests.' : 'A clear record of every request.'}>{filter === 'error' ? 'Failed requests will be listed here.' : 'Connect to the server to start a session. Requests and responses stay in this browser tab.'}</EmptyState> : <div className={styles.historyScroll}><table className={styles.historyTable}><caption className={styles.srOnly}>Console request history</caption><thead><tr><th scope="col">Time</th><th scope="col">Method / tool</th><th scope="col">Status</th><th scope="col">Duration</th><th scope="col">Request ID</th><th scope="col"><span className={styles.srOnly}>Actions</span></th></tr></thead><tbody>{visible.map(entry => <tr key={entry.id}><td>{timestamp(entry.startedAt)}</td><td><button className={styles.historyName} onClick={() => onInspect(entry)}>{entry.request.method === 'tools/call' && isObject(entry.request.params) ? String(entry.request.params.name) : entry.request.method}<small>{entry.request.method}</small></button></td><td><Status status={entry.status} /></td><td>{entry.latencyMs !== undefined ? `${entry.latencyMs} ms` : '—'}</td><td><code>{entry.request.id ?? 'Notification'}</code></td><td><button className={styles.iconButton} aria-label={`Copy request ${entry.request.id ?? entry.id}`} onClick={() => onCopy(pretty({ request: entry.request, response: entry.response, error: entry.error }))}><Copy size={13} /></button><button className={styles.iconButton} aria-label={`Inspect request ${entry.request.id ?? entry.id}`} onClick={() => onInspect(entry)}><ArrowUpRight size={14} /></button></td></tr>)}</tbody></table></div>}
    <div className={styles.panelFooter}><span>Latest 100 requests in this tab</span><span>Clearing history does not change CRM records</span></div>
  </section>;
}

export function ToolRunner({ tools, selectedTool, args, connected, busy, running = false, catalogLoading, onSelect, onArgs, onRun, onFormat, onRefresh, error }: {
  tools: ConsoleTool[]; selectedTool: ConsoleTool | null; args: string; connected: boolean; busy: boolean; running?: boolean; catalogLoading: boolean;
  onSelect: (tool: ConsoleTool) => void; onArgs: (value: string) => void; onRun: () => void; onFormat: () => void; onRefresh: () => void; error: string;
}) {
  const [query, setQuery] = useState('');
  const [category, setCategory] = useState('all');
  const [access, setAccess] = useState<'all' | 'read' | 'write'>('all');
  const [editorView, setEditorView] = useState<'arguments' | 'schema'>('arguments');
  const categories = [...new Set(tools.map(tool => tool.name.split('.')[0]))].sort();
  const visible = tools.filter(tool => (category === 'all' || tool.name.split('.')[0] === category) && (access === 'all' || isReadTool(tool) === (access === 'read')) && `${tool.name} ${tool.description || ''}`.toLowerCase().includes(query.toLowerCase().trim()));
  const properties = Object.entries(selectedTool?.inputSchema?.properties ?? {});
  const required = selectedTool?.inputSchema?.required ?? [];
  return <div className={styles.runnerGrid}>
    <aside className={styles.catalog} aria-label="Available MCP tools"><div className={styles.catalogHeading}><span><Layers size={15} /><strong>Tool library</strong><i>{tools.length}</i></span><button className={styles.iconButton} onClick={onRefresh} disabled={!connected || busy} aria-label="Refresh tool library" title="Refresh tools"><RefreshCw size={14} className={catalogLoading ? styles.spinning : ''} /></button></div>
      <div className={styles.catalogFilters}><label className={styles.toolSearch}><Search size={14} /><input type="search" placeholder="Find a tool..." aria-label="Search tools" value={query} onChange={event => setQuery(event.target.value)} /></label><div className={styles.filterRow}><select value={category} onChange={event => setCategory(event.target.value)} aria-label="Tool category"><option value="all">All categories</option>{categories.map(name => <option key={name} value={name}>{name.replace(/_/g, ' ')}</option>)}</select><select value={access} onChange={event => setAccess(event.target.value as 'all' | 'read' | 'write')} aria-label="Tool access type"><option value="all">All tools</option><option value="read">Read tools</option><option value="write">Write actions</option></select></div></div>
      <div className={styles.toolList}>{catalogLoading && tools.length === 0 ? <div className={styles.catalogEmpty} role="status"><Loader2 className={styles.spinning} size={20} /><p>Loading available tools...</p></div> : !connected ? <div className={styles.catalogEmpty}><Unplug size={24} /><strong>Connect to discover tools</strong><p>Your workspace tools will appear here after connecting.</p></div> : visible.length === 0 ? <div className={styles.catalogEmpty}><Search size={22} /><strong>No matching tools</strong><p>Try a different name or filter.</p><button onClick={() => { setQuery(''); setCategory('all'); setAccess('all'); }}>Reset filters</button></div> : visible.map(tool => <button key={tool.name} className={`${styles.toolItem} ${selectedTool?.name === tool.name ? styles.selectedTool : ''}`} onClick={() => { onSelect(tool); setEditorView('arguments'); }} disabled={busy} aria-pressed={selectedTool?.name === tool.name}><span className={styles.toolIcon}><Code2 size={14} /></span><span><code>{tool.name}</code><small>{tool.description || 'MCP tool'}</small></span><ChevronRight size={13} /></button>)}</div><div className={styles.catalogFooter}>{connected ? `${visible.length} of ${tools.length} tools` : 'Workspace-scoped tool catalog'}<ShieldCheck size={12} /></div>
    </aside>
    <section className={styles.editorPanel} aria-label="Tool runner">{!selectedTool ? <EmptyState icon={<Bot size={27} />} title={connected ? 'Choose a tool to get started.' : 'Your workspace, ready for agents.'}>{connected ? 'Browse the library to inspect a tool, prepare arguments, and run it against your CRM.' : 'Connect to the MCP server to explore and test the tools available to your account.'}</EmptyState> : <>
      <div className={styles.toolDetail}><div className={styles.toolTitle}><span className={styles.toolDetailIcon}><Code2 size={19} /></span><div><span className={styles.eyebrow}>SELECTED TOOL</span><h2>{selectedTool.name}</h2></div><span className={`${styles.accessBadge} ${isReadTool(selectedTool) ? styles.readBadge : styles.writeBadge}`}>{isReadTool(selectedTool) ? <ShieldCheck size={11} /> : <SlidersHorizontal size={11} />}{isReadTool(selectedTool) ? 'Read tool' : 'Write action'}</span></div><p>{selectedTool.description || 'Inspect the schema below for the expected arguments.'}</p><div className={styles.toolDetailMeta}><span>{properties.length} {properties.length === 1 ? 'parameter' : 'parameters'}</span><i /><span>{required.length} required</span></div></div>
      <div className={styles.editorToolbar}><div className={styles.editorTabs} role="group" aria-label="Tool editor view"><button className={editorView === 'arguments' ? styles.editorTabActive : ''} aria-pressed={editorView === 'arguments'} onClick={() => setEditorView('arguments')}><Braces size={13} />Arguments</button><button className={editorView === 'schema' ? styles.editorTabActive : ''} aria-pressed={editorView === 'schema'} onClick={() => setEditorView('schema')}><FileJson2 size={13} />Input schema</button></div>{editorView === 'arguments' && <button className={styles.formatButton} onClick={onFormat} disabled={busy}><Braces size={12} />Format JSON</button>}</div>
      {editorView === 'arguments' ? <><div className={styles.codeEditor}><div className={styles.codeHeader}><span>arguments.json</span><span>JSON</span></div><textarea id="console-tool-arguments" aria-label={`JSON arguments for ${selectedTool.name}`} aria-invalid={!!error} aria-describedby={error ? 'console-arguments-error' : 'console-arguments-help'} value={args} onChange={event => onArgs(event.target.value)} spellCheck={false} disabled={busy} onKeyDown={event => { if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') { event.preventDefault(); if (connected && !busy) onRun(); } }} /></div>{error && <div id="console-arguments-error" className={styles.inlineError} role="alert"><CircleAlert size={14} />{error}</div>}<p id="console-arguments-help" className={styles.editorHelp}>Arguments are validated against the tool's required fields before sending.</p></> : <div className={styles.schemaView}>{properties.length > 0 ? <div className={styles.parameterList}>{properties.map(([name, property]) => <div className={styles.parameter} key={name}><div><code>{name}</code><span>{Array.isArray(property.type) ? property.type.join(' | ') : String(property.type || 'any')}</span>{required.includes(name) && <b>Required</b>}</div>{typeof property.description === 'string' && <p>{property.description}</p>}{Array.isArray(property.enum) && <small>Allowed: {property.enum.join(', ')}</small>}</div>)}</div> : <p className={styles.noParameters}>This tool does not declare any parameters.</p>}<details><summary><ChevronRight size={12} />Full JSON schema</summary><pre tabIndex={0}>{pretty(selectedTool.inputSchema ?? {})}</pre></details></div>}
      <div className={styles.runBar}><span><ShieldCheck size={12} />{isReadTool(selectedTool) ? 'Reads workspace data' : 'Can change workspace data'}</span><div><kbd>Ctrl ↵</kbd><button className={styles.primaryButton} onClick={onRun} disabled={!connected || busy}>{running ? <Loader2 size={14} className={styles.spinning} /> : <Play size={13} />}{running ? 'Running...' : 'Run tool'}</button></div></div>
    </>}</section>
  </div>;
}

export function AgentConsolePage() {
  const [connected, setConnected] = useState(false);
  const [operation, setOperation] = useState<'connect' | 'catalog' | 'tool' | 'raw' | null>(null);
  const [tools, setTools] = useState<ConsoleTool[]>([]);
  const [selectedTool, setSelectedTool] = useState<ConsoleTool | null>(null);
  const [args, setArgs] = useState('{}');
  const [history, setHistory] = useState<ConsoleRequest[]>([]);
  const [selectedRequest, setSelectedRequest] = useState<number | null>(null);
  const [server, setServer] = useState<ServerInfo | null>(null);
  const [protocol, setProtocol] = useState(MCP_PROTOCOL);
  const [tab, setTab] = useState<ConsoleTab>('runner');
  const [raw, setRaw] = useState(initialRaw);
  const [error, setError] = useState('');
  const [argumentError, setArgumentError] = useState('');
  const [notice, setNotice] = useState('');
  const [signInRequired, setSignInRequired] = useState(false);
  const requestId = useRef(1);
  const logId = useRef(1);
  const protocolRef = useRef(MCP_PROTOCOL);
  const mounted = useRef(true);
  const requests = useRef(new Set<AbortController>());
  const resultRegion = useRef<HTMLDivElement>(null);
  const busy = operation !== null;
  const activeEntry = history.find(entry => entry.id === selectedRequest) ?? null;
  const lastResponse = history.find(entry => entry.latencyMs !== undefined);
  const toolCalls = history.filter(entry => entry.request.method === 'tools/call').length;

  useEffect(() => {
    mounted.current = true;
    const controllers = requests.current;
    return () => { mounted.current = false; controllers.forEach(controller => controller.abort()); };
  }, []);
  useEffect(() => { if (!notice) return; const timeout = window.setTimeout(() => setNotice(''), 4000); return () => window.clearTimeout(timeout); }, [notice]);

  const request = useCallback(async (payload: RpcRequest) => {
    const id = logId.current++;
    const started = performance.now();
    const controller = new AbortController();
    requests.current.add(controller);
    setHistory(previous => [{ id, request: payload, response: null, startedAt: new Date().toISOString(), status: 'pending' as const }, ...previous].slice(0, 100));
    setSelectedRequest(id);
    try {
      const response = await sendMcp(payload, { protocol: protocolRef.current, signal: AbortSignal.any([controller.signal, AbortSignal.timeout(30000)]) });
      if (mounted.current) setHistory(previous => previous.map(entry => entry.id === id ? { ...entry, response, status: 'success', latencyMs: Math.round(performance.now() - started) } : entry));
      return response;
    } catch (caught) {
      const message = caught instanceof Error ? caught.message : 'The request failed.';
      if (mounted.current) {
        setHistory(previous => previous.map(entry => entry.id === id ? { ...entry, response: caught instanceof McpRequestError ? caught.response : null, status: 'error', error: message, latencyMs: Math.round(performance.now() - started) } : entry));
        if (caught instanceof McpRequestError && caught.status === 401) { setConnected(false); setSignInRequired(true); }
      }
      throw caught;
    } finally { requests.current.delete(controller); }
  }, []);
  const catalog = async () => {
    const found: ConsoleTool[] = [];
    const cursors = new Set<string>();
    let cursor: string | undefined;
    do {
      const response = await request({ jsonrpc: '2.0', id: requestId.current++, method: 'tools/list', params: cursor ? { cursor } : {} });
      if (!Array.isArray(response?.result?.tools)) throw new Error('The server did not return a tool catalog.');
      for (const candidate of response.result.tools) if (isObject(candidate) && typeof candidate.name === 'string') found.push(candidate as unknown as ConsoleTool);
      cursor = typeof response.result.nextCursor === 'string' && response.result.nextCursor ? response.result.nextCursor : undefined;
      if (cursor && cursors.has(cursor)) throw new Error('The tool catalog returned a repeated page. Try refreshing the library.');
      if (cursor) cursors.add(cursor);
      if (cursors.size > 50) throw new Error('The tool catalog is too large to load in the console.');
    } while (cursor);
    const unique = [...new Map(found.map(tool => [tool.name, tool])).values()].sort((a, b) => a.name.localeCompare(b.name));
    if (mounted.current) {
      setTools(unique);
      const selection = unique.find(tool => tool.name === selectedTool?.name) || unique.find(tool => tool.name === 'companies.list') || unique.find(isReadTool) || unique[0] || null;
      setSelectedTool(selection);
      if (selection && selection.name !== selectedTool?.name) setArgs(exampleArguments(selection));
    }
    return unique.length;
  };
  const connect = async () => {
    if (busy) return;
    setOperation('connect'); setError(''); setArgumentError('');
    protocolRef.current = MCP_PROTOCOL;
    try {
      const response = await request({ jsonrpc: '2.0', id: requestId.current++, method: 'initialize', params: { protocolVersion: MCP_PROTOCOL, capabilities: {}, clientInfo: { name: 'torq-console', version: '1.0.0' } } });
      if (!response?.result || typeof response.result.protocolVersion !== 'string') throw new Error('The server did not complete initialization.');
      protocolRef.current = response.result.protocolVersion;
      await request({ jsonrpc: '2.0', method: 'notifications/initialized' });
      const count = await catalog();
      if (mounted.current) {
        const info = isObject(response.result.serverInfo) ? response.result.serverInfo : {};
        setServer({ name: typeof info.name === 'string' ? info.name : 'MCP server', version: typeof info.version === 'string' ? info.version : '' });
        setProtocol(protocolRef.current); setConnected(true); setSignInRequired(false); setSelectedRequest(null); setNotice(`Connected. ${count} tools available.`);
      }
    } catch (caught) { if (mounted.current) { setConnected(false); setError(caught instanceof Error ? caught.message : 'Could not connect.'); } }
    finally { if (mounted.current) setOperation(null); }
  };
  const refreshCatalog = async () => {
    if (busy || !connected) return;
    setOperation('catalog'); setError('');
    try { const count = await catalog(); if (mounted.current) { setSelectedRequest(null); setNotice(`Tool library refreshed. ${count} tools available.`); } }
    catch (caught) { if (mounted.current) setError(caught instanceof Error ? caught.message : 'Could not refresh tools.'); }
    finally { if (mounted.current) setOperation(null); }
  };
  const runTool = async () => {
    if (!selectedTool || busy || !connected) return;
    let parameters;
    try { parameters = parseArguments(args, selectedTool); } catch (caught) { setArgumentError(caught instanceof Error ? caught.message : 'Invalid arguments.'); return; }
    setOperation('tool'); setError(''); setArgumentError('');
    try { await request({ jsonrpc: '2.0', id: requestId.current++, method: 'tools/call', params: { name: selectedTool.name, arguments: parameters } }); if (mounted.current) setNotice(`${selectedTool.name} completed.`); }
    catch (caught) { if (mounted.current) setError(caught instanceof Error ? caught.message : 'The tool failed.'); }
    finally { if (mounted.current) setOperation(null); }
  };
  const runRaw = async () => {
    if (busy || !connected) return;
    let payload;
    try { payload = parseRawRequest(raw); } catch (caught) { setArgumentError(caught instanceof Error ? caught.message : 'Invalid request.'); return; }
    setOperation('raw'); setArgumentError(''); setError('');
    try { await request(payload); }
    catch (caught) { if (mounted.current) setError(caught instanceof Error ? caught.message : 'The request failed.'); }
    finally { if (mounted.current) setOperation(null); }
  };
  const copy = async (value: string) => {
    try { await navigator.clipboard.writeText(value); setNotice('Copied to clipboard.'); }
    catch { setError('Clipboard access is unavailable. Select the text and copy it manually.'); }
  };
  const format = () => {
    try { if (tab === 'raw') setRaw(pretty(parseRawRequest(raw))); else setArgs(pretty(parseArguments(args))); setArgumentError(''); }
    catch (caught) { setArgumentError(caught instanceof Error ? caught.message : 'Invalid JSON.'); }
  };

  return <div className={styles.page}>
    <header className={styles.pageHeader}><div><span className={styles.eyebrow}>AUTOMATION / DEVELOPER TOOLS</span><h1>Agent Console</h1><p>Explore your CRM tools. Test requests. See exactly what happens.</p></div><div className={styles.headerActions}><Link to="/agents" className={styles.secondaryButton}><SlidersHorizontal size={14} />Manage agents<ArrowUpRight size={12} /></Link><button className={connected ? styles.secondaryButton : styles.primaryButton} disabled={busy} onClick={connected ? () => { setConnected(false); setTools([]); setSelectedTool(null); setServer(null); setError(''); setNotice('Disconnected from the console.'); } : connect}>{operation === 'connect' ? <Loader2 size={14} className={styles.spinning} /> : connected ? <Unplug size={14} /> : <Plug size={14} />}{operation === 'connect' ? 'Connecting...' : connected ? 'Disconnect' : 'Connect server'}</button></div></header>
    <div className={styles.connectionBar}><div><span className={`${styles.connectionState} ${connected ? styles.isConnected : ''}`}><i />{operation === 'connect' ? 'Connecting' : connected ? 'Connected' : 'Not connected'}</span><span className={styles.connectionDivider} /><code>{MCP_ENDPOINT}</code><span className={styles.transport}>Stateless HTTP</span></div><span className={styles.sessionAuth}><ShieldCheck size={12} />Workspace session</span></div>
    {error && <div className={styles.errorBanner} role="alert"><CircleAlert size={16} /><span>{error}</span>{signInRequired && <Link to="/login" state={{ from: '/agent-console' }} className={styles.signInLink}>Sign in</Link>}<button onClick={() => setError('')} aria-label="Dismiss console error"><X size={15} /></button></div>}
    <div className={styles.summaryGrid}><div><span><Layers size={14} />Available tools</span><strong>{connected ? tools.length : '—'}</strong></div><div><span><Play size={13} />Tool calls</span><strong>{toolCalls}<small>in this history</small></strong></div><div><span><Clock3 size={14} />Last response</span><strong>{lastResponse ? lastResponse.latencyMs : '—'}{lastResponse && <small>ms</small>}</strong></div><div><span><Bot size={14} />Server</span><strong className={styles.serverName}>{server ? server.name : 'Not connected'}{server?.version && <small>v{server.version}</small>}</strong></div></div>
    <div className={styles.pageTabs} role="group" aria-label="Console view">{([{ id: 'runner', label: 'Tool runner', icon: Code2 }, { id: 'history', label: 'Request history', icon: History }, { id: 'raw', label: 'Raw request', icon: Braces }] as const).map(item => <button key={item.id} className={tab === item.id ? styles.pageTabActive : ''} aria-pressed={tab === item.id} onClick={() => { setTab(item.id); setArgumentError(''); }}><item.icon size={14} />{item.label}{item.id === 'history' && history.length > 0 && <span>{history.length}</span>}</button>)}<span className={styles.protocol}>MCP {connected ? protocol : MCP_PROTOCOL}</span></div>
    {tab === 'runner' && <ToolRunner tools={tools} selectedTool={selectedTool} args={args} connected={connected} busy={busy} running={operation === 'tool'} catalogLoading={operation === 'catalog' || operation === 'connect'} onSelect={tool => { if (selectedTool?.name === tool.name) return; setSelectedTool(tool); setArgs(exampleArguments(tool)); setArgumentError(''); setSelectedRequest(null); }} onArgs={value => { setArgs(value); setArgumentError(''); }} onRun={runTool} onFormat={format} onRefresh={refreshCatalog} error={argumentError} />}
    {tab === 'raw' && <section className={styles.rawPanel}><div className={styles.sectionHeader}><div><Braces size={16} /><h2>JSON-RPC request</h2></div><button className={styles.formatButton} onClick={format} disabled={busy}><Braces size={12} />Format JSON</button></div><div className={styles.rawDescription}>Send a JSON-RPC request or notification directly to the connected workspace server.</div><div className={styles.codeEditor}><div className={styles.codeHeader}><span>request.json</span><span>JSON-RPC 2.0</span></div><textarea aria-label="Raw JSON-RPC request" value={raw} spellCheck={false} onChange={event => { setRaw(event.target.value); setArgumentError(''); }} disabled={busy} aria-invalid={!!argumentError} onKeyDown={event => { if ((event.ctrlKey || event.metaKey) && event.key === 'Enter') { event.preventDefault(); void runRaw(); } }} /></div>{argumentError && <div className={styles.inlineError} role="alert"><CircleAlert size={14} />{argumentError}</div>}<div className={styles.runBar}><span><Code2 size={12} />{MCP_ENDPOINT}</span><div><kbd>Ctrl ↵</kbd><button className={styles.primaryButton} onClick={runRaw} disabled={!connected || busy}>{operation === 'raw' ? <Loader2 size={14} className={styles.spinning} /> : <ArrowRight size={14} />}{operation === 'raw' ? 'Sending...' : 'Send request'}</button></div></div></section>}
    {tab === 'history' && <ConsoleHistory entries={history} onInspect={entry => { setSelectedRequest(entry.id); window.requestAnimationFrame(() => { resultRegion.current?.scrollIntoView({ block: 'nearest', behavior: window.matchMedia('(prefers-reduced-motion: reduce)').matches ? 'instant' : 'smooth' }); resultRegion.current?.focus({ preventScroll: true }); }); }} onCopy={copy} onClear={() => { setHistory([]); setSelectedRequest(null); setNotice('Console history cleared.'); }} />}
    <div ref={resultRegion} tabIndex={-1} className={styles.resultAnchor}><ConsoleResult entry={activeEntry} onCopy={copy} /></div>
    <footer className={styles.footer}><span><ShieldCheck size={12} />Tools run with your current workspace permissions</span><Link to="/agents">Agent access & API keys<ArrowUpRight size={12} /></Link></footer>
    <span className={styles.notice} role="status" aria-live="polite">{notice && <><Check size={13} />{notice}</>}</span>
  </div>;
}
