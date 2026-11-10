import { useSearchParams } from 'react-router-dom';
import { AdminTitle } from './AdminArea';
import { useState, useEffect, useCallback } from 'react';
import { Plus, Search, Key, Loader2, Trash2, Edit, Copy, Shield, Wifi, Globe, Database, TestTube, RefreshCw } from 'lucide-react';
import { api } from './api';
import { Table, Modal, FormField, Input, Textarea, Select, Badge, Pagination, SearchFilter, ToastContainer, Toast, ConfirmDialog, getStatusBadge } from './components';

interface ApiKey {
  id: string;
  name: string;
  prefix: string;
  agent_id: string;
  revoked_at?: string;
  scopes: string[];
  rate_limit: number;
  expires_at?: string;
  last_used_at?: string;
  created_at: string;
  created_by: string;
  active: boolean;
}

interface Agent {
  id: string;
  name: string;
  description: string;
  provider: string;
  enabled: boolean;
  scopes: string[];
  rate_limit?: number;
  last_used_at?: string;
  created_at: string;
  updated_at: string;
}

const DEFAULT_PAGE_SIZE = 25;

const scopeOptions = [
  { value: 'companies:read', label: 'Companies: Read' },
  { value: 'companies:write', label: 'Companies: Write' },
  { value: 'contacts:read', label: 'Contacts: Read' },
  { value: 'contacts:write', label: 'Contacts: Write' },
  { value: 'leads:read', label: 'Leads: Read' },
  { value: 'leads:write', label: 'Leads: Write' },
  { value: 'deals:read', label: 'Deals: Read' },
  { value: 'deals:write', label: 'Deals: Write' },
  { value: 'tasks:read', label: 'Tasks: Read' },
  { value: 'tasks:write', label: 'Tasks: Write' },
  { value: 'projects:read', label: 'Projects: Read' },
  { value: 'projects:write', label: 'Projects: Write' },
  { value: 'search:read', label: 'Search: Read' },
  { value: 'workflows:read', label: 'Workflows: Read' },
  { value: 'workflows:write', label: 'Workflows: Write' },
  { value: 'integrations:read', label: 'Integrations: Read' },
  { value: 'integrations:write', label: 'Integrations: Write' },
  { value: 'admin:read', label: 'Admin: Read' },
  { value: 'admin:write', label: 'Admin: Write' },
];

const agentTypes = [
  { value: 'chat', label: 'Chat Agent' },
  { value: 'worker', label: 'Background Worker' },
  { value: 'webhook', label: 'Webhook Handler' },
  { value: 'mcp', label: 'MCP Server' },
];

export function AgentsPage() {
  const [apiKeys, setApiKeys] = useState<ApiKey[]>([]);
  const [agents, setAgents] = useState<Agent[]>([]);
  const [totalKeys, setTotalKeys] = useState(0);
  const [totalAgents, setTotalAgents] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize] = useState(DEFAULT_PAGE_SIZE);
  const [query, setQuery] = useState('');
  const [params] = useSearchParams();
  const [activeTab, setActiveTab] = useState<'keys' | 'agents'>(() => params.get('tab') === 'agents' ? 'agents' : 'keys');
  const [loading, setLoading] = useState(false);
  const [toasts, setToasts] = useState<Toast[]>([]);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingKey, setEditingKey] = useState<ApiKey | null>(null);
  const [editingAgent, setEditingAgent] = useState<Agent | null>(null);
  const [formData, setFormData] = useState({
    name: '',
    scopes: [] as string[],
    rate_limit: '1000',
    expires_at: '',
    description: '',
    type: 'chat' as 'chat' | 'worker' | 'webhook' | 'mcp',
    config: '{}',
  });
  const [formErrors, setFormErrors] = useState<Record<string, string>>({});
  const [submitLoading, setSubmitLoading] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<ApiKey | Agent | null>(null);
  const [deleteType, setDeleteType] = useState<'key' | 'agent' | null>(null);
  const [selectedAgentID, setSelectedAgentID] = useState('');
  const [generatedKey, setGeneratedKey] = useState<string | null>(null);

  const totalPages = Math.ceil((activeTab === 'keys' ? totalKeys : totalAgents) / pageSize);

  const showToast = useCallback((type: Toast['type'], message: string) => {
    const id = Date.now();
    setToasts((prev) => [...prev, { id, type, message }]);
    setTimeout(() => setToasts((prev) => prev.filter((t) => t.id !== id)), 5000);
  }, []);

  const fetchApiKeys = useCallback(async () => {
    setLoading(true);
    try {
      const response = await api.get<{ items: ApiKey[]; total: number }>('/api-keys', {
        params: { q: query, limit: pageSize, offset: (page - 1) * pageSize },
      });
      setApiKeys(response.items || []);
      setTotalKeys(response.total || 0);
    } catch (error) {
      console.error('Failed to fetch API keys:', error);
      showToast('error', 'Failed to load API keys');
    } finally {
      setLoading(false);
    }
  }, [query, page, pageSize, showToast]);

  const fetchAgents = useCallback(async () => {
    setLoading(true);
    try {
      const response = await api.get<{ items: Agent[]; total: number }>('/agents', {
        params: { q: query, limit: pageSize, offset: (page - 1) * pageSize },
      });
      setAgents(response.items || []);
      setTotalAgents(response.total || 0);
    } catch (error) {
      console.error('Failed to fetch agents:', error);
      showToast('error', 'Failed to load agents');
    } finally {
      setLoading(false);
    }
  }, [query, page, pageSize, showToast]);

  useEffect(() => {
    if (activeTab === 'keys') fetchApiKeys();
    fetchAgents();
  }, [activeTab, fetchApiKeys, fetchAgents]);

  const handleCreateKey = () => {
    setEditingKey(null);
    setSelectedAgentID(agents.find(a => a.enabled)?.id || '');
    setFormData({ name: '', scopes: [], rate_limit: '1000', expires_at: '', description: '', type: 'chat', config: '{}' });
    setFormErrors({});
    setGeneratedKey(null);
    setModalOpen(true);
  };

  const handleCreateAgent = () => {
    setEditingAgent(null);
    setFormData({ name: '', scopes: [], rate_limit: '1000', expires_at: '', description: '', type: 'chat', config: '{}' });
    setFormErrors({});
    setGeneratedKey(null);
    setModalOpen(true);
  };

  const handleEditKey = (key: ApiKey) => {
    setEditingKey(key);
    setSelectedAgentID(key.agent_id);
    setFormData({ name: key.name, scopes: key.scopes, rate_limit: String(key.rate_limit), expires_at: key.expires_at ? key.expires_at.split('T')[0] : '', description: '', type: 'chat', config: '{}' });
    setFormErrors({});
    setGeneratedKey(null);
    setModalOpen(true);
  };

  const handleEditAgent = (agent: Agent) => {
    setEditingAgent(agent);
    setFormData({ name: agent.name, scopes: agent.scopes || [], rate_limit: String(agent.rate_limit || 1000), expires_at: '', description: agent.description, type: agent.provider as typeof formData.type, config: '{}' });
    setFormErrors({});
    setGeneratedKey(null);
    setModalOpen(true);
  };

  const handleDelete = (item: ApiKey | Agent, type: 'key' | 'agent') => {
    setDeleteConfirm(item);
    setDeleteType(type);
  };

  const confirmDelete = async () => {
    if (!deleteConfirm || !deleteType) return;
    setSubmitLoading(true);
    try {
      if (deleteType === 'key') {
        await api.delete(`/api-keys/${deleteConfirm.id}`);
        showToast('success', 'API key deleted successfully');
        fetchApiKeys();
      } else {
        await api.delete(`/agents/${deleteConfirm.id}`);
        showToast('success', 'Agent deleted successfully');
        fetchAgents();
      }
    } catch (error) {
      showToast('error', 'Failed to delete');
    } finally {
      setSubmitLoading(false);
      setDeleteConfirm(null);
      setDeleteType(null);
    }
  };

  const validateForm = () => {
    const errors: Record<string, string> = {};
    if (!formData.name?.trim()) errors.name = 'Name is required';
    if (formData.scopes.length === 0) errors.scopes = 'At least one scope is required';
    if (activeTab === 'keys' && !selectedAgentID) errors.agent_id = 'Select an agent';
    if (activeTab === 'agents' && !formData.type) errors.type = 'Agent type is required';
    try { JSON.parse(formData.config); } catch { errors.config = 'Invalid JSON'; }
    const rateLimit = Number(formData.rate_limit);
    if (isNaN(rateLimit) || rateLimit < 1) errors.rate_limit = 'Rate limit must be positive';
    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async () => {
    if (!validateForm()) return;
    setSubmitLoading(true);
    try {
      let config;
      try { config = JSON.parse(formData.config); } catch { config = {}; }
      if (activeTab === 'keys') {
        const keyData = { name: formData.name, ...(editingKey ? {} : {agent_id: selectedAgentID}), scopes: formData.scopes, rate_limit: Number(formData.rate_limit), expires_at: formData.expires_at ? new Date(formData.expires_at + 'T23:59:59Z').toISOString() : (editingKey ? '' : undefined) };
        if (editingKey) {
          await api.patch(`/api-keys/${editingKey.id}`, keyData);
          showToast('success', 'API key updated successfully');
        } else {
          const response = await api.post<{ key: string; api_key: ApiKey }>('/api-keys/create', keyData);
          showToast('success', 'API key created successfully');
          setGeneratedKey(response.key);
        }
        fetchApiKeys();
      } else {
        const agentData = { name: formData.name, description: formData.description, provider: formData.type, enabled: editingAgent?.enabled ?? true, scopes: formData.scopes, rate_limit: Number(formData.rate_limit) };
        if (editingAgent) {
          await api.patch(`/agents/${editingAgent.id}`, agentData);
          showToast('success', 'Agent updated successfully');
        } else {
          await api.post('/agents/create', agentData);
          showToast('success', 'Agent created successfully');
        }
        fetchAgents();
      }
      if (activeTab !== 'keys' || editingKey) setModalOpen(false);
    } catch (error: any) {
      if (error.status === 400) {
        showToast('error', error.message || 'Validation failed');
      } else {
        showToast('error', 'Failed to save');
      }
    } finally {
      setSubmitLoading(false);
    }
  };

  const copyKey = () => {
    if (generatedKey) {
      navigator.clipboard.writeText(generatedKey);
      showToast('success', 'API key copied to clipboard');
    }
  };

  const keyColumns = [
    { key: 'name', header: 'Name', render: (k: ApiKey) => <span className="font-medium">{k.name}</span> },
    { key: 'key_prefix', header: 'Prefix', render: (k: ApiKey) => <code className="text-xs">{k.prefix}...</code> },
    { key: 'scopes', header: 'Scopes', render: (k: ApiKey) => <span className="text-sm">{k.scopes.join(', ')}</span> },
    { key: 'rate_limit', header: 'Rate Limit', render: (k: ApiKey) => `${k.rate_limit}/min` },
    { key: 'active', header: 'Status', render: (k: ApiKey) => <Badge variant={(!k.revoked_at && (!k.expires_at || new Date(k.expires_at).getTime() > Date.now())) ? 'success' : 'neutral'}>{(!k.revoked_at && (!k.expires_at || new Date(k.expires_at).getTime() > Date.now())) ? 'Active' : 'Inactive'}</Badge> },
    { key: 'last_used_at', header: 'Last Used', render: (k: ApiKey) => k.last_used_at ? new Date(k.last_used_at).toLocaleString() : 'Never' },
    { key: 'expires_at', header: 'Expires', render: (k: ApiKey) => k.expires_at ? new Date(k.expires_at).toLocaleDateString() : 'Never' },
  ];

  const agentColumns = [
    { key: 'name', header: 'Name', render: (a: Agent) => <span className="font-medium">{a.name}</span> },
    { key: 'provider', header: 'Type', render: (a: Agent) => <Badge>{a.provider}</Badge> },
    { key: 'enabled', header: 'Status', render: (a: Agent) => <Badge>{a.enabled ? 'Active' : 'Disabled'}</Badge> },
    { key: 'scopes', header: 'Scopes', render: (a: Agent) => a.scopes.join(', ') },
    { key: 'last_used_at', header: 'Last Used', render: (a: Agent) => a.last_used_at ? new Date(a.last_used_at).toLocaleString() : 'Never' },
  ];

  return (
    <div className="space-y-6">
      <div className="ad-page-header">
        <AdminTitle title="Access & agents" description="Manage agents, API keys, and the permissions they use." />
        <button className="btn-primary" onClick={activeTab === 'keys' ? handleCreateKey : handleCreateAgent}>
          <Plus className="w-4 h-4 mr-2" />
          {activeTab === 'keys' ? 'New API Key' : 'New Agent'}
        </button>
      </div>

      <div className="border-b border-neutral-200 dark:border-neutral-800">
        <nav className="flex gap-1 px-1" aria-label="Tabs">
          <button
            aria-pressed={activeTab === 'keys'} onClick={() => { setActiveTab('keys'); setPage(1); }}
            className={`px-4 py-2 text-sm font-medium rounded-t-lg transition-colors ${activeTab === 'keys' ? 'bg-white dark:bg-neutral-900 text-black dark:text-white border-b-2 border-black dark:border-white' : 'text-neutral-500 hover:text-neutral-700 dark:hover:text-neutral-300'}`}
          >
            API Keys
          </button>
          <button
            aria-pressed={activeTab === 'agents'} onClick={() => { setActiveTab('agents'); setPage(1); }}
            className={`px-4 py-2 text-sm font-medium rounded-t-lg transition-colors ${activeTab === 'agents' ? 'bg-white dark:bg-neutral-900 text-black dark:text-white border-b-2 border-black dark:border-white' : 'text-neutral-500 hover:text-neutral-700 dark:hover:text-neutral-300'}`}
          >
            Agents
          </button>
        </nav>
      </div>

      {activeTab === 'keys' && (
        <>
          <SearchFilter
            query={query}
            onQueryChange={setQuery}
            placeholder="Search API keys..."
            onFilterChange={() => setPage(1)}
          />

          <Table
            items={apiKeys}
            columns={keyColumns}
            onEdit={handleEditKey}
            onDelete={(k) => handleDelete(k, 'key')}
            loading={loading}
            emptyMessage="No API keys found. Create your first key to get started."
          />

          <Pagination
            currentPage={page}
            totalPages={totalPages}
            totalItems={totalKeys}
            pageSize={pageSize}
            onPageChange={setPage}
          />
        </>
      )}

      {activeTab === 'agents' && (
        <>
          <SearchFilter
            query={query}
            onQueryChange={setQuery}
            placeholder="Search agents..."
            onFilterChange={() => setPage(1)}
          />

          <Table
            items={agents}
            columns={agentColumns}
            onEdit={handleEditAgent}
            onDelete={(a) => handleDelete(a, 'agent')}
            loading={loading}
            emptyMessage="No agents found. Create your first agent to get started."
          />

          <Pagination
            currentPage={page}
            totalPages={totalPages}
            totalItems={totalAgents}
            pageSize={pageSize}
            onPageChange={setPage}
          />
        </>
      )}

      <Modal isOpen={modalOpen} onClose={() => { setModalOpen(false); setGeneratedKey(null); }} title={activeTab === 'keys' ? (editingKey ? 'Edit API Key' : 'New API Key') : (editingAgent ? 'Edit Agent' : 'New Agent')}>
        <div className="space-y-4 max-h-[70vh] overflow-y-auto">
          <FormField label="Name" error={formErrors.name} required>
            <Input value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} placeholder={activeTab === 'keys' ? 'API Key name' : 'Agent name'} />
          </FormField>

              <FormField label="Scopes" error={formErrors.scopes} required>
                <div className="flex flex-wrap gap-2">
                  {scopeOptions.map((scope) => (
                    <label key={scope.value} className="flex items-center gap-2 px-3 py-1.5 bg-neutral-50 dark:bg-neutral-800 rounded-lg border border-neutral-200 dark:border-neutral-700 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={formData.scopes.includes(scope.value)}
                        onChange={(e) => setFormData({ ...formData, scopes: e.target.checked ? [...formData.scopes, scope.value] : formData.scopes.filter(s => s !== scope.value) })}
                        className="w-4 h-4 accent-black"
                      />
                      <span className="text-sm">{scope.label}</span>
                    </label>
                  ))}
                </div>
              </FormField>
          {activeTab === 'keys' ? (
            <>
              {!editingKey && <FormField label="Agent" error={formErrors.agent_id} required><Select value={selectedAgentID} onChange={e => setSelectedAgentID(e.target.value)}><option value="">Select agent</option>{agents.filter(a => a.enabled).map(a => <option key={a.id} value={a.id}>{a.name}</option>)}</Select></FormField>}

              <FormField label="Rate Limit (requests/min)" error={formErrors.rate_limit}>
                <Input type="number" min="1" value={formData.rate_limit} onChange={(e) => setFormData({ ...formData, rate_limit: e.target.value })} placeholder="1000" />
              </FormField>
              <FormField label="Expires At (optional)">
                <Input type="date" value={formData.expires_at} onChange={(e) => setFormData({ ...formData, expires_at: e.target.value })} />
              </FormField>
            </>
          ) : (
            <>
              <FormField label="Description">
                <Textarea value={formData.description} onChange={(e) => setFormData({ ...formData, description: e.target.value })} placeholder="Agent description" rows={3} />
              </FormField>
              <FormField label="Type" error={formErrors.type} required>
                <Select value={formData.type} onChange={(e) => setFormData({ ...formData, type: e.target.value as typeof formData.type })}>
                  {agentTypes.map((opt) => <option key={opt.value} value={opt.value}>{opt.label}</option>)}
                </Select>
              </FormField>

            </>
          )}

          {generatedKey && (
            <div className="p-4 bg-neutral-50 dark:bg-neutral-800 rounded-lg border border-black/50 dark:border-white/50">
              <div className="flex items-center justify-between mb-2">
                <span className="font-medium text-black dark:text-white">Generated API Key (copy now - won&apos;t be shown again)</span>
                <button onClick={copyKey} className="btn-secondary text-sm">
                  <Copy className="w-4 h-4 mr-1" /> Copy
                </button>
              </div>
              <code className="text-xs break-all bg-neutral-100 dark:bg-neutral-900 p-2 rounded block">{generatedKey}</code>
            </div>
          )}
        </div>
        <div className="modal-footer">
          <button onClick={() => { setModalOpen(false); setGeneratedKey(null); }} className="btn-secondary" disabled={submitLoading}>Cancel</button>
          <button onClick={handleSubmit} className="btn-primary" disabled={submitLoading || !!generatedKey}>
            {submitLoading ? <Loader2 className="loading-spinner w-4 h-4 mr-2" /> : (editingKey || editingAgent ? 'Update' : 'Create')}
          </button>
        </div>
      </Modal>

      <ConfirmDialog
        isOpen={!!deleteConfirm}
        onClose={() => { setDeleteConfirm(null); setDeleteType(null); }}
        onConfirm={confirmDelete}
        title={deleteType === 'key' ? 'Delete API Key' : 'Delete Agent'}
        message={`Are you sure you want to delete "${deleteConfirm?.name}"? This action cannot be undone.`}
        confirmText="Delete"
        variant="danger"
        loading={submitLoading}
      />

      <ToastContainer toasts={toasts} onRemove={(id) => setToasts((prev) => prev.filter((t) => t.id !== id))} />
    </div>
  );
}
