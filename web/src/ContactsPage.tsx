import { exportCrmPage } from './crm-data';
import { CrmContactCards, CrmHeader, CrmSummary, CrmState, CrmPagination, CrmDialog as Modal } from './CrmUI';
import { workBatch } from './work-data';
import { useRecordRoute } from './useRecordRoute';
import { useState, useEffect, useCallback, useRef } from 'react';
import { Grid, Table as TableIcon } from 'lucide-react';
import { api, Contact, ContactCreateInput, Company } from './api';
import { Table, FormField, Input, Textarea, Select, Badge, ToastContainer, Toast, ConfirmDialog, getStatusBadge, Button, DataTableToolbar, Avatar } from './components';

const DEFAULT_PAGE_SIZE = 25;

const statusOptions = [
  { value: 'active', label: 'Active' },
  { value: 'inactive', label: 'Inactive' },
];

export function ContactsPage() {
  const [contacts, setContacts] = useState<Contact[]>([]);
  const [companies, setCompanies] = useState<Company[]>([]);
  const [total, setTotal] = useState(0);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(DEFAULT_PAGE_SIZE);
  const [query, setQuery] = useState('');
  const [statusFilter, setStatusFilter] = useState('');
  const [companyFilter, setCompanyFilter] = useState('');
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState('');
  const requestSequence = useRef(0);
  const [toasts, setToasts] = useState<Toast[]>([]);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingContact, setEditingContact] = useState<Contact | null>(null);
  const [formData, setFormData] = useState<ContactCreateInput>({
    first_name: '', last_name: '', email: '', phone: '', job_title: '',
    company_id: '', owner_id: '', status: 'active', tags: [],
    custom_fields: {}, notes: '',
  });
  const [formErrors, setFormErrors] = useState<Partial<ContactCreateInput>>({});
  const [submitLoading, setSubmitLoading] = useState(false);
  const [deleteConfirm, setDeleteConfirm] = useState<Contact | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [viewMode, setViewMode] = useState<'table' | 'cards'>('table');

  useEffect(() => { setSelectedIds(new Set()); }, [query, statusFilter, companyFilter, page, pageSize]);

  const showToast = useCallback((type: Toast['type'], message: string) => {
    const id = Date.now() + Math.random();
    setToasts((prev) => [...prev, { id, type, message }]);
    setTimeout(() => setToasts((prev) => prev.filter((t) => t.id !== id)), 5000);
  }, []);

  const fetchContacts = useCallback(async () => {
    const sequence = ++requestSequence.current;
    setLoading(true);
    setLoadError('');
    try {
      const response = await api.contacts.list({
        q: query,
        status: statusFilter || undefined,
        company_id: companyFilter || undefined,
        limit: pageSize,
        offset: (page - 1) * pageSize,
      });
      if (sequence !== requestSequence.current) return;
      if (page > Math.max(1, Math.ceil(response.total / pageSize))) { setPage(Math.max(1, Math.ceil(response.total / pageSize))); return; }
      setContacts(response.items);
      setTotal(response.total);
    } catch {
      if (sequence !== requestSequence.current) return;
      setLoadError('Failed to load contacts');
      setContacts([]);
    } finally {
      if (sequence === requestSequence.current) setLoading(false);
    }
  }, [query, statusFilter, companyFilter, page, pageSize]);

  const fetchCompanies = useCallback(async () => {
    try {
      const response = await api.companies.list({ limit: 1000, status: 'active' });
      setCompanies(response.items);
    } catch (error) {
      console.error('Failed to fetch companies:', error);
    }
  }, []);

  useEffect(() => {
    fetchContacts();
    fetchCompanies();
  }, [fetchContacts, fetchCompanies]);

  const handleCreate = () => {
    setEditingContact(null);
    setFormData({
      first_name: '', last_name: '', email: '', phone: '', job_title: '',
      company_id: '', owner_id: '', status: 'active', tags: [],
      custom_fields: {}, notes: '',
    });
    setFormErrors({});
    setModalOpen(true);
  };

  const handleEdit = (contact: Contact) => {
    setEditingContact(contact);
    setFormData({
      first_name: contact.first_name, last_name: contact.last_name,
      email: contact.email, phone: contact.phone || '',
      job_title: contact.job_title || '', company_id: contact.company_id || '',
      owner_id: contact.owner_id || '', status: contact.status,
      tags: contact.tags || [], custom_fields: contact.custom_fields || {},
      notes: contact.notes || '',
    });
    setFormErrors({});
    setModalOpen(true);
  };

  const handleDelete = (contact: Contact) => setDeleteConfirm(contact);

  const handleBulkArchive = async () => {
    if (submitLoading) return;
    setSubmitLoading(true);
    try {
      const result = await workBatch([...selectedIds], api.contacts.archive);
      setSelectedIds(new Set(result.failed));
      showToast(result.failed.length ? 'error' : 'success', result.succeeded.length + ' archived' + (result.failed.length ? '; ' + result.failed.length + ' failed. Retry the selected records.' : ' successfully'));
      await fetchContacts();
    } finally { setSubmitLoading(false); }
  };
  const confirmDelete = async () => {
    if (!deleteConfirm) return;
    setSubmitLoading(true);
    try {
      await api.contacts.archive(deleteConfirm.id);
      showToast('success', 'Contact archived successfully');
      fetchContacts();
    } catch {
      showToast('error', 'Failed to archive contact');
    } finally {
      setSubmitLoading(false);
      setDeleteConfirm(null);
    }
  };

  const validateForm = () => {
    const errors: Partial<ContactCreateInput> = {};
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
      if (editingContact) {
        await api.contacts.update(editingContact.id, formData);
        showToast('success', 'Contact updated successfully');
      } else {
        await api.contacts.create(formData);
        showToast('success', 'Contact created successfully');
      }
      setModalOpen(false);
      fetchContacts();
    } catch (error: any) {
      if (error.status === 400) showToast('error', error.message);
      else showToast('error', 'Failed to save contact');
    } finally {
      setSubmitLoading(false);
    }
  };

  const columns = [
    { key: 'first_name', header: 'Name', render: (c: Contact) => (
      <div className="flex items-center gap-3">
        <Avatar name={`${c.first_name} ${c.last_name}`} size="sm" />
        <div>
          <button className="crm-table-name" onClick={e => { e.stopPropagation(); handleEdit(c); }}>{c.first_name} {c.last_name}</button>
          {c.job_title && <p className="text-xs text-tertiary">{c.job_title}</p>}
        </div>
      </div>
    )},
    { key: 'email', header: 'Email', render: (c: Contact) => <a href={`mailto:${c.email}`} onClick={event => event.stopPropagation()} className="text-link hover:underline">{c.email}</a> },
    { key: 'phone', header: 'Phone', render: (c: Contact) => c.phone ? <a href={`tel:${c.phone}`} onClick={event => event.stopPropagation()} className="text-link hover:underline">{c.phone}</a> : '—' },
    { key: 'company_id', header: 'Company', render: (c: Contact) => {
      const company = companies.find(co => co.id === c.company_id);
      return company ? (
        <span className="font-medium">{company.name}</span>
      ) : <span className="text-tertiary">—</span>;
    }},
    { key: 'status', header: 'Status', render: (c: Contact) => <Badge variant={getStatusBadge(c.status)} size="sm" dot>{c.status}</Badge> },
  ];

  const filters = [
    { label: 'Status', value: 'status', options: statusOptions },
    { label: 'Company', value: 'company', options: [{ value: '', label: 'All' }, ...companies.map(c => ({ value: c.id, label: c.name }))] },
  ];

  const filterValues = { status: statusFilter, company: companyFilter };
  const hasActiveFilters = !!statusFilter || !!companyFilter;

  const handleFilterChange = (key: string, value: string) => {
    if (key === 'status') setStatusFilter(value);
    if (key === 'company') setCompanyFilter(value);
    setPage(1);
  };

  const clearFilters = () => {
    setStatusFilter('');
    setCompanyFilter('');
    setPage(1);
  };

  const handleSelectionChange = (ids: Set<string>) => setSelectedIds(ids);

  useRecordRoute({ getRecord: api.contacts.get, onCreate: handleCreate, onRecord: handleEdit, onError: message => showToast('error', message) });

  return (
    <div className="crm-area">
      <CrmHeader title="Contacts" description="Keep the people behind every opportunity close." onCreate={handleCreate} busy={loading || !!loadError || !contacts.length} onExport={() => exportCrmPage('contacts', contacts, [{ key: 'first_name', label: 'first name' }, { key: 'last_name', label: 'last name' }, { key: 'email', label: 'email' }, { key: 'phone', label: 'phone' }, { key: 'company_id', label: 'company id' }, { key: 'status', label: 'status' }])} />
      <CrmSummary loading={loading || !!loadError} items={[{ label: 'Matching contacts', value: total, note: 'With current filters' }, { label: 'Active relationships', value: contacts.filter(c => c.status === 'active').length, note: 'On this page' }, { label: 'Linked to a company', value: contacts.filter(c => c.company_id).length, note: 'On this page' }]} />

      <DataTableToolbar
        title="All Contacts"
        description={`${contacts.length} loaded on this page`}
        searchProps={{ query, onChange: (value: string) => { setQuery(value); setPage(1); }, placeholder: 'Search contacts by name or email...' }}
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

      <CrmState loading={loading} error={loadError} empty={!contacts.length} onRetry={fetchContacts} onCreate={handleCreate} noun="contacts" filtered={!!query || hasActiveFilters}>
      {viewMode === 'table' ? (
        <>
          <Table
            items={contacts}
            columns={columns}
            onEdit={handleEdit}
            onDelete={handleDelete}
            onRowClick={(contact) => handleEdit(contact)}
            loading={loading}
            emptyMessage="No contacts found. Create your first contact to get started."
            selectable
            selectedIds={selectedIds}
            onSelectionChange={handleSelectionChange}
          />
        </>
      ) : (
        <CrmContactCards items={contacts} companies={companies} selected={selectedIds} onSelect={setSelectedIds} onEdit={handleEdit} onArchive={handleDelete} />
      )}

      </CrmState>
      {!loadError && <CrmPagination page={page} pageSize={pageSize} total={total} onPage={setPage} onSize={size => { setPageSize(size); setPage(1); }} busy={loading || submitLoading} />}

      <Modal busy={submitLoading} isOpen={modalOpen} onClose={() => setModalOpen(false)} title={editingContact ? 'Edit Contact' : 'New Contact'} size="lg">
        <div className="space-y-4 max-h-[70vh] overflow-y-auto">
          <div className="form-row-2">
            <FormField label="First Name" error={formErrors.first_name} required>
              <Input value={formData.first_name} onChange={(e) => setFormData({ ...formData, first_name: e.target.value })} placeholder="First name" />
            </FormField>
            <FormField label="Last Name" error={formErrors.last_name} required>
              <Input value={formData.last_name} onChange={(e) => setFormData({ ...formData, last_name: e.target.value })} placeholder="Last name" />
            </FormField>
          </div>
          <FormField label="Email" error={formErrors.email} required>
            <Input type="email" value={formData.email} onChange={(e) => setFormData({ ...formData, email: e.target.value })} placeholder="contact@example.com" />
          </FormField>
          <FormField label="Phone">
            <Input value={formData.phone} onChange={(e) => setFormData({ ...formData, phone: e.target.value })} placeholder="+1 (555) 123-4567" />
          </FormField>
          <FormField label="Job Title">
            <Input value={formData.job_title} onChange={(e) => setFormData({ ...formData, job_title: e.target.value })} placeholder="Job title" />
          </FormField>
          <FormField label="Company">
            <Select value={formData.company_id} onChange={(e) => setFormData({ ...formData, company_id: e.target.value })}>
              <option value="">Select company (optional)</option>
              {companies.map((c) => <option key={c.id} value={c.id}>{c.name}</option>)}
            </Select>
          </FormField>
          <FormField label="Status">
            <Select value={formData.status} onChange={(e) => setFormData({ ...formData, status: e.target.value })}>
              {statusOptions.map((opt) => <option key={opt.value} value={opt.value}>{opt.label}</option>)}
            </Select>
          </FormField>
          <FormField label="Notes">
            <Textarea value={formData.notes} onChange={(e) => setFormData({ ...formData, notes: e.target.value })} placeholder="Internal notes" rows={3} />
          </FormField>
        </div>
        <div className="modal-footer">
          <Button variant="secondary" onClick={() => setModalOpen(false)} disabled={submitLoading}>Cancel</Button>
          <Button variant="primary" onClick={handleSubmit} disabled={submitLoading} loading={submitLoading}>
            {editingContact ? 'Update' : 'Create'}
          </Button>
        </div>
      </Modal>

      <ConfirmDialog
        isOpen={!!deleteConfirm}
        onClose={() => setDeleteConfirm(null)}
        onConfirm={confirmDelete}
        title="Archive Contact"
        message={`Are you sure you want to archive "${deleteConfirm?.first_name} ${deleteConfirm?.last_name}"? The record will be removed from active views.`}
        confirmText="Archive"
        variant="danger"
        loading={submitLoading}
      />

      <ToastContainer toasts={toasts} onRemove={(id) => setToasts((prev) => prev.filter((t) => t.id !== id))} />
    </div>
  );
}

// Icon imports handled at top