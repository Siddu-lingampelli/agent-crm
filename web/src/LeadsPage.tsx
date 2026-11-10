import { crmMoney, exportCrmPage } from './crm-data';
import { CrmHeader, CrmSummary, CrmState, CrmPagination, CrmDialog as Modal } from './CrmUI';
import { workBatch, workDate, useWorkToday } from './work-data';
import { useRecordRoute } from './useRecordRoute';
import { useState, useEffect, useCallback, useRef } from 'react';
import { Calendar, Table as TableIcon, Kanban as KanbanIcon } from 'lucide-react';
import { api, Lead, LeadCreateInput, LeadConvertInput, type PipelineChoice } from './api';
import { Table, FormField, Input, Textarea, Select, Badge, ToastContainer, Toast, ConfirmDialog, getStatusBadge, Button, DataTableToolbar, Avatar, KanbanBoard } from './components';

const DEFAULT_PAGE_SIZE = 25;

const statusOptions = [
  { value: 'new', label: 'New' },
  { value: 'qualified', label: 'Qualified' },
  { value: 'proposal', label: 'Proposal' },
  { value: 'negotiation', label: 'Negotiation' },
  { value: 'won', label: 'Won' },
  { value: 'lost', label: 'Lost' },
];

const priorityOptions = [
  { value: 'low', label: 'Low' },
  { value: 'medium', label: 'Medium' },
  { value: 'high', label: 'High' },
];

const sourceOptions = [
  { value: 'website', label: 'Website' },
  { value: 'referral', label: 'Referral' },
  { value: 'cold_call', label: 'Cold Call' },
  { value: 'email', label: 'Email' },
  { value: 'social', label: 'Social Media' },
  { value: 'event', label: 'Event' },
  { value: 'other', label: 'Other' },
];

export function LeadsPage() {
  const today = useWorkToday();
  const [leads, setLeads] = useState<Lead[]>([]);
  const [pipelines, setPipelines] = useState<PipelineChoice[]>([]);
  const [conversionError, setConversionError] = useState('');
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [query, setQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [sourceFilter, setSourceFilter] = useState('');
  const [priorityFilter, setPriorityFilter] = useState('');
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const requestSequence = useRef(0);
  const [toasts, setToasts] = useState<Toast[]>([]);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingLead, setEditingLead] = useState<Lead | null>(null);
  const [formData, setFormData] = useState<LeadCreateInput>({
    display_name: '', first_name: '', last_name: '', email: '', phone: '',
    company_name: '', company_id: '', job_title: '', status: 'new',
    source: 'website', owner_id: '', priority: 'medium',
    estimated_value: 0, currency: 'USD', lead_score: 0,
    expected_close: '', follow_up_date: '', tags: [],
    custom_fields: {}, description: '', qualification: '',
  });
  const [formErrors, setFormErrors] = useState<Partial<LeadCreateInput>>({});
  const [submitLoading, setSubmitLoading] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<Lead | null>(null);
  const [convertModalOpen, setConvertModalOpen] = useState(false);
  const [convertingLead, setConvertingLead] = useState<Lead | null>(null);
  const [convertData, setConvertData] = useState<LeadConvertInput>({
    contact_id: '', company_id: '', deal_name: '', pipeline_id: '',
    stage_id: '', amount: 0, currency: 'USD', expected_close: '',
  });
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [viewMode, setViewMode] = useState<'table' | 'kanban'>('table');

  useEffect(() => { setSelectedIds(new Set()); }, [query, statusFilter, sourceFilter, priorityFilter, page, pageSize]);

  const showToast = useCallback((type: Toast['type'], message: string) => {
    const id = Date.now() + Math.random();
    setToasts((prev) => [...prev, { id, type, message }]);
    setTimeout(() => setToasts((prev) => prev.filter((t) => t.id !== id)), 5000);
  }, []);

  const fetchLeads = useCallback(async () => {
    const sequence = ++requestSequence.current;
    setLoading(true);
    setLoadError('');
    try {
      const response = await api.leads.list({
        q: query,
        status: statusFilter || undefined,
        source: sourceFilter || undefined,
        priority: priorityFilter || undefined,
        limit: pageSize,
        offset: (page - 1) * pageSize,
      });
      if (sequence !== requestSequence.current) return;
      if (page > Math.max(1, Math.ceil(response.total / pageSize))) { setPage(Math.max(1, Math.ceil(response.total / pageSize))); return; }
      setLeads(response.items);
      setTotal(response.total);
    } catch {
      if (sequence !== requestSequence.current) return;
      setLoadError('Failed to load leads');
      setLeads([]);
    } finally {
      if (sequence === requestSequence.current) setLoading(false);
    }
  }, [query, statusFilter, sourceFilter, priorityFilter, page, pageSize]);

  useEffect(() => { void fetchLeads(); }, [fetchLeads]);
  useEffect(() => {
    let active = true;
    api.get<{ items: PipelineChoice[] }>('/pipelines').then(result => { if (active) setPipelines(result.items ?? []); }).catch(() => { if (active) setConversionError('Could not load pipelines. Refresh the page to retry.'); });
    return () => { active = false; };
  }, []);

  const handleCreate = () => {
    setEditingLead(null);
    setFormData({
      display_name: '', first_name: '', last_name: '', email: '', phone: '',
      company_name: '', company_id: '', job_title: '', status: 'new',
      source: 'website', owner_id: '', priority: 'medium',
      estimated_value: 0, currency: 'USD', lead_score: 0,
      expected_close: '', follow_up_date: '', tags: [],
      custom_fields: {}, description: '', qualification: '',
    });
    setFormErrors({});
    setModalOpen(true);
  };

  const handleEdit = (lead: Lead) => {
    setEditingLead(lead);
    setFormData({
      display_name: lead.display_name, first_name: lead.first_name,
      last_name: lead.last_name, email: lead.email, phone: lead.phone || '',
      company_name: lead.company_name || '', company_id: lead.company_id || '',
      job_title: lead.job_title || '', status: lead.status,
      source: lead.source, owner_id: lead.owner_id || '',
      priority: lead.priority, estimated_value: lead.estimated_value || 0,
      currency: lead.currency, lead_score: lead.lead_score || 0,
      expected_close: lead.expected_close?.slice(0, 10) || '', follow_up_date: lead.follow_up_date?.slice(0, 10) || '',
      tags: lead.tags || [], custom_fields: lead.custom_fields || {},
      description: lead.description || '', qualification: lead.qualification || '',
    });
    setFormErrors({});
    setModalOpen(true);
  };

  const handleConvert = (lead: Lead) => {
    const pipeline = pipelines.find(item => item.is_default) || pipelines[0];
    setConvertingLead(lead);
    setConvertData({
      contact_id: '', company_id: lead.company_id || '',
      deal_name: `${lead.company_name || lead.display_name} - Deal`,
      pipeline_id: pipeline?.id || '', stage_id: pipeline?.stages[0]?.id || '', amount: lead.estimated_value || 0,
      currency: lead.currency,
      expected_close: lead.expected_close?.slice(0, 10) || new Date(Date.now() + 30 * 24 * 60 * 60 * 1000).toISOString().split('T')[0],
    });
    if (pipelines.length) setConversionError('');
    setModalOpen(false);
    setConvertModalOpen(true);
  };

  const handleDelete = (lead: Lead) => setDeleteConfirm(lead);

  const confirmDelete = async () => {
    if (!deleteConfirm) return;
    setSubmitLoading(true);
    try {
      await api.leads.archive(deleteConfirm.id);
      showToast('success', 'Lead archived successfully');
      fetchLeads();
    } catch {
      showToast('error', 'Failed to archive lead');
    } finally {
      setSubmitLoading(false);
      setDeleteConfirm(null);
    }
  };

  const handleBulkArchive = async () => {
    if (submitLoading) return;
    setSubmitLoading(true);
    try {
      const result = await workBatch([...selectedIds], api.leads.archive);
      setSelectedIds(new Set(result.failed));
      showToast(result.failed.length ? 'error' : 'success', result.succeeded.length + ' archived' + (result.failed.length ? '; ' + result.failed.length + ' failed. Retry the selected records.' : ' successfully'));
      await fetchLeads();
    } finally { setSubmitLoading(false); }
  };
  const validateForm = () => {
    const errors: Partial<LeadCreateInput> = {};
    if (!formData.display_name.trim()) errors.display_name = 'Display name is required';
    if (!formData.first_name.trim()) errors.first_name = 'First name is required';
    if (!formData.last_name.trim()) errors.last_name = 'Last name is required';
    if (!formData.email.trim()) errors.email = 'Email is required';
    else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(formData.email)) errors.email = 'Invalid email format';
    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async () => {
    if (!validateForm()) return;
    setSubmitLoading(true);
    try {
      if (editingLead) {
        await api.leads.update(editingLead.id, formData);
        showToast('success', 'Lead updated successfully');
      } else {
        await api.leads.create(formData);
        showToast('success', 'Lead created successfully');
      }
      setModalOpen(false);
      fetchLeads();
    } catch (error: any) {
      if (error.status === 400) showToast('error', error.message);
      else showToast('error', 'Failed to save lead');
    } finally {
      setSubmitLoading(false);
    }
  };

  const handleConvertSubmit = async () => {
    if (!convertingLead) return;
    if (!convertData.deal_name.trim() || !convertData.pipeline_id || !convertData.stage_id || !convertData.expected_close || !Number.isFinite(convertData.amount) || convertData.amount < 0) {
      setConversionError('Choose a pipeline and stage, enter a deal name, close date, and a valid amount.');
      return;
    }
    setConversionError('');
    setSubmitLoading(true);
    try {
      await api.leads.convert(convertingLead.id, convertData);
      showToast('success', 'Lead converted successfully');
      setConvertModalOpen(false);
      fetchLeads();
    } catch (error: any) {
      if (error.status === 400) showToast('error', error.message);
      else showToast('error', 'Failed to convert lead');
    } finally {
      setSubmitLoading(false);
    }
  };

  const columns = [
    { key: 'display_name', header: 'Name', render: (l: Lead) => (
      <div className="flex items-center gap-3">
        <Avatar name={l.display_name} size="sm" />
        <div>
          <button className="crm-table-name" onClick={e => { e.stopPropagation(); handleEdit(l); }}>{l.display_name}</button>
          {l.job_title && <p className="text-xs text-tertiary">{l.job_title}</p>}
        </div>
      </div>
    )},
    { key: 'email', header: 'Email', render: (l: Lead) => <a href={`mailto:${l.email}`} onClick={event => event.stopPropagation()} className="text-link hover:underline">{l.email}</a> },
    { key: 'company_name', header: 'Company' },
    { key: 'status', header: 'Status', render: (l: Lead) => <Badge variant={getStatusBadge(l.status)} size="sm" dot>{l.status}</Badge> },
    { key: 'priority', header: 'Priority', render: (l: Lead) => <Badge variant={getStatusBadge(l.priority)} size="sm">{l.priority}</Badge> },
    { key: 'source', header: 'Source', render: (l: Lead) => <span className="text-secondary capitalize">{(l.source || 'Not specified').replaceAll('_', ' ')}</span> },
    { key: 'estimated_value', header: 'Est. Value', render: (l: Lead) => l.estimated_value != null ? (
      <span className="font-medium">{crmMoney(l.estimated_value, l.currency)}</span>
    ) : <span className="text-tertiary">—</span> },
    { key: 'lead_score', header: 'Score', render: (l: Lead) => <span className="font-mono text-sm">{l.lead_score}</span> },
    { key: 'follow_up_date', header: 'Follow Up', render: (l: Lead) => l.follow_up_date ? workDate(l.follow_up_date) : '—' },
    { key: 'created_at', header: 'Created', render: (l: Lead) => workDate(l.created_at) },
  ];

  const filters = [
    { label: 'Status', value: 'status', options: statusOptions },
    { label: 'Source', value: 'source', options: sourceOptions },
    { label: 'Priority', value: 'priority', options: priorityOptions },
  ];

  const filterValues = { status: statusFilter, source: sourceFilter, priority: priorityFilter };
  const hasActiveFilters = !!statusFilter || !!sourceFilter || !!priorityFilter;

  const handleFilterChange = (key: string, value: string) => {
    if (key === 'status') setStatusFilter(value);
    if (key === 'source') setSourceFilter(value);
    if (key === 'priority') setPriorityFilter(value);
    setPage(1);
  };

  const clearFilters = () => {
    setStatusFilter('');
    setSourceFilter('');
    setPriorityFilter('');
    setPage(1);
  };

  const handleSelectionChange = (ids: Set<string>) => setSelectedIds(ids);

  const kanbanColumns = [
    { id: 'new', title: 'New', color: '#695be8', items: leads.filter(l => l.status === 'new') },
    { id: 'qualified', title: 'Qualified', color: '#503ebd', items: leads.filter(l => l.status === 'qualified') },
    { id: 'proposal', title: 'Proposal', color: '#ff4f00', items: leads.filter(l => l.status === 'proposal') },
    { id: 'negotiation', title: 'Negotiation', color: '#201515', items: leads.filter(l => l.status === 'negotiation') },
    { id: 'won', title: 'Won', color: '#156123', items: leads.filter(l => l.status === 'won') },
    { id: 'lost', title: 'Lost', color: '#503ebd', items: leads.filter(l => l.status === 'lost') },
  ];

  useRecordRoute({ getRecord: api.leads.get, onCreate: handleCreate, onRecord: handleEdit, onError: message => showToast('error', message) });

  return (
    <div className="crm-area">
      <CrmHeader title="Leads" description="Turn the next conversation into a lasting relationship." onCreate={handleCreate} busy={loading || !!loadError || !leads.length} onExport={() => exportCrmPage('leads', leads, [{ key: 'display_name', label: 'display name' }, { key: 'email', label: 'email' }, { key: 'company_name', label: 'company name' }, { key: 'status', label: 'status' }, { key: 'source', label: 'source' }, { key: 'priority', label: 'priority' }, { key: 'estimated_value', label: 'estimated value' }, { key: 'currency', label: 'currency' }])} />
      <CrmSummary loading={loading || !!loadError} items={[{ label: 'Matching leads', value: total, note: 'With current filters' }, { label: 'Qualified leads', value: leads.filter(l => l.status === 'qualified').length, note: 'On this page' }, { label: 'Needs a follow-up', value: leads.filter(l => l.follow_up_date && l.follow_up_date.slice(0, 10) <= today && !['won', 'lost'].includes(l.status)).length, note: 'Due today or earlier, on this page' }]} />

      <DataTableToolbar
        title="All Leads"
        description={`${leads.length} loaded on this page`}
        searchProps={{ query, onChange: (value: string) => { setQuery(value); setPage(1); }, placeholder: 'Search leads by name, email, company...' }}
        filterProps={{ filters, values: filterValues, onChange: handleFilterChange, onClear: clearFilters, hasActiveFilters }}
        viewOptions={{
          currentView: viewMode,
          views: [
            { id: 'table', label: 'Table', icon: <TableIcon className="w-4 h-4" /> },
            { id: 'kanban', label: 'Kanban', icon: <KanbanIcon className="w-4 h-4" /> },
          ],
          onChange: (view: string) => setViewMode(view as 'table' | 'kanban')
        }}
        bulkActions={selectedIds.size > 0 && !submitLoading ? {
          selectedCount: selectedIds.size,
          actions: [
            { label: 'Archive', onClick: handleBulkArchive, variant: 'danger' },
            { label: 'Clear selection', onClick: () => setSelectedIds(new Set()) },
          ]
        } : undefined}
      />

      {viewMode === 'kanban' && !loading && !loadError && <p className="crm-board-note">Board shows the leads on this page. Use the page controls to see more. Drag a card or use its Move to menu.</p>}
      <CrmState loading={loading} error={loadError} empty={!leads.length} onRetry={fetchLeads} onCreate={handleCreate} noun="leads" filtered={!!query || hasActiveFilters}>
      {viewMode === 'table' ? (
        <>
          <Table
            items={leads}
            columns={columns}
            onEdit={handleEdit}
            onDelete={handleDelete}
            onRowClick={(lead) => handleEdit(lead)}
            loading={loading}
            emptyMessage="No leads found. Create your first lead to get started."
            selectable
            selectedIds={selectedIds}
            onSelectionChange={handleSelectionChange}
          />
        </>
      ) : (
        <KanbanBoard
          columns={kanbanColumns}
          renderCard={(lead) => (
            <div className="p-3 hover:shadow-md transition-shadow cursor-pointer" onClick={() => handleEdit(lead)}>
              <div className="flex items-start justify-between gap-2 mb-2">
                <div className="flex-1 min-w-0">
                  <button className="crm-board-open" onClick={e => { e.stopPropagation(); handleEdit(lead); }}>{lead.display_name}</button>
                  {lead.company_name && <p className="text-sm text-tertiary truncate">{lead.company_name}</p>}
                </div>
                <Badge variant={getStatusBadge(lead.priority)} size="sm">{lead.priority}</Badge>
              </div>
              <div className="flex items-center gap-2 text-xs text-secondary">
                <span className="truncate">{lead.email}</span>
                {lead.estimated_value && (
                  <span className="font-medium text-black">{crmMoney(lead.estimated_value, lead.currency)}</span>
                )}
              </div>
              {lead.follow_up_date && (
                <div className="mt-2 text-xs text-warning flex items-center gap-1">
                  <Calendar className="w-3 h-3" />
                  Follow up: {workDate(lead.follow_up_date)}
                </div>
              )}
            </div>
          )}
          onDragEnd={async (item, _sourceColumn, targetColumn) => {
            try {
              await api.leads.update(item.id, { status: targetColumn as Lead['status'] });
              showToast('success', `Lead moved to ${targetColumn}`);
              fetchLeads();
            } catch {
              showToast('error', 'Failed to update lead status');
            }
          }}
          onAddCard={(columnId) => {
            handleCreate();
            setFormData(prev => ({ ...prev, status: columnId as Lead['status'] }));
          }}
        />
      )}

      </CrmState>
      {!loadError && <CrmPagination page={page} pageSize={pageSize} total={total} onPage={setPage} onSize={size => { setPageSize(size); setPage(1); }} busy={loading || submitLoading} />}

      <Modal busy={submitLoading} isOpen={modalOpen} onClose={() => setModalOpen(false)} title={editingLead ? 'Edit Lead' : 'New Lead'} size="lg">
        <div className="space-y-4 max-h-[70vh] overflow-y-auto">
          <FormField label="Display Name" error={formErrors.display_name} required>
            <Input value={formData.display_name} onChange={(e) => setFormData({ ...formData, display_name: e.target.value })} placeholder="Lead display name" />
          </FormField>
          <div className="form-row-2">
            <FormField label="First Name" error={formErrors.first_name} required>
              <Input value={formData.first_name} onChange={(e) => setFormData({ ...formData, first_name: e.target.value })} placeholder="First name" />
            </FormField>
            <FormField label="Last Name" error={formErrors.last_name} required>
              <Input value={formData.last_name} onChange={(e) => setFormData({ ...formData, last_name: e.target.value })} placeholder="Last name" />
            </FormField>
          </div>
          <FormField label="Email" error={formErrors.email} required>
            <Input type="email" value={formData.email} onChange={(e) => setFormData({ ...formData, email: e.target.value })} placeholder="lead@example.com" />
          </FormField>
          <FormField label="Phone">
            <Input value={formData.phone} onChange={(e) => setFormData({ ...formData, phone: e.target.value })} placeholder="+1 (555) 123-4567" />
          </FormField>
          <FormField label="Company Name">
            <Input value={formData.company_name} onChange={(e) => setFormData({ ...formData, company_name: e.target.value })} placeholder="Company name" />
          </FormField>
          <FormField label="Job Title">
            <Input value={formData.job_title} onChange={(e) => setFormData({ ...formData, job_title: e.target.value })} placeholder="Job title" />
          </FormField>
          <div className="form-row-2">
            <FormField label="Status">
              <Select value={formData.status} onChange={(e) => setFormData({ ...formData, status: e.target.value })}>
                {statusOptions.map((opt) => <option key={opt.value} value={opt.value}>{opt.label}</option>)}
              </Select>
            </FormField>
            <FormField label="Source">
              <Select value={formData.source} onChange={(e) => setFormData({ ...formData, source: e.target.value })}>
                {sourceOptions.map((opt) => <option key={opt.value} value={opt.value}>{opt.label}</option>)}
              </Select>
            </FormField>
          </div>
          <div className="form-row-2">
            <FormField label="Priority">
              <Select value={formData.priority} onChange={(e) => setFormData({ ...formData, priority: e.target.value })}>
                {priorityOptions.map((opt) => <option key={opt.value} value={opt.value}>{opt.label}</option>)}
              </Select>
            </FormField>
            <FormField label="Estimated Value">
              <Input type="number" value={String(formData.estimated_value)} onChange={(e) => setFormData({ ...formData, estimated_value: Number(e.target.value) || 0 })} placeholder="0" />
            </FormField>
          </div>
          <div className="form-row-2">
            <FormField label="Currency">
              <Select value={formData.currency} onChange={(e) => setFormData({ ...formData, currency: e.target.value })}>
                <option value="USD">USD</option>
                <option value="EUR">EUR</option>
                <option value="GBP">GBP</option>
              </Select>
            </FormField>
            <FormField label="Lead Score">
              <Input type="number" value={String(formData.lead_score)} onChange={(e) => setFormData({ ...formData, lead_score: Number(e.target.value) || 0 })} placeholder="0" />
            </FormField>
          </div>
          <div className="form-row-2">
            <FormField label="Expected Close Date">
              <Input type="date" value={formData.expected_close} onChange={(e) => setFormData({ ...formData, expected_close: e.target.value })} />
            </FormField>
            <FormField label="Follow Up Date">
              <Input type="date" value={formData.follow_up_date} onChange={(e) => setFormData({ ...formData, follow_up_date: e.target.value })} />
            </FormField>
          </div>
          <FormField label="Description">
            <Textarea value={formData.description} onChange={(e) => setFormData({ ...formData, description: e.target.value })} placeholder="Lead description" rows={3} />
          </FormField>
          <FormField label="Qualification Notes">
            <Textarea value={formData.qualification} onChange={(e) => setFormData({ ...formData, qualification: e.target.value })} placeholder="Qualification details" rows={2} />
          </FormField>
        </div>
        <div className="modal-footer">
          {editingLead && !['won', 'lost'].includes(editingLead.status) && <Button variant="secondary" onClick={() => handleConvert(editingLead)} disabled={submitLoading || !pipelines.length}>Convert lead</Button>}
          <Button variant="secondary" onClick={() => setModalOpen(false)} disabled={submitLoading}>Cancel</Button>
          <Button variant="primary" onClick={handleSubmit} disabled={submitLoading} loading={submitLoading}>
            {editingLead ? 'Update' : 'Create'}
          </Button>
        </div>
      </Modal>

      <Modal busy={submitLoading} isOpen={convertModalOpen} onClose={() => setConvertModalOpen(false)} title="Convert Lead" size="md">
        <div className="space-y-4">
          {conversionError && <p className="form-error" role="alert">{conversionError}</p>}
          <p className="text-secondary">Create a contact, company, and deal from this lead.</p>
          <FormField label="Deal Name" required>
            <Input value={convertData.deal_name} onChange={(e) => setConvertData({ ...convertData, deal_name: e.target.value })} placeholder="Deal name" />
          </FormField>
          <FormField label="Pipeline" required>
            <Select value={convertData.pipeline_id} onChange={e => setConvertData({ ...convertData, pipeline_id: e.target.value, stage_id: pipelines.find(item => item.id === e.target.value)?.stages[0]?.id || '' })}><option value="">Choose pipeline</option>{pipelines.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</Select>
          </FormField>
          <FormField label="Stage" required>
            <Select value={convertData.stage_id} onChange={e => setConvertData({ ...convertData, stage_id: e.target.value })}><option value="">Choose stage</option>{pipelines.find(item => item.id === convertData.pipeline_id)?.stages.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</Select>
          </FormField>
          <FormField label={`Amount (${convertData.currency})`} required>
            <Input type="number" step="0.01" value={String(convertData.amount)} onChange={(e) => setConvertData({ ...convertData, amount: Number(e.target.value) || 0 })} placeholder="0" />
          </FormField>
          <FormField label="Expected Close Date" required>
            <Input type="date" value={convertData.expected_close} onChange={(e) => setConvertData({ ...convertData, expected_close: e.target.value })} />
          </FormField>
        </div>
        <div className="modal-footer">
          <Button variant="secondary" onClick={() => setConvertModalOpen(false)} disabled={submitLoading}>Cancel</Button>
          <Button variant="primary" onClick={handleConvertSubmit} disabled={submitLoading} loading={submitLoading}>Convert</Button>
        </div>
      </Modal>

      <ConfirmDialog
        isOpen={!!deleteConfirm}
        onClose={() => setDeleteConfirm(null)}
        onConfirm={confirmDelete}
        title="Archive Lead"
        message={`Are you sure you want to archive "${deleteConfirm?.display_name}"? The record will be removed from active views.`}
        confirmText="Archive"
        variant="danger"
        loading={submitLoading}
      />

      <ToastContainer toasts={toasts} onRemove={(id) => setToasts((prev) => prev.filter((t) => t.id !== id))} />
    </div>
  );
}
