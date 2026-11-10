import { AdminTitle } from './AdminArea';
import { useState, useEffect, useCallback } from 'react';
import { Plus, Search, Zap, Loader2, Trash2, Edit, TestTube, Shield, Key, Globe } from 'lucide-react';
import { api } from './api';
import { Table, Modal, FormField, Input, Textarea, Select, Badge, Pagination, SearchFilter, ToastContainer, Toast, ConfirmDialog, getStatusBadge } from './components';

interface Webhook {
  id: string;
  workspace_id: string;
  name: string;
  url: string;
  events: string[];
  secret: string;
  status: 'active' | 'inactive' | 'disabled';
  failure_count: number;
  last_triggered_at: string | null;
  last_success_at: string | null;
  last_failure_at: string | null;
  last_failure_reason: string;
  created_at: string;
  updated_at: string;
}

const DEFAULT_PAGE_SIZE = 25;

const statusOptions = [
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
  { value: 'disabled', label: 'Disabled' },
];

export function WebhooksPage() {
  const [webhooks, setWebhooks] = useState<Webhook[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [query, setQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [loading, setLoading] = useState(false);
  const [toasts, setToasts] = useState<Toast[]>([]);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingWebhook, setEditingWebhook] = useState<Webhook | null>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<Webhook | null>(null);
  const [testLoading, setTestLoading] = useState<string | null>(null);

  interface WebhookFormData {
    name: string;
    url: string;
    events: string;
    secret: string;
    status: string;
  }
  const [formData, setFormData] = useState<WebhookFormData>({
    name: '',
    url: '',
    events: '',
    secret: '',
    status: 'active',
  });
  const [formErrors, setFormErrors] = useState<Record<string, string>>({});
  const [submitLoading, setSubmitLoading] = useState(false);

  const totalPages = Math.ceil(total / pageSize);

  const showToast = useCallback((type: Toast['type'], message: string) => {
    const id = Date.now();
    setToasts((prev) => [...prev, { id, type, message }]);
    setTimeout(() => setToasts((prev) => prev.filter((t) => t.id !== id)), 5000);
  }, []);

  const fetchWebhooks = useCallback(async () => {
    setLoading(true);
    try {
      const response = await api.webhooks.list({
        status: statusFilter || undefined,
        limit: pageSize,
        offset: (page - 1) * pageSize,
      });
      setWebhooks(response.items || []);
      setTotal(response.total || 0);
    } catch (error) {
      console.error('Failed to fetch webhooks:', error);
      showToast('error', 'Failed to load webhooks');
    } finally {
      setLoading(false);
    }
  }, [statusFilter, page, pageSize, showToast]);

  useEffect(() => {
    fetchWebhooks();
  }, [fetchWebhooks]);

  const handleCreate = () => {
    setEditingWebhook(null);
    setFormData({
      name: '',
      url: '',
      events: '["lead.created", "deal.updated"]',
      secret: '',
      status: 'active',
    });
    setFormErrors({});
    setModalOpen(true);
  };

  const handleEdit = (webhook: Webhook) => {
    setEditingWebhook(webhook);
    setFormData({
      name: webhook.name,
      url: webhook.url,
      events: JSON.stringify(webhook.events, null, 2),
      secret: '',
      status: webhook.status,
    });
    setFormErrors({});
    setModalOpen(true);
  };

  const handleDelete = (webhook: Webhook) => {
    setDeleteConfirm(webhook);
  };

  const confirmDelete = async () => {
    if (!deleteConfirm) return;
    setSubmitLoading(true);
    try {
      await api.webhooks.archive(deleteConfirm.id);
      showToast('success', 'Webhook deleted successfully');
      fetchWebhooks();
    } catch (error) {
      showToast('error', 'Failed to delete webhook');
    } finally {
      setSubmitLoading(false);
      setDeleteConfirm(null);
    }
  };

  const handleTest = async (webhook: Webhook) => {
    setTestLoading(webhook.id);
    try {
      await api.webhooks.test(webhook.id);
      showToast('success', 'Test webhook sent successfully');
      fetchWebhooks();
    } catch (error) {
      showToast('error', 'Failed to test webhook');
    } finally {
      setTestLoading(null);
    }
  };

  const validateForm = () => {
    const errors: Record<string, string> = {};
    if (!formData.name?.trim()) errors.name = 'Webhook name is required';
    if (!formData.url?.trim()) errors.url = 'URL is required';
    else {
      try { new URL(formData.url); } catch { errors.url = 'Invalid URL'; }
    }
    if (!formData.events?.trim()) errors.events = 'Events are required';
    else {
      try { JSON.parse(formData.events); } catch { errors.events = 'Invalid JSON array'; }
    }
    if (!editingWebhook && !formData.secret?.trim()) errors.secret = 'Secret is required';
    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async () => {
    if (!validateForm()) return;
    setSubmitLoading(true);
    try {
      let events;
      try { events = JSON.parse(formData.events); } catch { events = []; }

      const webhookData = {
        name: formData.name,
        url: formData.url,
        events,
        ...(formData.secret.trim() ? { secret: formData.secret } : {}),
        status: formData.status,
      };
      if (editingWebhook) {
        await api.webhooks.update(editingWebhook.id, webhookData);
        showToast('success', 'Webhook updated successfully');
      } else {
        await api.webhooks.create({ ...webhookData, secret: formData.secret });
        showToast('success', 'Webhook created successfully');
      }
      setModalOpen(false);
      fetchWebhooks();
    } catch (error: any) {
      if (error.status === 400) {
        showToast('error', error.message || 'Validation failed');
      } else {
        showToast('error', 'Failed to save webhook');
      }
    } finally {
      setSubmitLoading(false);
    }
  };

  const webhookColumns = [
    { key: 'name', header: 'Name', render: (w: Webhook) => <span className="font-medium">{w.name}</span> },
    { key: 'url', header: 'URL', render: (w: Webhook) => <span className="max-w-xs truncate block font-mono text-sm">{w.url}</span> },
    { key: 'events', header: 'Events', render: (w: Webhook) => (
      <span className="max-w-xs truncate block">{w.events.join(', ')}</span>
    )},
    { key: 'status', header: 'Status', render: (w: Webhook) => <Badge variant={getStatusBadge(w.status)}>{w.status}</Badge> },
    { key: 'failure_count', header: 'Failures', render: (w: Webhook) => w.failure_count > 0 ? <span className="text-red-500 font-medium">{w.failure_count}</span> : '0' },
    { key: 'last_triggered_at', header: 'Last Triggered', render: (w: Webhook) => w.last_triggered_at ? new Date(w.last_triggered_at).toLocaleString() : 'Never' },
    { key: 'last_success_at', header: 'Last Success', render: (w: Webhook) => w.last_success_at ? new Date(w.last_success_at).toLocaleString() : 'Never' },
    { key: 'actions', header: 'Actions', render: (w: Webhook) => (
      <div className="flex items-center gap-1">
        <button onClick={() => handleTest(w)} className="p-1.5 text-neutral-400 hover:text-blue-500" title="Test" disabled={testLoading === w.id}>
          {testLoading === w.id ? <Loader2 className="loading-spinner w-4 h-4" /> : <TestTube className="w-4 h-4" />}
        </button>
        <button onClick={() => handleEdit(w)} className="p-1.5 text-neutral-400 hover:text-black dark:hover:text-white" title="Edit"><Edit className="w-4 h-4" /></button>
        <button onClick={() => handleDelete(w)} className="p-1.5 text-neutral-400 hover:text-red-500" title="Delete"><Trash2 className="w-4 h-4" /></button>
      </div>
    )},
  ];

  return (
    <div className="space-y-6">
      <div className="ad-page-header">
        <AdminTitle title="Webhooks" description="Connect CRM events to your services and monitor deliveries." />
        <button className="btn-primary" onClick={handleCreate}>
          <Plus className="w-4 h-4 mr-2" />
          New Webhook
        </button>
      </div>

      <SearchFilter
        query={query}
        onQueryChange={setQuery}
        placeholder="Search webhooks..."
        filters={[{ label: 'Status', value: 'status', options: statusOptions }]}
        filterValues={{ status: statusFilter }}
        onFilterChange={(key, value) => { if (key === 'status') setStatusFilter(value); setPage(1); }}
      />

      <Table
        items={webhooks}
        columns={webhookColumns}
        onEdit={handleEdit}
        onDelete={handleDelete}
        loading={loading}
        emptyMessage="No webhooks found. Create your first webhook to get started."
      />

      <Pagination
            currentPage={page}
            totalPages={totalPages}
            totalItems={total}
            pageSize={pageSize}
            onPageChange={setPage}
            onPageSizeChange={(size) => { setPageSize(size); setPage(1); }}
            showPageSize
          />

      <Modal isOpen={modalOpen} onClose={() => setModalOpen(false)} title={editingWebhook ? 'Edit Webhook' : 'New Webhook'}>
        <div className="space-y-4 max-h-[70vh] overflow-y-auto">
          <FormField label="Name" error={formErrors.name} required>
            <Input value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} placeholder="Webhook name" />
          </FormField>
          <FormField label="URL" error={formErrors.url} required>
            <Input value={formData.url} onChange={(e) => setFormData({ ...formData, url: e.target.value })} placeholder="https://example.com/webhook" type="url" />
          </FormField>
          <FormField label="Events (JSON Array)" error={formErrors.events} required>
            <Textarea
              value={formData.events}
              onChange={(e) => setFormData({ ...formData, events: e.target.value })}
              placeholder='["lead.created", "deal.updated", "contact.created"]'
              rows={4}
              className="font-mono text-sm"
            />
          </FormField>
          <FormField label="Secret" error={formErrors.secret} required>
            <Input value={formData.secret} onChange={(e) => setFormData({ ...formData, secret: e.target.value })} placeholder="Webhook secret for signature verification" type="password" />
          </FormField>
          <FormField label="Status">
            <Select value={formData.status} onChange={(e) => setFormData({ ...formData, status: e.target.value })}>
              {statusOptions.map((opt) => <option key={opt.value} value={opt.value}>{opt.label}</option>)}
            </Select>
          </FormField>
        </div>
        <div className="modal-footer">
          <button onClick={() => setModalOpen(false)} className="btn-secondary" disabled={submitLoading}>Cancel</button>
          <button onClick={handleSubmit} className="btn-primary" disabled={submitLoading}>
            {submitLoading ? <Loader2 className="loading-spinner w-4 h-4 mr-2" /> : (editingWebhook ? 'Update' : 'Create')}
          </button>
        </div>
      </Modal>

      <ConfirmDialog
        isOpen={!!deleteConfirm}
        onClose={() => setDeleteConfirm(null)}
        onConfirm={confirmDelete}
        title="Delete Webhook"
        message={`Are you sure you want to delete "${deleteConfirm?.name}"? This action cannot be undone.`}
        confirmText="Delete"
        variant="danger"
        loading={submitLoading}
      />

      <ToastContainer toasts={toasts} onRemove={(id) => setToasts((prev) => prev.filter((t) => t.id !== id))} />
    </div>
  );
}