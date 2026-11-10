import { crmMoney, crmValueSummary, exportCrmPage } from './crm-data';
import { CrmHeader, CrmSummary, CrmState, CrmPagination, CrmDialog as Modal } from './CrmUI';
import { workBatch, workDate } from './work-data';
import { useSearchParams } from 'react-router-dom';
import { useRecordRoute } from './useRecordRoute';
import { useState, useEffect, useCallback, useRef } from 'react';
import { Table as TableIcon, Kanban as KanbanIcon, Calendar } from 'lucide-react';
import { api, Deal, Company } from './api';
import type { Contact, PipelineChoice } from './api';
import { Table, FormField, Input, Textarea, Select, Badge, ToastContainer, Toast, ConfirmDialog, getStatusBadge, Button, DataTableToolbar, Avatar, KanbanBoard } from './components';

const DEFAULT_PAGE_SIZE = 25;

const currencyOptions = [
  { value: 'USD', label: 'USD' },
  { value: 'EUR', label: 'EUR' },
  { value: 'GBP', label: 'GBP' },
];

export function DealsPage() {
  const [routeParams] = useSearchParams();
  const [deals, setDeals] = useState<Deal[]>([]);
  const [pipelines, setPipelines] = useState<PipelineChoice[]>([]);
  const [pipelineError, setPipelineError] = useState('');
  const pipelineStages = pipelines.flatMap(pipeline => pipeline.stages.map(stage => ({ ...stage, pipeline_id: pipeline.id, color: stage.category === 'won' ? '#4d805a' : stage.category === 'lost' ? '#ae6355' : '#dd6b35' })));
  const [companies, setCompanies] = useState<Company[]>([]);
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [query, setQuery] = useState('');
  const [stageFilter, setStageFilter] = useState(routeParams.get('stage_id')??'');
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const requestSequence = useRef(0);
  const [toasts, setToasts] = useState<Toast[]>([]);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingDeal, setEditingDeal] = useState<Deal | null>(null);
  interface DealFormData {
  name: string;
  pipeline_id: string;
  stage_id: string;
  amount: number | string;
  currency: string;
  probability: number | string;
  expected_close: string;
  company_id: string;
  primary_contact_id: string;
  lead_id: string;
  owner_id: string;
  source: string;
  description: string;
  lost_reason: string;
  custom_fields: Record<string, any>;
}

const [formData, setFormData] = useState<DealFormData>({
    name: '', pipeline_id: (pipelines.find(pipeline => pipeline.is_default) || pipelines[0])?.id || '', stage_id: (pipelines.find(pipeline => pipeline.is_default) || pipelines[0])?.stages[0]?.id || '', amount: 0, currency: 'USD',
    probability: 50, expected_close: '', company_id: '', primary_contact_id: '',
    lead_id: '', owner_id: '', source: '', description: '', lost_reason: '',
    custom_fields: {},
  });
  const [formErrors, setFormErrors] = useState<Record<string, string>>({});
  const [submitLoading, setSubmitLoading] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<Deal | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [viewMode, setViewMode] = useState<'table' | 'kanban'>('kanban');

  useEffect(() => { setSelectedIds(new Set()); }, [query, stageFilter, page, pageSize]);

  const showToast = useCallback((type: Toast['type'], message: string) => {
    const id = Date.now() + Math.random();
    setToasts((prev) => [...prev, { id, type, message }]);
    setTimeout(() => setToasts((prev) => prev.filter((t) => t.id !== id)), 5000);
  }, []);

  const fetchDeals = useCallback(async () => {
    const sequence = ++requestSequence.current;
    setLoading(true);
    setLoadError('');
    try {
      const response = await api.deals.list({
        q: query,
        stage_id: stageFilter || undefined,
        limit: pageSize,
        offset: (page - 1) * pageSize,
      });
      if (sequence !== requestSequence.current) return;
      if (page > Math.max(1, Math.ceil(response.total / pageSize))) { setPage(Math.max(1, Math.ceil(response.total / pageSize))); return; }
      setDeals(response.items);
      setTotal(response.total);
    } catch {
      if (sequence !== requestSequence.current) return;
      setLoadError('Failed to load deals');
      setDeals([]);
    } finally {
      if (sequence === requestSequence.current) setLoading(false);
    }
  }, [query, stageFilter, page, pageSize]);

  const fetchCompanies = useCallback(async () => {
    try {
      const response = await api.companies.list({ limit: 1000, status: 'active' });
      setCompanies(response.items);
    } catch (error) {
      console.error('Failed to fetch companies:', error);
    }
  }, []);

  const fetchContacts = useCallback(async () => {
    try {
      const response = await api.contacts.list({ limit: 1000, status: 'active' });
      setContacts(response.items);
    } catch (error) {
      console.error('Failed to fetch contacts:', error);
    }
  }, []);

  useEffect(() => {
    fetchDeals();
    fetchCompanies();
    fetchContacts();
  }, [fetchDeals, fetchCompanies, fetchContacts]);

  useEffect(() => {
    let active = true;
    api.get<{ items: PipelineChoice[] }>('/pipelines').then(data => { if (active) setPipelines(data.items ?? []); })
      .catch(() => { if (active) setPipelineError('Could not load pipelines. Refresh this page to try again.'); });
    return () => { active = false; };
  }, []);

  const handleCreate = () => {
    setEditingDeal(null);
    setFormData({
      name: '', pipeline_id: '', stage_id: '', amount: 0, currency: 'USD',
      probability: 50, expected_close: '', company_id: '', primary_contact_id: '',
      lead_id: '', owner_id: '', source: '', description: '', lost_reason: '',
      custom_fields: {},
    });
    setFormErrors({});
    setModalOpen(true);
  };

  const handleEdit = (deal: Deal) => {
    setEditingDeal(deal);
    setFormData({
      name: deal.name, pipeline_id: deal.pipeline_id || '', stage_id: deal.stage_id || '',
      amount: deal.amount || 0, currency: deal.currency,
      probability: deal.probability ?? 50, expected_close: deal.expected_close ? deal.expected_close.split('T')[0] : '',
      company_id: deal.company_id || '', primary_contact_id: deal.primary_contact_id || '',
      lead_id: deal.lead_id || '', owner_id: deal.owner_id || '',
      source: deal.source || '', description: deal.description || '',
      lost_reason: deal.lost_reason || '', custom_fields: deal.custom_fields || {},
    });
    setFormErrors({});
    setModalOpen(true);
  };

  const handleDelete = (deal: Deal) => setDeleteConfirm(deal);

  const handleBulkArchive = async () => {
    if (submitLoading) return;
    setSubmitLoading(true);
    try {
      const result = await workBatch([...selectedIds], api.deals.archive);
      setSelectedIds(new Set(result.failed));
      showToast(result.failed.length ? 'error' : 'success', result.succeeded.length + ' archived' + (result.failed.length ? '; ' + result.failed.length + ' failed. Retry the selected records.' : ' successfully'));
      await fetchDeals();
    } finally { setSubmitLoading(false); }
  };
  const confirmDelete = async () => {
    if (!deleteConfirm) return;
    setSubmitLoading(true);
    try {
      await api.deals.archive(deleteConfirm.id);
      showToast('success', 'Deal archived successfully');
      fetchDeals();
    } catch {
      showToast('error', 'Failed to archive deal');
    } finally {
      setSubmitLoading(false);
      setDeleteConfirm(null);
    }
  };

  const validateForm = () => {
    const errors: Record<string, string> = {};
    if (!formData.name.trim()) errors.name = 'Deal name is required';
    if (!formData.pipeline_id.trim()) errors.pipeline_id = 'Pipeline ID is required';
    if (!formData.stage_id.trim()) errors.stage_id = 'Stage ID is required';
    if (!formData.expected_close.trim()) errors.expected_close = 'Expected close date is required';
    const amount = Number(formData.amount ?? 0);
    const probability = Number(formData.probability ?? 50);
    if (!Number.isFinite(amount) || amount < 0) errors.amount = 'Enter a valid, nonnegative amount';
    if (!Number.isFinite(probability) || probability < 0 || probability > 100) errors.probability = 'Probability must be between 0 and 100';
    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async () => {
    if (!validateForm()) return;
    setSubmitLoading(true);
    try {
      const dealData = {
        name: formData.name, pipeline_id: formData.pipeline_id, stage_id: formData.stage_id,
        amount: Number(formData.amount ?? 0), currency: formData.currency, probability: Number(formData.probability ?? 50),
        expected_close: formData.expected_close, company_id: formData.company_id || undefined,
        primary_contact_id: formData.primary_contact_id || undefined, lead_id: formData.lead_id || undefined,
        owner_id: formData.owner_id || undefined, source: formData.source || undefined,
        description: formData.description || undefined, lost_reason: formData.lost_reason || undefined,
        custom_fields: formData.custom_fields || {},
      };
      if (editingDeal) {
        await api.deals.update(editingDeal.id, dealData);
        showToast('success', 'Deal updated successfully');
      } else {
        await api.deals.create(dealData);
        showToast('success', 'Deal created successfully');
      }
      setModalOpen(false);
      fetchDeals();
    } catch (error: any) {
      if (error.status === 400) showToast('error', error.message);
      else showToast('error', 'Failed to save deal');
    } finally {
      setSubmitLoading(false);
    }
  };

  const columns = [
    { key: 'name', header: 'Deal Name', render: (d: Deal) => (
      <div className="flex items-center gap-3">
        <Avatar name={d.name} size="sm" />
        <button className="crm-table-name" onClick={e => { e.stopPropagation(); handleEdit(d); }}>{d.name}</button>
      </div>
    )},
    { key: 'company_id', header: 'Company', render: (d: Deal) => {
      const company = companies.find(c => c.id === d.company_id);
      return company ? <span className="font-medium">{company.name}</span> : <span className="text-tertiary">—</span>;
    }},
    { key: 'primary_contact_id', header: 'Contact', render: (d: Deal) => {
      const contact = contacts.find(c => c.id === d.primary_contact_id);
      return contact ? <span>{contact.first_name} {contact.last_name}</span> : <span className="text-tertiary">—</span>;
    }},
    { key: 'amount', header: 'Amount', render: (d: Deal) => d.amount !== null ? (
      <span className="font-semibold">{crmMoney(d.amount, d.currency)}</span>
    ) : <span className="text-tertiary">—</span> },
    { key: 'probability', header: 'Probability', render: (d: Deal) => (
      <div className="flex items-center gap-2">
        <div className="h-2 w-20 bg-border-primary rounded-full overflow-hidden">
          <div className="h-full bg-black rounded-full" style={{ width: `${d.probability}%` }} />
        </div>
        <span className="text-sm font-medium w-12">{d.probability}%</span>
      </div>
    )},
    { key: 'expected_close', header: 'Close Date', render: (d: Deal) => d.expected_close ? workDate(d.expected_close) : '—' },
    { key: 'stage_id', header: 'Stage', render: (d: Deal) => {
      const stage = pipelineStages.find(s => s.id === d.stage_id);
      return stage ? (
        <Badge variant={getStatusBadge(d.stage_id)} size="sm" dot>{stage.name}</Badge>
      ) : <Badge variant="neutral" size="sm">{d.stage_id}</Badge>;
    }},
  ];

  const filters = [
    { label: 'Stage', value: 'stage', options: [{ value: '', label: 'All' }, ...pipelineStages.map(s => ({ value: s.id, label: pipelines.length > 1 ? `${pipelines.find(p => p.id === s.pipeline_id)?.name} / ${s.name}` : s.name }))] },
  ];

  const filterValues = { stage: stageFilter };
  const hasActiveFilters = !!stageFilter;

  const handleFilterChange = (key: string, value: string) => {
    if (key === 'stage') setStageFilter(value);
    setPage(1);
  };

  const clearFilters = () => { setStageFilter(''); setPage(1); };

  const handleSelectionChange = (ids: Set<string>) => setSelectedIds(ids);

  const kanbanColumns = pipelineStages.map(stage => ({
    id: stage.id,
    title: pipelines.length > 1 ? `${pipelines.find(p => p.id === stage.pipeline_id)?.name} / ${stage.name}` : stage.name,
    color: stage.color,
    items: deals.filter(d => d.stage_id === stage.id),
  }));

  const categoryFor = (deal: Deal) => pipelineStages.find(stage => stage.id === deal.stage_id)?.category;

  useRecordRoute({ getRecord: api.deals.get, onCreate: handleCreate, onRecord: handleEdit, onError: message => showToast('error', message) });

  return (
    <div className="crm-area">
      <CrmHeader title="Deals" description="A clearer view of every opportunity in motion." onCreate={handleCreate} busy={loading || !!loadError || !deals.length} onExport={() => exportCrmPage('deals', deals, [{ key: 'name', label: 'name' }, { key: 'amount', label: 'amount' }, { key: 'currency', label: 'currency' }, { key: 'probability', label: 'probability' }, { key: 'stage_id', label: 'stage id' }, { key: 'expected_close', label: 'expected close' }])} />
      <CrmSummary loading={loading || !!loadError || !!pipelineError} items={[{ label: 'Matching deals', value: total, note: 'With current filters' }, { label: 'Open pipeline', value: crmValueSummary(deals.filter(d => categoryFor(d) === 'open')), note: 'On this page, grouped by currency' }, { label: 'Weighted pipeline', value: crmValueSummary(deals.filter(d => categoryFor(d) === 'open').map(d => ({ ...d, amount: (d.amount || 0) * d.probability / 100 }))), note: 'Amount × probability, on this page' }]} />
      {pipelineError && <p className="crm-notice" role="alert">{pipelineError} The table view is still available.</p>}

      <DataTableToolbar
        title="Pipeline"
        description={`${deals.length} loaded on this page`}
        searchProps={{ query, onChange: (value: string) => { setQuery(value); setPage(1); }, placeholder: 'Search deals by name...' }}
        filterProps={{ filters, values: filterValues, onChange: handleFilterChange, onClear: clearFilters, hasActiveFilters }}
        viewOptions={{
          currentView: viewMode,
          views: [
            { id: 'kanban', label: 'Kanban', icon: <KanbanIcon className="w-4 h-4" /> },
            { id: 'table', label: 'Table', icon: <TableIcon className="w-4 h-4" /> },
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

      {viewMode === 'kanban' && !loading && !loadError && <p className="crm-board-note">Board shows the deals on this page. Use the page controls to see more. Drag a card or use its Move to menu.</p>}
      <CrmState loading={loading} error={loadError} empty={!deals.length} onRetry={fetchDeals} onCreate={handleCreate} noun="deals" filtered={!!query || hasActiveFilters}>
      {viewMode === 'kanban' ? (
        <KanbanBoard
          columns={kanbanColumns}
          renderCard={(deal) => {
            const company = companies.find(c => c.id === deal.company_id);
            const contact = contacts.find(c => c.id === deal.primary_contact_id);
            return (
              <div className="p-3 hover:shadow-md transition-shadow cursor-pointer" onClick={() => handleEdit(deal)}>
                <div className="flex items-start justify-between gap-2 mb-2">
                  <div className="flex-1 min-w-0">
                    <button className="crm-board-open" onClick={e => { e.stopPropagation(); handleEdit(deal); }}>{deal.name}</button>
                    {company && <p className="text-sm text-tertiary truncate">{company.name}</p>}
                  </div>
                  <Badge variant="warning" size="sm">
                    {crmMoney(deal.amount || 0, deal.currency)}
                  </Badge>
                </div>
                <div className="flex items-center gap-2 text-xs text-secondary">
                  {contact && <span className="truncate">{contact.first_name} {contact.last_name}</span>}
                  <div className="h-1.5 w-16 bg-border-primary rounded-full overflow-hidden">
                    <div className="h-full bg-black rounded-full" style={{ width: `${deal.probability}%` }} />
                  </div>
                  <span className="font-medium text-black w-10">{deal.probability}%</span>
                </div>
                {deal.expected_close && (
                  <div className="mt-2 text-xs text-warning flex items-center gap-1">
                    <Calendar className="w-3 h-3" />
                    Close: {workDate(deal.expected_close)}
                  </div>
                )}
              </div>
            );
          }}
          onDragEnd={async (item, _sourceColumn, targetColumn) => {
            try {
              await api.deals.update(item.id, { stage_id: targetColumn, pipeline_id: pipelineStages.find(stage => stage.id === targetColumn)?.pipeline_id });
              showToast('success', `Deal moved to ${pipelineStages.find(s => s.id === targetColumn)?.name}`);
              fetchDeals();
            } catch {
              showToast('error', 'Failed to update deal stage');
            }
          }}
          onAddCard={(columnId) => {
            handleCreate();
            setFormData(prev => ({ ...prev, stage_id: columnId, pipeline_id: pipelineStages.find(stage => stage.id === columnId)?.pipeline_id || '' }));
          }}
        />
      ) : (
        <>
          <Table
            items={deals}
            columns={columns}
            onEdit={handleEdit}
            onDelete={handleDelete}
            onRowClick={(deal) => handleEdit(deal)}
            loading={loading}
            emptyMessage="No deals found. Create your first deal to get started."
            selectable
            selectedIds={selectedIds}
            onSelectionChange={handleSelectionChange}
          />
        </>
      )}

      </CrmState>
      {!loadError && <CrmPagination page={page} pageSize={pageSize} total={total} onPage={setPage} onSize={size => { setPageSize(size); setPage(1); }} busy={loading || submitLoading} />}

      <Modal busy={submitLoading} isOpen={modalOpen} onClose={() => setModalOpen(false)} title={editingDeal ? 'Edit Deal' : 'New Deal'} size="lg">
        <div className="space-y-4 max-h-[70vh] overflow-y-auto">{pipelineError && <p className="form-error" role="alert">{pipelineError}</p>}
          <FormField label="Deal Name" error={formErrors.name} required>
            <Input value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} placeholder="Deal name" />
          </FormField>
          <div className="form-row-2">
            <FormField label="Pipeline" error={formErrors.pipeline_id} required>
              <Select value={formData.pipeline_id} onChange={event => setFormData({ ...formData, pipeline_id: event.target.value, stage_id: pipelines.find(pipeline => pipeline.id === event.target.value)?.stages[0]?.id || '' })}><option value="">Select pipeline</option>{pipelines.map(pipeline => <option key={pipeline.id} value={pipeline.id}>{pipeline.name}</option>)}</Select>
            </FormField>
            <FormField label="Stage" error={formErrors.stage_id} required>
              <Select value={formData.stage_id} onChange={(e) => setFormData({ ...formData, stage_id: e.target.value })}>
                <option value="">Select stage</option>{pipelineStages.filter(stage => stage.pipeline_id === formData.pipeline_id).map(opt => <option key={opt.id} value={opt.id}>{opt.name}</option>)}
              </Select>
            </FormField>
          </div>
          <div className="form-row-3">
            <FormField label="Amount" error={formErrors.amount} required>
              <Input type="number" step="0.01" value={String(formData.amount ?? 0)} onChange={(e) => setFormData({ ...formData, amount: e.target.value })} placeholder="0" />
            </FormField>
            <FormField label="Currency">
              <Select value={formData.currency} onChange={(e) => setFormData({ ...formData, currency: e.target.value })}>
                {currencyOptions.map((opt) => <option key={opt.value} value={opt.value}>{opt.label}</option>)}
              </Select>
            </FormField>
            <FormField label="Probability (%)" error={formErrors.probability} required>
              <Input type="number" min="0" max="100" value={String(formData.probability ?? 50)} onChange={(e) => setFormData({ ...formData, probability: e.target.value })} placeholder="50" />
            </FormField>
          </div>
          <FormField label="Expected Close Date" error={formErrors.expected_close} required>
            <Input type="date" value={formData.expected_close} onChange={(e) => setFormData({ ...formData, expected_close: e.target.value })} />
          </FormField>
          <div className="form-row-2">
            <FormField label="Company">
              <Select value={formData.company_id} onChange={(e) => setFormData({ ...formData, company_id: e.target.value })}>
                <option value="">Select company (optional)</option>
                {companies.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
              </Select>
            </FormField>
            <FormField label="Primary Contact">
              <Select value={formData.primary_contact_id} onChange={(e) => setFormData({ ...formData, primary_contact_id: e.target.value })}>
                <option value="">Select contact (optional)</option>
                {contacts.map((c) => <option key={c.id} value={c.id}>{c.first_name} {c.last_name} ({c.email})</option>)}
              </Select>
            </FormField>
          </div>
          <FormField label="Source">
            <Input value={formData.source} onChange={(e) => setFormData({ ...formData, source: e.target.value })} placeholder="Lead source" />
          </FormField>
          <FormField label="Description">
            <Textarea value={formData.description} onChange={(e) => setFormData({ ...formData, description: e.target.value })} placeholder="Deal description" rows={3} />
          </FormField>
          <FormField label="Lost Reason">
            <Textarea value={formData.lost_reason} onChange={(e) => setFormData({ ...formData, lost_reason: e.target.value })} placeholder="Reason if lost" rows={2} />
          </FormField>
        </div>
        <div className="modal-footer">
          <Button variant="secondary" onClick={() => setModalOpen(false)} disabled={submitLoading}>Cancel</Button>
          <Button variant="primary" onClick={handleSubmit} disabled={submitLoading} loading={submitLoading}>
            {editingDeal ? 'Update' : 'Create'}
          </Button>
        </div>
      </Modal>

      <ConfirmDialog
        isOpen={!!deleteConfirm}
        onClose={() => setDeleteConfirm(null)}
        onConfirm={confirmDelete}
        title="Archive Deal"
        message={`Are you sure you want to archive "${deleteConfirm?.name}"? The record will be removed from active views.`}
        confirmText="Archive"
        variant="danger"
        loading={submitLoading}
      />

      <ToastContainer toasts={toasts} onRemove={(id) => setToasts((prev) => prev.filter((t) => t.id !== id))} />
    </div>
  );
}

