import { exportCrmPage } from './crm-data';
import { CrmCompanyCards, CrmHeader, CrmSummary, CrmState, CrmPagination, CrmDialog as Modal } from './CrmUI';
import { workBatch, workDate } from './work-data';
import { useRecordRoute } from './useRecordRoute';
import { useState, useEffect, useCallback, useRef } from 'react';
import { Grid, Table as TableIcon } from 'lucide-react';
import { api, Company, CompanyCreateInput } from './api';
import { Table, FormField, Input, Textarea, Select, Badge, ToastContainer, Toast, ConfirmDialog, getStatusBadge, Button, DataTableToolbar, Avatar } from './components';

const DEFAULT_PAGE_SIZE = 25;

const sizeBandOptions = [
  { value: '', label: 'Not specified' },
  { value: '1-10', label: '1-10' },
  { value: '11-50', label: '11-50' },
  { value: '51-200', label: '51-200' },
  { value: '201-500', label: '201-500' },
  { value: '501-1000', label: '501-1000' },
  { value: '1001-5000', label: '1001-5000' },
  { value: '5001+', label: '5001+' },
];

const industryOptions = [
  { value: '', label: 'Not specified' },
  { value: 'Technology', label: 'Technology' },
  { value: 'Healthcare', label: 'Healthcare' },
  { value: 'Finance', label: 'Finance' },
  { value: 'Manufacturing', label: 'Manufacturing' },
  { value: 'Retail', label: 'Retail' },
  { value: 'Education', label: 'Education' },
  { value: 'Real Estate', label: 'Real Estate' },
  { value: 'Professional Services', label: 'Professional Services' },
  { value: 'Other', label: 'Other' },
];

const statusOptions = [
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
];

export function CompaniesPage() {
  const [companies, setCompanies] = useState<Company[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [query, setQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [isClientFilter, setIsClientFilter] = useState('');
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const requestSequence = useRef(0);
  const [toasts, setToasts] = useState<Toast[]>([]);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingCompany, setEditingCompany] = useState<Company | null>(null);
  const [formData, setFormData] = useState<CompanyCreateInput>({
    name: '', domain: '', industry: '', size_band: '', website: '',
    phone: '', email: '', address: '', is_client: false,
    status: 'active', owner_id: '', tags: [], custom_fields: {}, notes: '',
  });
  const [formErrors, setFormErrors] = useState<Partial<CompanyCreateInput>>({});
  const [submitLoading, setSubmitLoading] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<Company | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [viewMode, setViewMode] = useState<'table' | 'cards'>('table');

  useEffect(() => { setSelectedIds(new Set()); }, [query, statusFilter, isClientFilter, page, pageSize]);

  const showToast = useCallback((type: Toast['type'], message: string) => {
    const id = Date.now() + Math.random();
    setToasts((prev) => [...prev, { id, type, message }]);
    setTimeout(() => setToasts((prev) => prev.filter((t) => t.id !== id)), 5000);
  }, []);

  const fetchCompanies = useCallback(async () => {
    const sequence = ++requestSequence.current;
    setLoading(true);
    setLoadError('');
    try {
      const response = await api.companies.list({
        q: query,
        status: statusFilter || undefined,
        is_client: isClientFilter ? isClientFilter === 'true' : undefined,
        limit: pageSize,
        offset: (page - 1) * pageSize,
        sort: 'created_at',
        order: 'desc',
      });
      if (sequence !== requestSequence.current) return;
      if (page > Math.max(1, Math.ceil(response.total / pageSize))) { setPage(Math.max(1, Math.ceil(response.total / pageSize))); return; }
      setCompanies(response.items);
      setTotal(response.total);
    } catch {
      if (sequence !== requestSequence.current) return;
      setLoadError('Failed to load companies');
      setCompanies([]);
    } finally {
      if (sequence === requestSequence.current) setLoading(false);
    }
  }, [query, statusFilter, isClientFilter, page, pageSize]);

  useEffect(() => { fetchCompanies(); }, [fetchCompanies]);

  const handleCreate = () => {
    setEditingCompany(null);
    setFormData({
      name: '', domain: '', industry: '', size_band: '', website: '',
      phone: '', email: '', address: '', is_client: false,
      status: 'active', owner_id: '', tags: [], custom_fields: {}, notes: '',
    });
    setFormErrors({});
    setModalOpen(true);
  };

  const handleEdit = (company: Company) => {
    setEditingCompany(company);
    setFormData({
      name: company.name, domain: company.domain, industry: company.industry,
      size_band: company.size_band || '', website: company.website || '',
      phone: company.phone || '', email: company.email || '',
      address: company.address || '', is_client: company.is_client,
      status: company.status, owner_id: company.owner_id || '',
      tags: company.tags || [], custom_fields: company.custom_fields || {},
      notes: company.notes || '',
    });
    setFormErrors({});
    setModalOpen(true);
  };

  const handleDelete = (company: Company) => setDeleteConfirm(company);

  const handleBulkArchive = async () => {
    if (submitLoading) return;
    setSubmitLoading(true);
    try {
      const result = await workBatch([...selectedIds], api.companies.archive);
      setSelectedIds(new Set(result.failed));
      showToast(result.failed.length ? 'error' : 'success', result.succeeded.length + ' archived' + (result.failed.length ? '; ' + result.failed.length + ' failed. Retry the selected records.' : ' successfully'));
      await fetchCompanies();
    } finally { setSubmitLoading(false); }
  };
  const confirmDelete = async () => {
    if (!deleteConfirm) return;
    setSubmitLoading(true);
    try {
      await api.companies.archive(deleteConfirm.id);
      showToast('success', 'Company archived successfully');
      fetchCompanies();
    } catch {
      showToast('error', 'Failed to archive company');
    } finally {
      setSubmitLoading(false);
      setDeleteConfirm(null);
    }
  };

  const validateForm = () => {
    const errors: Partial<CompanyCreateInput> = {};
    if (!formData.name.trim()) errors.name = 'Name is required';
    if (!formData.domain.trim()) errors.domain = 'Domain is required';
    if (!formData.industry.trim()) errors.industry = 'Industry is required';
    setFormErrors(errors);
    return Object.keys(errors).length === 0;
  };

  const handleSubmit = async () => {
    if (!validateForm()) return;
    setSubmitLoading(true);
    try {
      if (editingCompany) {
        await api.companies.update(editingCompany.id, formData);
        showToast('success', 'Company updated successfully');
      } else {
        await api.companies.create(formData);
        showToast('success', 'Company created successfully');
      }
      setModalOpen(false);
      fetchCompanies();
    } catch (error: any) {
      if (error.status === 400) showToast('error', error.message);
      else showToast('error', 'Failed to save company');
    } finally {
      setSubmitLoading(false);
    }
  };

  const columns = [
    { key: 'name', header: 'Name', render: (c: Company) => (
      <div className="flex items-center gap-3">
        <Avatar name={c.name} size="sm" />
        <div>
          <button className="crm-table-name" onClick={e => { e.stopPropagation(); handleEdit(c); }}>{c.name}</button>
          {c.domain && <p className="text-xs text-tertiary">{c.domain}</p>}
        </div>
      </div>
    )},
    { key: 'domain', header: 'Domain' },
    { key: 'industry', header: 'Industry' },
    { key: 'size_band', header: 'Size', render: (c: Company) => c.size_band || '—' },
    { key: 'is_client', header: 'Client', render: (c: Company) => <Badge variant={c.is_client ? 'success' : 'neutral'} size="sm" dot>{c.is_client ? 'Yes' : 'No'}</Badge> },
    { key: 'status', header: 'Status', render: (c: Company) => <Badge variant={getStatusBadge(c.status)} size="sm" dot>{c.status}</Badge> },
    { key: 'created_at', header: 'Created', render: (c: Company) => workDate(c.created_at) },
  ];

  const filters = [
    { label: 'Status', value: 'status', options: statusOptions },
    { label: 'Client', value: 'is_client', options: [{ value: '', label: 'All' }, { value: 'true', label: 'Yes' }, { value: 'false', label: 'No' }] },
  ];

  const filterValues = { status: statusFilter, is_client: isClientFilter };
  const hasActiveFilters = !!statusFilter || !!isClientFilter;

  const handleFilterChange = (key: string, value: string) => {
    if (key === 'status') setStatusFilter(value);
    if (key === 'is_client') setIsClientFilter(value);
    setPage(1);
  };

  const clearFilters = () => {
    setStatusFilter(''); setIsClientFilter(''); setPage(1);
  };

  const handleSelectionChange = (ids: Set<string>) => setSelectedIds(ids);


  useRecordRoute({ getRecord: api.companies.get, onCreate: handleCreate, onRecord: handleEdit, onError: message => showToast('error', message) });

  return (
    <div className="crm-area">
      <CrmHeader title="Companies" description="Every account, with the context to grow it." onCreate={handleCreate} busy={loading || !!loadError || !companies.length} onExport={() => exportCrmPage('companies', companies, [{ key: 'name', label: 'name' }, { key: 'domain', label: 'domain' }, { key: 'industry', label: 'industry' }, { key: 'status', label: 'status' }, { key: 'is_client', label: 'is client' }, { key: 'email', label: 'email' }])} />
      <CrmSummary loading={loading || !!loadError} items={[{ label: 'Matching companies', value: total, note: 'With current filters' }, { label: 'Client accounts', value: companies.filter(c => c.is_client).length, note: 'On this page' }, { label: 'Active accounts', value: companies.filter(c => c.status === 'active').length, note: 'On this page' }]} />

      <DataTableToolbar
        title="All Companies"
        description={`${companies.length} loaded on this page`}
        searchProps={{ query, onChange: (value: string) => { setQuery(value); setPage(1); }, placeholder: 'Search companies by name, domain, industry...' }}
        filterProps={{ filters, values: filterValues, onChange: handleFilterChange, onClear: clearFilters, hasActiveFilters }}
        viewOptions={{
          currentView: viewMode,
          views: [
            { id: 'table', label: 'Table', icon: <TableIcon className="w-4 h-4" /> },
            { id: 'cards', label: 'Cards', icon: <Grid className="w-4 h-4" /> },
          ],
          onChange: (view: string) => setViewMode(view as 'table' | 'cards')
        }}
        bulkActions={selectedIds.size > 0 && !submitLoading ? {
          selectedCount: selectedIds.size,
          actions: [
            { label: 'Archive', onClick: handleBulkArchive, variant: 'danger' },
            { label: 'Clear selection', onClick: () => setSelectedIds(new Set()) },
          ]
        } : undefined}
      />

      <CrmState loading={loading} error={loadError} empty={!companies.length} onRetry={fetchCompanies} onCreate={handleCreate} noun="companies" filtered={!!query || hasActiveFilters}>
      {viewMode === 'table' ? (
        <>
          <Table
            items={companies}
            columns={columns}
            onEdit={handleEdit}
            onDelete={handleDelete}
            onRowClick={(company) => handleEdit(company)}
            loading={loading}
            emptyMessage="No companies found. Create your first company to get started."
            selectable
            selectedIds={selectedIds}
            onSelectionChange={handleSelectionChange}
          />
        </>
      ) : (
        <CrmCompanyCards items={companies} selected={selectedIds} onSelect={setSelectedIds} onEdit={handleEdit} onArchive={handleDelete} />
      )}

      </CrmState>
      {!loadError && <CrmPagination page={page} pageSize={pageSize} total={total} onPage={setPage} onSize={size => { setPageSize(size); setPage(1); }} busy={loading || submitLoading} />}

      <Modal busy={submitLoading} isOpen={modalOpen} onClose={() => setModalOpen(false)} title={editingCompany ? 'Edit Company' : 'New Company'} size="lg">
        <div className="space-y-4 max-h-[70vh] overflow-y-auto">
          <FormField label="Name" error={formErrors.name} required>
            <Input value={formData.name} onChange={(e) => setFormData({ ...formData, name: e.target.value })} placeholder="Company name" />
          </FormField>
          <FormField label="Domain" error={formErrors.domain} required>
            <Input value={formData.domain} onChange={(e) => setFormData({ ...formData, domain: e.target.value })} placeholder="example.com" />
          </FormField>
          <FormField label="Industry" error={formErrors.industry} required>
            <Select value={formData.industry} onChange={(e) => setFormData({ ...formData, industry: e.target.value })}>
              {industryOptions.map((opt) => <option key={opt.value} value={opt.value}>{opt.label}</option>)}
            </Select>
          </FormField>
          <FormField label="Size Band">
            <Select value={formData.size_band} onChange={(e) => setFormData({ ...formData, size_band: e.target.value })}>
              {sizeBandOptions.map((opt) => <option key={opt.value} value={opt.value}>{opt.label}</option>)}
            </Select>
          </FormField>
          <FormField label="Website">
            <Input type="url" value={formData.website} onChange={(e) => setFormData({ ...formData, website: e.target.value })} placeholder="https://example.com" />
          </FormField>
          <div className="form-row-2">
            <FormField label="Phone">
              <Input value={formData.phone} onChange={(e) => setFormData({ ...formData, phone: e.target.value })} placeholder="+1 (555) 123-4567" />
            </FormField>
            <FormField label="Email">
              <Input type="email" value={formData.email} onChange={(e) => setFormData({ ...formData, email: e.target.value })} placeholder="contact@example.com" />
            </FormField>
          </div>
          <FormField label="Address">
            <Textarea value={formData.address} onChange={(e) => setFormData({ ...formData, address: e.target.value })} placeholder="Company address" rows={2} />
          </FormField>
          <div className="form-row-2">
            <FormField label="Status">
              <Select value={formData.status} onChange={(e) => setFormData({ ...formData, status: e.target.value })}>
                {statusOptions.map((opt) => <option key={opt.value} value={opt.value}>{opt.label}</option>)}
              </Select>
            </FormField>
            <FormField label="Is Client">
              <label className="flex items-center gap-2 cursor-pointer">
                <input type="checkbox" checked={formData.is_client} onChange={(e) => setFormData({ ...formData, is_client: e.target.checked })} className="w-4 h-4 text-black border-border-primary rounded focus:ring-black" />
                <span className="text-sm">Mark as client</span>
              </label>
            </FormField>
          </div>
          <FormField label="Notes">
            <Textarea value={formData.notes} onChange={(e) => setFormData({ ...formData, notes: e.target.value })} placeholder="Internal notes" rows={3} />
          </FormField>
        </div>
        <div className="modal-footer">
          <Button variant="secondary" onClick={() => setModalOpen(false)} disabled={submitLoading}>Cancel</Button>
          <Button variant="primary" onClick={handleSubmit} disabled={submitLoading} loading={submitLoading}>
            {editingCompany ? 'Update' : 'Create'}
          </Button>
        </div>
      </Modal>

      <ConfirmDialog
        isOpen={!!deleteConfirm}
        onClose={() => setDeleteConfirm(null)}
        onConfirm={confirmDelete}
        title="Archive Company"
        message={`Are you sure you want to archive "${deleteConfirm?.name}"? The record will be removed from active views.`}
        confirmText="Archive"
        variant="danger"
        loading={submitLoading}
      />

      <ToastContainer toasts={toasts} onRemove={(id) => setToasts((prev) => prev.filter((t) => t.id !== id))} />
    </div>
  );
}

