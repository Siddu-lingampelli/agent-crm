import { createContext, useContext, useId, useState, useRef, useEffect } from 'react';
import {
  Plus, Search, MoreHorizontal, Edit, Trash2, Eye, ChevronLeft, ChevronRight, Loader2,
  X, Check, ChevronDown, ChevronUp, Filter, Columns, Download, Upload, RefreshCw,
  Bell, User, LogOut, Settings, Menu, Sun, Moon, HelpCircle, Shield, Zap,
  Mail, Phone, MapPin, Globe, Link2, Tag, Star, Flag, Archive, ExternalLink,
  Calendar, Clock, DollarSign, TrendingUp, Users, Building2, Target, Activity,
  BarChart3, PieChart, LineChart, FileText, FolderOpen,
  Home, ChevronRight as ChevronRightIcon, AlertTriangle, Info, Briefcase, Grid, Table as TableIcon, Kanban as KanbanIcon, CheckCircle2, Calendar as CalendarIcon, Flag as FlagIcon, AlertCircle, CheckCircle, Inbox
} from 'lucide-react';
import { api } from './api';

// ============================================
// TYPES & INTERFACES
// ============================================

interface Column<T> {
  key: string;
  header: string;
  render?: (item: T) => React.ReactNode;
  className?: string;
  width?: string;
  minWidth?: string;
  align?: 'left' | 'center' | 'right';
  sortable?: boolean;
}

interface TableProps<T> {
  items: T[];
  columns: Column<T>[];
  onEdit?: (item: T) => void;
  onDelete?: (item: T) => void;
  onView?: (item: T) => void;
  onRowClick?: (item: T) => void;
  loading?: boolean;
  emptyMessage?: string;
  emptyIcon?: React.ReactNode;
  striped?: boolean;
  hoverable?: boolean;
  selectable?: boolean;
  selectedIds?: Set<string>;
  onSelectionChange?: (ids: Set<string>) => void;
  sortColumn?: string;
  sortDirection?: 'asc' | 'desc';
  onSort?: (column: string) => void;
}

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
  size?: 'sm' | 'md' | 'lg' | 'xl' | '2xl' | '3xl' | '4xl' | 'full';
  showCloseButton?: boolean;
  closeOnOverlayClick?: boolean;
  closeOnEscape?: boolean;
}

interface FormFieldProps {
  label: string;
  error?: string;
  required?: boolean;
  children: React.ReactNode;
  helpText?: string;
  className?: string;
}

interface BadgeProps {
  children: React.ReactNode;
  variant?: 'default' | 'primary' | 'success' | 'warning' | 'error' | 'info' | 'neutral';
  size?: 'sm' | 'md' | 'lg';
  dot?: boolean;
  className?: string;
}

interface PaginationProps {
  currentPage: number;
  totalPages: number;
  totalItems: number;
  pageSize: number;
  onPageChange: (page: number) => void;
  onPageSizeChange?: (size: number) => void;
  pageSizeOptions?: number[];
  showPageSize?: boolean;
  showTotal?: boolean;
}

interface SearchFilterProps {
  query: string;
  onQueryChange: (q: string) => void;
  placeholder?: string;
  filters?: FilterConfig[];
  filterValues?: Record<string, string>;
  onFilterChange?: (key: string, value: string) => void;
  onClearFilters?: () => void;
  hasActiveFilters?: boolean;
  className?: string;
}

interface FilterConfig {
  label: string;
  value: string;
  options: { value: string; label: string }[];
  type?: 'select' | 'date' | 'daterange' | 'multiselect';
  placeholder?: string;
}

interface Toast {
  id: number;
  type: 'success' | 'error' | 'warning' | 'info';
  title?: string;
  message: string;
  duration?: number;
  action?: { label: string; onClick: () => void };
}

interface ToastContainerProps {
  toasts: Toast[];
  onRemove: (id: number) => void;
}

interface ConfirmDialogProps {
  isOpen: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  variant?: 'danger' | 'primary' | 'warning';
  loading?: boolean;
  destructive?: boolean;
}

interface DropdownProps {
  trigger: React.ReactNode;
  items: DropdownItem[];
  align?: 'left' | 'right';
  offset?: number;
}

interface DropdownItem {
  label: string;
  onClick: () => void;
  icon?: React.ReactNode;
  disabled?: boolean;
  danger?: boolean;
  divider?: boolean;
  header?: string;
  shortcut?: string;
}

interface TabsProps {
  tabs: TabConfig[];
  activeTab: string;
  onChange: (tab: string) => void;
  variant?: 'default' | 'pills' | 'underline';
  className?: string;
}

interface TabConfig {
  id: string;
  label: string;
  icon?: React.ReactNode;
  count?: number;
  disabled?: boolean;
  children?: React.ReactNode;
}

interface AvatarProps {
  src?: string;
  alt?: string;
  name?: string;
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  status?: 'online' | 'offline' | 'busy' | 'away';
  className?: string;
}

interface AvatarGroupProps {
  avatars: AvatarProps[];
  max?: number;
  className?: string;
}

interface StatCardProps {
  title: string;
  value: string | number;
  icon: React.ReactNode;
  iconColor: string;
  trend?: {
    value: number;
    label: string;
    positive?: boolean;
  };
  loading?: boolean;
  onClick?: () => void;
  className?: string;
}

interface EmptyStateProps {
  icon?: React.ReactNode;
  title: string;
  description?: string;
  action?: React.ReactNode;
  className?: string;
}

interface BreadcrumbItem {
  label: string;
  href?: string;
  onClick?: () => void;
  current?: boolean;
  icon?: React.ReactNode;
}

interface BreadcrumbsProps {
  items: BreadcrumbItem[];
  separator?: React.ReactNode;
  className?: string;
}

interface KPIMetric {
  label: string;
  value: string | number;
  change?: number;
  changeLabel?: string;
  icon?: React.ReactNode;
  iconColor?: string;
  format?: 'number' | 'currency' | 'percentage' | 'compact';
}

interface DataTableToolbarProps {
  title: string;
  description?: string;
  actions?: React.ReactNode;
  searchProps?: {
    query: string;
    onChange: (q: string) => void;
    placeholder?: string;
  };
  filterProps?: {
    filters: FilterConfig[];
    values: Record<string, string>;
    onChange: (key: string, value: string) => void;
    onClear: () => void;
    hasActiveFilters: boolean;
  };
  viewOptions?: {
    currentView: string;
    views: { id: string; label: string; icon: React.ReactNode }[];
    onChange: (view: string) => void;
  };
  bulkActions?: {
    selectedCount: number;
    actions: { label: string; onClick: () => void; variant?: 'primary' | 'danger' }[];
  };
}

// ============================================
// TABLE COMPONENT
// ============================================

export function Table<T extends { id: string }>({
  items,
  columns,
  onEdit,
  onDelete,
  onView,
  onRowClick,
  loading = false,
  emptyMessage = 'No records found',
  emptyIcon,
  striped = false,
  hoverable = true,
  selectable = false,
  selectedIds = new Set(),
  onSelectionChange,
  sortColumn,
  sortDirection,
  onSort,
}: TableProps<T>) {
  const handleSelectAll = () => {
    const pageSelected = items.every(item => selectedIds.has(item.id));
    const selection = new Set(selectedIds);
    items.forEach(item => pageSelected ? selection.delete(item.id) : selection.add(item.id));
    onSelectionChange?.(selection);
  };

  const handleSelectRow = (id: string) => {
    const newSelection = new Set(selectedIds);
    if (newSelection.has(id)) {
      newSelection.delete(id);
    } else {
      newSelection.add(id);
    }
    onSelectionChange?.(newSelection);
  };

  if (loading) {
    return (
      <div className="table-container">
        <table className="table">
          <thead>
            <tr>
              {columns.map((col) => (
                <th key={col.key} className={col.className} style={{ width: col.width, minWidth: col.minWidth }}>
                  {col.header}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {Array.from({ length: 5 }).map((_, i) => (
              <tr key={i}>
                {columns.map((col) => (
                  <td key={col.key} className={col.className}>
                    <div className="skeleton skeleton-text" style={{ width: col.width || '100%' }} />
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <div className="empty-state">
        <div className="empty-state-icon">
          {emptyIcon || <Search className="w-12 h-12" />}
        </div>
        <p className="empty-state-title">No Records</p>
        <p className="empty-state-text">{emptyMessage}</p>
      </div>
    );
  }

  const allSelected = items.length > 0 && items.every(item => selectedIds.has(item.id));

  return (
    <div className="table-container">
      <table className={`table ${striped ? 'table-striped' : ''} ${hoverable ? 'table-hoverable' : ''} ${onRowClick ? 'table-clickable' : ''}`}>
        <thead>
          <tr>
            {selectable && (
              <th className="w-12 text-center">
                <input
                  type="checkbox"
                  checked={allSelected}
                  onChange={handleSelectAll}
                  className="checkbox-input"
                  aria-label="Select all rows on this page"
                />
              </th>
            )}
            {columns.map((col) => (
              <th
                key={col.key}
                className={`${col.className || ''} ${col.sortable && onSort ? 'cursor-pointer select-none' : ''} ${col.align === 'center' ? 'text-center' : col.align === 'right' ? 'text-right' : ''}`}
                style={{ width: col.width, minWidth: col.minWidth }}
                onClick={() => col.sortable && onSort?.(col.key)}
                scope="col"
                tabIndex={col.sortable && onSort ? 0 : undefined}
                onKeyDown={event => {
                  if (col.sortable && onSort && (event.key === 'Enter' || event.key === ' ')) { event.preventDefault(); onSort(col.key); }
                }}
                aria-sort={col.sortable && onSort ? sortColumn === col.key ? (sortDirection === 'asc' ? 'ascending' : 'descending') : 'none' : undefined}
              >
                <div className="flex items-center gap-1.5">
                  <span>{col.header}</span>
                  {col.sortable && onSort && sortColumn === col.key && (
                    <>
                      <ChevronUp className={sortDirection === 'asc' ? 'w-4 h-4 text-black' : 'w-4 h-4 text-gray-400'} />
                      <ChevronDown className={sortDirection === 'desc' ? 'w-4 h-4 text-black' : 'w-4 h-4 text-gray-400'} />
                    </>
                  )}
                </div>
              </th>
            ))}
            {(onEdit || onDelete || onView) && (
              <th className="w-24 text-center">Actions</th>
            )}
          </tr>
        </thead>
        <tbody>
          {items.map((item) => {
            const isSelected = selectedIds.has(item.id);
            return (
              <tr
                key={item.id}
                className={`${isSelected ? 'bg-gray-50' : ''} ${onRowClick ? 'cursor-pointer' : ''}`}
                onClick={() => onRowClick?.(item)}
                onDoubleClick={() => onView?.(item)}
              >
                {selectable && (
                  <td className="text-center">
                    <input
                      type="checkbox"
                      checked={isSelected}
                      onChange={() => handleSelectRow(item.id)}
                      onClick={(e) => e.stopPropagation()}
                      className="checkbox-input"
                      aria-label={`Select ${item.id}`}
                    />
                  </td>
                )}
                {columns.map((col) => (
                  <td
                    key={col.key}
                    className={`${col.className || ''} ${col.align === 'center' ? 'text-center' : col.align === 'right' ? 'text-right' : ''}`}
                  >
                    {col.render ? col.render(item) : String((item as any)[col.key] ?? '')}
                  </td>
                ))}
                {(onEdit || onDelete || onView) && (
                  <td className="text-center">
                    <div className="flex items-center justify-center gap-1">
                      {onView && (
                        <button
                          onClick={(e) => { e.stopPropagation(); onView(item); }}
                          className="btn-icon btn-ghost p-1.5"
                          aria-label="View"
                        >
                          <Eye className="w-4 h-4" />
                        </button>
                      )}
                      {onEdit && (
                        <button
                          onClick={(e) => { e.stopPropagation(); onEdit(item); }}
                          className="btn-icon btn-ghost p-1.5"
                          aria-label="Edit"
                        >
                          <Edit className="w-4 h-4" />
                        </button>
                      )}
                      {onDelete && (
                        <button
                          onClick={(e) => { e.stopPropagation(); onDelete(item); }}
                          className="btn-icon btn-ghost p-1.5 text-red-600 hover:text-red-700"
                          aria-label="Delete"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      )}
                    </div>
                  </td>
                )}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

// ============================================
// MODAL COMPONENT
// ============================================

export function Modal({
  isOpen,
  onClose,
  title,
  children,
  footer,
  size = 'md',
  showCloseButton = true,
  closeOnOverlayClick = true,
  closeOnEscape = true,
}: ModalProps) {
  if (!isOpen) return null;

  const sizeClasses = {
    sm: 'modal-sm',
    md: 'modal-md',
    lg: 'modal-lg',
    xl: 'modal-xl',
    '2xl': 'modal-2xl',
    '3xl': 'modal-3xl',
    '4xl': 'modal-4xl',
    full: 'modal-full',
  };

  return (
    <div
      className="modal-overlay"
      onClick={closeOnOverlayClick ? onClose : undefined}
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
    >
      <div className={`modal-content ${sizeClasses[size]}`} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2 id="modal-title" className="modal-title">{title}</h2>
          {showCloseButton && (
            <button
              onClick={onClose}
              className="modal-close"
              aria-label="Close"
            >
              <X className="w-5 h-5" />
            </button>
          )}
        </div>
        <div className="modal-body">{children}</div>
        {footer && <div className="modal-footer">{footer}</div>}
      </div>
    </div>
  );
}

// ============================================
// FORM COMPONENTS
// ============================================

const FormFieldContext = createContext<{ id: string; labelId: string; descriptionId?: string; invalid: boolean } | null>(null);

export function FormField({ label, error, required, children, helpText, className }: FormFieldProps) {
  const id = useId();
  const labelId = `${id}-label`;
  const descriptionId = error || helpText ? `${id}-description` : undefined;
  return (
    <div className={`form-group ${className || ''}`}>
      <label className="form-label" id={labelId} htmlFor={id}>{label} {required && <span className="text-red-600">*</span>}</label>
      <FormFieldContext.Provider value={{ id, labelId, descriptionId, invalid: !!error }}>{children}</FormFieldContext.Provider>
      {error && <p id={descriptionId} className="form-error"><X className="w-3 h-3" /> {error}</p>}
      {helpText && !error && <p id={descriptionId} className="form-help">{helpText}</p>}
    </div>
  );
}

export function Input({ error, className, icon, ...props }: React.InputHTMLAttributes<HTMLInputElement> & { error?: string; icon?: React.ReactNode }) {
  const field = useContext(FormFieldContext);
  return (
    <div className="relative">
      {icon && <div className="input-icon-left" aria-hidden="true">{icon}</div>}
      <input
        id={field?.id}
        aria-labelledby={field?.labelId}
        aria-describedby={field?.descriptionId}
        aria-invalid={!!error || field?.invalid || undefined}
        className={`form-input ${error ? 'form-input-error' : ''} ${icon ? 'pl-10' : ''} ${className || ''}`}
        {...props}
      />
    </div>
  );
}

export function Textarea({ error, className, ...props }: React.TextareaHTMLAttributes<HTMLTextAreaElement> & { error?: string }) {
  const field = useContext(FormFieldContext);
  return (
    <textarea
      id={field?.id}
      aria-labelledby={field?.labelId}
      aria-describedby={field?.descriptionId}
      aria-invalid={!!error || field?.invalid || undefined}
      className={`form-textarea ${error ? 'form-textarea-error' : ''} ${className || ''}`}
      {...props}
    />
  );
}

export function Select({ error, className, ...props }: React.SelectHTMLAttributes<HTMLSelectElement> & { error?: string }) {
  const field = useContext(FormFieldContext);
  return (
    <select
      id={field?.id}
      aria-labelledby={field?.labelId}
      aria-describedby={field?.descriptionId}
      aria-invalid={!!error || field?.invalid || undefined}
      className={`form-select ${error ? 'form-select-error' : ''} ${className || ''}`}
      {...props}
    />
  );
}

export function Checkbox({ checked, onChange, className = '', id, label, ...props }: {
  checked: boolean;
  onChange: (checked: boolean) => void;
  className?: string;
  id: string;
  label: string;
} & Omit<React.InputHTMLAttributes<HTMLInputElement>, 'checked' | 'onChange'>) {
  return (
    <label className={`checkbox ${className}`}>
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        id={id}
        className="checkbox-input"
        {...props}
      />
      <span className="checkbox-label">{label}</span>
    </label>
  );
}

export function Switch({ checked, onChange, className = '', id, label, ...props }: {
  checked: boolean;
  onChange: (checked: boolean) => void;
  className?: string;
  id?: string;
  label?: string;
} & Omit<React.InputHTMLAttributes<HTMLInputElement>, 'checked' | 'onChange'>) {
  return (
    <label className={`switch ${className}`}>
      <input
        type="checkbox"
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        id={id}
        className="switch-input"
        {...props}
      />
      {label && <span className="switch-label">{label}</span>}
    </label>
  );
}

// ============================================
// BADGE COMPONENT
// ============================================

export function Badge({ children, variant = 'default', size = 'md', dot = false, className = '' }: BadgeProps) {
  const variantClasses = {
    default: 'badge-neutral',
    primary: 'badge-primary',
    success: 'badge-success',
    warning: 'badge-warning',
    error: 'badge-error',
    info: 'badge-info',
    neutral: 'badge-neutral',
  };

  const sizeClasses = {
    sm: 'badge-sm',
    md: 'badge-md',
    lg: 'badge-lg',
  };

  return (
    <span className={`badge ${variantClasses[variant]} ${sizeClasses[size]} ${dot ? 'badge-dot' : ''} ${className}`}>
      {children}
    </span>
  );
}

// Status badge helper
export function getStatusBadge(status: string): BadgeProps['variant'] {
  const statusMap: Record<string, BadgeProps['variant']> = {
    success: 'primary',
    inactive: 'neutral',
    won: 'primary',
    lost: 'error',
    pending: 'warning',
    open: 'primary',
    closed: 'neutral',
    new: 'info',
    qualified: 'primary',
    proposal: 'primary',
    negotiation: 'warning',
    discovery: 'info',
    in_progress: 'primary',
    completed: 'primary',
    on_hold: 'warning',
    cancelled: 'error',
    action: 'primary',
    condition: 'info',
    delay: 'warning',
    webhook: 'primary',
    mcp_call: 'primary',
    manual: 'neutral',
    schedule: 'primary',
    event: 'primary',
    connected: 'primary',
    disconnected: 'neutral',
    error: 'error',
    disabled: 'neutral',
  };
  return statusMap[status] || 'default';
}

// ============================================
// PAGINATION COMPONENT
// ============================================

export function Pagination({
  currentPage,
  totalPages,
  totalItems,
  pageSize,
  onPageChange,
  onPageSizeChange,
  pageSizeOptions = [10, 25, 50, 100],
  showPageSize = true,
  showTotal = true,
}: PaginationProps) {
  if (totalPages <= 1) return null;

  const pages = Array.from({ length: Math.min(totalPages, 5) }, (_, i) => {
    if (totalPages <= 5) return i + 1;
    if (currentPage <= 3) return i + 1;
    if (currentPage >= totalPages - 2) return totalPages - 4 + i;
    return currentPage - 2 + i;
  });

  return (
    <div className="pagination">
      <div className="flex items-center gap-4">
        {showTotal && (
          <span className="pagination-info">
            Showing {(currentPage - 1) * pageSize + 1} to {Math.min(currentPage * pageSize, totalItems)} of {totalItems}
          </span>
        )}
        {showPageSize && onPageSizeChange && (
          <select
            value={pageSize}
            onChange={(e) => onPageSizeChange(Number(e.target.value))}
            className="form-select w-auto"
            aria-label="Page size"
          >
            {pageSizeOptions.map((size) => (
              <option key={size} value={size}>{size} per page</option>
            ))}
          </select>
        )}
      </div>
      <div className="pagination-controls">
        <button
          onClick={() => onPageChange(currentPage - 1)}
          disabled={currentPage === 1}
          className="pagination-btn"
          aria-label="Previous page"
        >
          <ChevronLeft className="w-5 h-5" />
        </button>
        {pages.map((page) => (
          <button
            key={page}
            onClick={() => onPageChange(page)}
            className={`pagination-page ${page === currentPage ? 'pagination-page-active' : ''}`}
            aria-label={`Page ${page}`}
            aria-current={page === currentPage ? 'page' : undefined}
          >
            {page}
          </button>
        ))}
        <button
          onClick={() => onPageChange(currentPage + 1)}
          disabled={currentPage === totalPages}
          className="pagination-btn"
          aria-label="Next page"
        >
          <ChevronRight className="w-5 h-5" />
        </button>
      </div>
    </div>
  );
}

// ============================================
// SEARCH & FILTER BAR
// ============================================

export function SearchFilter({ query, onQueryChange, placeholder = 'Search...', filters = [], filterValues = {}, onFilterChange, onClearFilters, hasActiveFilters = false, className = '' }: SearchFilterProps) {
  return (
    <div className={`search-filter-bar ${className}`}>
      <div className="search-input-wrapper">
        <Search className="search-icon" />
        <input
          type="search"
          value={query}
          onChange={(e) => onQueryChange(e.target.value)}
          placeholder={placeholder}
          className="search-input"
        />
      </div>
      {filters.map((filter) => {
        if (filter.type === 'date') {
          return (
            <input
              key={filter.label}
              type="date"
              value={filterValues[filter.value] || ''}
              onChange={(e) => onFilterChange?.(filter.value, e.target.value)}
              className="filter-select"
              placeholder={filter.placeholder}
            />
          );
        }
        return (
          <select
            key={filter.label}
            value={filterValues[filter.value] || ''}
            onChange={(e) => onFilterChange?.(filter.value, e.target.value)}
            className="filter-select"
          >
            <option value="">All {filter.label}</option>
            {filter.options.map((opt) => (
              <option key={opt.value} value={opt.value}>{opt.label}</option>
            ))}
          </select>
        );
      })}
      {hasActiveFilters && onClearFilters && (
        <button onClick={onClearFilters} className="btn-secondary btn-sm">
          <Filter className="w-4 h-4 mr-1" /> Clear filters
        </button>
      )}
    </div>
  );
}

// ============================================
// TOAST NOTIFICATIONS
// ============================================

export function ToastContainer({ toasts, onRemove }: ToastContainerProps) {
  return (
    <div className="toast-container" aria-live="polite" aria-label="Notifications">
      {toasts.map((toast) => (
        <div
          key={toast.id}
          className={`toast toast-${toast.type} flex items-start gap-3 px-4 py-3 rounded-xl shadow-xl border animate-slide-in-right`}
        >
          <div className="toast-icon flex-shrink-0 mt-0.5">
            {toast.type === 'success' && <Check className="w-5 h-5 text-green-600" />}
            {toast.type === 'error' && <X className="w-5 h-5 text-red-600" />}
            {toast.type === 'warning' && <AlertTriangle className="w-5 h-5 text-amber-600" />}
            {toast.type === 'info' && <Info className="w-5 h-5 text-blue-600" />}
          </div>
          <div className="toast-content flex-1 min-w-0">
            {toast.title && <p className="toast-title">{toast.title}</p>}
            <p className="toast-message">{toast.message}</p>
          </div>
          <div className="flex items-center gap-2">
            {toast.action && (
              <button onClick={toast.action.onClick} className="text-sm font-medium underline hover:no-underline">
                {toast.action.label}
              </button>
            )}
            <button onClick={() => onRemove(toast.id)} className="toast-close">
              <X className="w-4 h-4" />
            </button>
          </div>
        </div>
      ))}
    </div>
  );
}

// ============================================
// CONFIRMATION DIALOG
// ============================================

export function ConfirmDialog({ isOpen, onClose, onConfirm, title, message, confirmText = 'Confirm', cancelText = 'Cancel', variant = 'danger', loading = false, destructive = true }: ConfirmDialogProps) {
  if (!isOpen) return null;

  const variantClasses = {
    danger: 'btn-danger',
    primary: 'btn-primary',
    warning: 'btn-warning',
  };

  return (
    <div className="modal-overlay" onClick={onClose} role="dialog" aria-modal="true" aria-labelledby="confirm-title">
      <div className="modal-content max-w-md" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2 id="confirm-title" className="modal-title">{title}</h2>
        </div>
        <div className="modal-body">
          <p className="text-gray-600">{message}</p>
        </div>
        <div className="modal-footer">
          <button onClick={onClose} disabled={loading} className="btn-secondary">{cancelText}</button>
          <button
            onClick={onConfirm}
            disabled={loading}
            className={variantClasses[variant]}
          >
            {loading ? (
              <>
                <Loader2 className="loading-spinner w-4 h-4 mr-2" />
                <span>Processing...</span>
              </>
            ) : (
              confirmText
            )}
          </button>
        </div>
      </div>
    </div>
  );
}

// ============================================
// DROPDOWN MENU
// ============================================

export function Dropdown({ trigger, items, align = 'right', offset = 4 }: DropdownProps) {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  return (
    <div className="dropdown" ref={dropdownRef}>
      <div onClick={() => setIsOpen(!isOpen)}>
        {trigger}
      </div>
      {isOpen && (
        <div
          className={`dropdown-menu ${align === 'right' ? 'right-0' : 'left-0'}`}
          style={{ marginTop: offset }}
          role="menu"
        >
          {items.map((item, index) => {
            if (item.divider) {
              return <div key={index} className="dropdown-divider" role="separator" />;
            }
            if (item.header) {
              return <div key={index} className="dropdown-header">{item.header}</div>;
            }
            return (
              <button
                key={index}
                onClick={() => { item.onClick(); setIsOpen(false); }}
                disabled={item.disabled}
                className={`dropdown-item w-full text-left ${item.danger ? 'text-red-600 hover:bg-red-50' : ''} ${item.disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
                role="menuitem"
              >
                {item.icon && <span className="w-4 h-4">{item.icon}</span>}
                <span className="flex-1">{item.label}</span>
                {item.shortcut && <kbd className="px-1.5 py-0.5 text-xs bg-gray-100 rounded">{item.shortcut}</kbd>}
              </button>
            );
          })}
        </div>
      )}
    </div>
  );
}

// ============================================
// TABS COMPONENT
// ============================================

export function Tabs({ tabs, activeTab, onChange, variant = 'default', className = '' }: TabsProps) {
  const variantStyles = {
    default: 'border-b border-gray-200',
    pills: 'gap-1 bg-gray-100 p-1 rounded-lg',
    underline: 'border-b border-gray-200',
  };

  const triggerStyles = {
    default: 'px-4 py-3 text-sm font-medium text-gray-500 border-b-2 border-transparent -mb-px hover:text-black hover:border-gray-300 transition-all',
    pills: 'px-4 py-2 text-sm font-medium text-gray-500 rounded-md hover:text-black hover:bg-gray-200 transition-all',
    underline: 'px-4 py-3 text-sm font-medium text-gray-500 border-b-2 border-transparent -mb-px hover:text-black hover:border-gray-300 transition-all',
  };

  const activeStyles = {
    default: 'text-black border-orange-500',
    pills: 'bg-white text-black shadow-sm',
    underline: 'text-black border-orange-500',
  };

  return (
    <div className={className}>
      <div className={`tabs-list flex ${variantStyles[variant]}`} role="tablist">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => !tab.disabled && onChange(tab.id)}
            disabled={tab.disabled}
            className={`tab-trigger ${triggerStyles[variant]} ${activeTab === tab.id ? activeStyles[variant] : ''} ${tab.disabled ? 'opacity-50 cursor-not-allowed' : ''}`}
            role="tab"
            aria-selected={activeTab === tab.id}
            aria-controls={`panel-${tab.id}`}
            id={`tab-${tab.id}`}
          >
            {tab.icon && <span className="flex items-center gap-2">{tab.icon}</span>}
            {tab.label}
            {tab.count !== undefined && (
              <span className="ml-2 px-1.5 py-0.5 text-xs rounded-full bg-gray-100">
                {tab.count}
              </span>
            )}
          </button>
        ))}
      </div>
      <div id={`panel-${activeTab}`} role="tabpanel" aria-labelledby={`tab-${activeTab}`} className="tab-content">
        {tabs.find(t => t.id === activeTab)?.children}
      </div>
    </div>
  );
}

// ============================================
// AVATAR COMPONENTS
// ============================================

export function Avatar({ src, alt, name, size = 'md', status, className = '' }: AvatarProps) {
  const sizeClasses = {
    xs: 'w-6 h-6 text-xs',
    sm: 'w-8 h-8 text-xs',
    md: 'w-10 h-10 text-sm',
    lg: 'w-12 h-12 text-base',
    xl: 'w-16 h-16 text-xl',
  };

  const getInitials = (name: string) => {
    return name
      .split(' ')
      .map(n => n[0])
      .join('')
      .toUpperCase()
      .slice(0, 2);
  };

  const getColorFromName = (name: string) => {
    const colors = [
      'bg-orange-100 text-orange-700',
      'bg-green-100 text-green-700',
      'bg-amber-100 text-amber-700',
      'bg-blue-100 text-blue-700',
      'bg-purple-100 text-purple-700',
      'bg-pink-100 text-pink-700',
    ];
    let hash = 0;
    for (let i = 0; i < name.length; i++) {
      hash = name.charCodeAt(i) + ((hash << 5) - hash);
    }
    return colors[Math.abs(hash) % colors.length];
  };

  return (
    <div className={`avatar relative inline-flex ${sizeClasses[size]} ${className}`}>
      {src ? (
        <img src={src} alt={alt || name || 'Avatar'} className="avatar-img rounded-full" />
      ) : (
        <span className={getColorFromName(name || 'User')}>{getInitials(name || 'User')}</span>
      )}
      {status && (
        <span className={`avatar-${status}`} aria-label={status} />
      )}
    </div>
  );
}

export function AvatarGroup({ avatars, max = 5, className = '' }: AvatarGroupProps) {
  const visibleAvatars = avatars.slice(0, max);
  const remainingCount = avatars.length - max;

  return (
    <div className={`avatar-group ${className}`} role="group" aria-label={`${avatars.length} people`}>
      {visibleAvatars.map((avatar, index) => (
        <Avatar key={index} {...avatar} className="border-2 border-white" />
      ))}
      {remainingCount > 0 && (
        <div className="avatar border-2 border-white bg-gray-100 text-gray-500">
          +{remainingCount}
        </div>
      )}
    </div>
  );
}

// ============================================
// STAT CARD
// ============================================

export function StatCard({ title, value, icon, iconColor, trend, loading, onClick, className = '' }: StatCardProps) {
  return (
    <div className={`stat-card ${onClick ? 'cursor-pointer' : ''} ${className}`} onClick={onClick}>
      <div className="stat-card-header">
        <span className="stat-card-title">{title}</span>
        <div className={`stat-card-icon p-2 rounded-lg ${iconColor}`}>
          {icon}
        </div>
      </div>
      {loading ? (
        <div className="skeleton skeleton-title w-24" />
      ) : (
        <p className="stat-card-value">{value}</p>
      )}
      {trend && (
        <div className="stat-card-trend">
          <TrendingUp className={`w-4 h-4 ${trend.positive ? 'stat-card-trend-positive' : 'stat-card-trend-negative'}`} />
          <span className={`${trend.positive ? 'stat-card-trend-positive' : 'stat-card-trend-negative'}`}>
            {trend.positive ? '+' : ''}{trend.value}% {trend.label}
          </span>
        </div>
      )}
    </div>
  );
}

// ============================================
// EMPTY STATE
// ============================================

export function EmptyState({ icon, title, description, action, className = '' }: EmptyStateProps) {
  return (
    <div className={`empty-state ${className}`}>
      <div className="empty-state-icon text-gray-400">
        {icon || <Inbox className="w-16 h-16" />}
      </div>
      <h2 className="empty-state-title">{title}</h2>
      {description && <p className="empty-state-text">{description}</p>}
      {action && <div className="empty-state-action">{action}</div>}
    </div>
  );
}

// ============================================
// BREADCRUMBS
// ============================================

export function Breadcrumbs({ items, separator = <ChevronRightIcon className="w-4 h-4 text-gray-400" />, className = '' }: BreadcrumbsProps) {
  return (
    <nav className={`breadcrumbs ${className}`} aria-label="Breadcrumb">
      <ol className="flex items-center gap-1.5 text-xs">
        {items.map((item, index) => (
          <li key={item.label} className="breadcrumb-item flex items-center gap-1.5">
            {index > 0 && <span className="breadcrumb-separator" aria-hidden="true">{separator}</span>}
            {item.href || item.onClick ? (
              <a
                href={item.href}
                onClick={item.onClick}
                className="breadcrumb-link flex items-center gap-1.5"
                aria-current={item.current ? 'page' : undefined}
              >
                {item.icon && <span>{item.icon}</span>}
                {item.label}
              </a>
            ) : (
              <span className="breadcrumb-current flex items-center gap-1.5" aria-current="page">
                {item.icon && <span>{item.icon}</span>}
                {item.label}
              </span>
            )}
          </li>
        ))}
      </ol>
    </nav>
  );
}

// ============================================
// DATA TABLE TOOLBAR
// ============================================

export function DataTableToolbar({ title, description, actions, searchProps, filterProps, viewOptions, bulkActions }: DataTableToolbarProps) {
  return <div className="record-toolbar">
    <div className="record-toolbar-head"><div><h2>{title}</h2>{description && <p>{description}</p>}</div>{actions}</div>
    <div className="record-toolbar-controls">
      {searchProps && <div className="search-input-wrapper"><Search className="search-icon" aria-hidden="true" /><input type="search" value={searchProps.query} onChange={event => searchProps.onChange(event.target.value)} placeholder={searchProps.placeholder || 'Search records...'} aria-label={searchProps.placeholder || 'Search records'} className="search-input" /></div>}
      {filterProps && <div className="record-filters">{filterProps.filters.map(filter => <select key={filter.value} value={filterProps.values[filter.value] || ''} onChange={event => filterProps.onChange(filter.value, event.target.value)} className="filter-select" aria-label={filter.label}><option value="">All {filter.label}</option>{filter.options.filter(option => option.value !== '').map(option => <option key={option.value} value={option.value}>{option.label}</option>)}</select>)}{filterProps.hasActiveFilters && <button onClick={filterProps.onClear} className="btn btn-ghost btn-sm"><X size={13} />Clear</button>}</div>}
      {viewOptions && <div className="btn-group" role="group" aria-label="View options">{viewOptions.views.map(view => <button key={view.id} onClick={() => viewOptions.onChange(view.id)} className={'btn ' + (viewOptions.currentView === view.id ? 'btn-primary' : 'btn-ghost')} aria-label={view.label} title={view.label} aria-pressed={viewOptions.currentView === view.id}>{view.icon}</button>)}</div>}
    </div>
    {bulkActions && bulkActions.selectedCount > 0 && <div className="record-bulk" role="region" aria-label="Selected record actions"><span>{bulkActions.selectedCount} selected</span>{bulkActions.actions.map(action => <button key={action.label} onClick={action.onClick} className={'btn btn-sm ' + (action.variant === 'danger' ? 'btn-danger' : action.variant === 'primary' ? 'btn-primary' : 'btn-secondary')}>{action.label}</button>)}</div>}
  </div>;
}

// ============================================
// LAYOUT COMPONENTS
// ============================================

interface LayoutProps {
  sidebar: React.ReactNode;
  header: React.ReactNode;
  children: React.ReactNode;
}

export function Layout({ sidebar, header, children }: LayoutProps) {
  return (
    <div className="flex h-screen bg-gray-50">
      <div className="fixed inset-y-0 left-0 z-50 w-64 bg-white border-r border-gray-200 transform transition-transform duration-200 ease-in-out lg:translate-x-0 lg:static lg:block">
        {sidebar}
      </div>
      <div className="flex-1 flex flex-col min-w-0 lg:ml-0">
        {header}
        <main className="flex-1 overflow-auto p-6 lg:p-8">
          {children}
        </main>
      </div>
    </div>
  );
}

interface SidebarProps {
  isOpen: boolean;
  onClose: () => void;
  navigation: Array<{ name: string; href: string; icon: React.ReactNode }>;
  adminNavigation: Array<{ name: string; href: string; icon: React.ReactNode }>;
  user?: { name: string; email: string; avatar?: string };
  onThemeToggle?: () => void;
  isDark?: boolean;
}

export function Sidebar({ isOpen, onClose, navigation, adminNavigation, user, onThemeToggle, isDark }: SidebarProps) {
  return (
    <aside
      className={`fixed inset-y-0 left-0 z-50 w-64 bg-white border-r border-gray-200 transform transition-transform duration-200 ease-in-out ${isOpen ? 'translate-x-0' : '-translate-x-full'} lg:translate-x-0 lg:static lg:block`}
      aria-label="Sidebar"
    >
      <div className="flex flex-col h-full">
        <div className="flex items-center justify-between h-16 px-4 border-b border-gray-200">
          <div className="flex items-center gap-2">
            <div className="w-8 h-8 bg-orange-600 rounded-lg flex items-center justify-center">
              <span className="text-white font-bold text-lg">T</span>
            </div>
            <span className="font-bold text-lg text-gray-900">TORQ</span>
          </div>
          <button onClick={onClose} className="lg:hidden p-2 text-gray-500 hover:text-gray-700">
            <X className="w-5 h-5" />
          </button>
        </div>

        <nav className="flex-1 overflow-y-auto p-4 space-y-1" aria-label="Main navigation">
          {navigation.map((item) => (
            <a
              key={item.name}
              href={item.href}
              className="flex items-center gap-3 px-3 py-2 text-sm font-medium text-gray-600 rounded-lg hover:bg-gray-100 hover:text-gray-900 transition-colors"
            >
              <span className="w-5 h-5">{item.icon}</span>
              {item.name}
            </a>
          ))}

          <div className="pt-4 border-t border-gray-200">
            <h3 className="px-3 text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2">Administration</h3>
            {adminNavigation.map((item) => (
              <a
                key={item.name}
                href={item.href}
                className="flex items-center gap-3 px-3 py-2 text-sm font-medium text-gray-600 rounded-lg hover:bg-gray-100 hover:text-gray-900 transition-colors"
              >
                <span className="w-5 h-5">{item.icon}</span>
                {item.name}
              </a>
            ))}
          </div>
        </nav>

        <div className="p-4 border-t border-gray-200">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-full bg-gray-100 flex items-center justify-center flex-shrink-0">
              <span className="text-sm font-medium text-gray-600">AJ</span>
            </div>
            <div className="flex-1 min-w-0">
              <p className="text-sm font-medium text-gray-900 truncate">{user?.name}</p>
              <p className="text-xs text-gray-500 truncate">{user?.email}</p>
            </div>
          </div>
        </div>
      </div>
    </aside>
  );
}

interface HeaderProps {
  onToggleSidebar: () => void;
  notifications?: number;
  user?: { name: string };
}

export function Header({ onToggleSidebar, notifications = 0, user }: HeaderProps) {
  return (
    <header className="sticky top-0 z-40 h-16 bg-white border-b border-gray-200">
      <div className="flex items-center justify-between h-full px-4 lg:px-8">
        <div className="flex items-center gap-4">
          <button
            onClick={onToggleSidebar}
            className="lg:hidden p-2 text-gray-500 hover:text-gray-700 rounded-lg hover:bg-gray-100"
            aria-label="Toggle sidebar"
          >
            <Menu className="w-6 h-6" />
          </button>
          <div className="hidden lg:block">
            <h1 className="text-lg font-semibold text-gray-900">Dashboard</h1>
          </div>
        </div>
        <div className="flex items-center gap-4">
          {notifications > 0 && (
            <button className="relative p-2 text-gray-500 hover:text-gray-700 rounded-lg hover:bg-gray-100">
              <Bell className="w-5 h-5" />
              <span className="absolute -top-1 -right-1 w-5 h-5 bg-orange-600 text-white text-xs font-medium rounded-full flex items-center justify-center">
                {notifications}
              </span>
            </button>
          )}
          <button className="p-2 text-gray-500 hover:text-gray-700 rounded-lg hover:bg-gray-100" aria-label="Settings">
            <Settings className="w-5 h-5" />
          </button>
          <div className="w-8 h-8 rounded-full bg-gray-100 flex items-center justify-center">
            <span className="text-sm font-medium text-gray-600">AJ</span>
          </div>
        </div>
      </div>
    </header>
  );
}

interface MainContentProps {
  children: React.ReactNode;
}

export function MainContent({ children }: MainContentProps) {
  return (
    <main className="flex-1 overflow-auto p-6 lg:p-8">
      {children}
    </main>
  );
}

export function PageHeader({
  title,
  description,
  icon,
  actions,
  breadcrumbs,
}: {
  title: string;
  description?: string;
  icon?: React.ReactNode;
  actions?: React.ReactNode;
  breadcrumbs?: Array<{ label: string; href?: string; onClick?: () => void; current?: boolean; icon?: React.ReactNode }>;
}) {
  return (
    <div className="record-page-header space-y-4 mb-6">
      {breadcrumbs && (
        <nav className="breadcrumbs" aria-label="Breadcrumb">
          <ol className="flex items-center gap-1.5 text-xs">
            {breadcrumbs.map((item, index) => (
              <li key={item.label} className="breadcrumb-item flex items-center gap-1.5">
                {index > 0 && <span className="breadcrumb-separator" aria-hidden="true"><ChevronRightIcon className="w-4 h-4 text-gray-400" /></span>}
                {item.href ? (
                  <a href={item.href} className="breadcrumb-link flex items-center gap-1.5" aria-current={item.current ? 'page' : undefined}>
                    {item.icon && <span>{item.icon}</span>}
                    {item.label}
                  </a>
                ) : (
                  <span className="breadcrumb-current flex items-center gap-1.5" aria-current="page">
                    {item.icon && <span>{item.icon}</span>}
                    {item.label}
                  </span>
                )}
              </li>
            ))}
          </ol>
        </nav>
      )}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div className="flex items-center gap-3">
          {icon && <div className="record-page-icon">{icon}</div>}
          <div>
            <h1 className="record-page-title text-2xl font-bold text-gray-900">{title}</h1>
            {description && <p className="text-gray-600 mt-1">{description}</p>}
          </div>
        </div>
        {actions && <div className="flex items-center gap-2">{actions}</div>}
      </div>
    </div>
  );
}

export function CardGrid({ children, columns = 3, gap = 6, className = '' }: { children: React.ReactNode; columns?: 1 | 2 | 3 | 4; gap?: number; className?: string }) {
  const columnClasses = {
    1: 'grid-cols-1',
    2: 'grid-cols-1 md:grid-cols-2',
    3: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-3',
    4: 'grid-cols-1 md:grid-cols-2 lg:grid-cols-4',
  };

  return (
    <div className={`grid ${columnClasses[columns]} ${className}`} style={{ gap: gap * 4 }}>
      {children}
    </div>
  );
}

// ============================================
// UTILITY EXPORTS
// ============================================

export { api } from './api';
export type { Column, TableProps, ModalProps, FormFieldProps, BadgeProps, PaginationProps, SearchFilterProps, FilterConfig, Toast, ToastContainerProps, ConfirmDialogProps, DropdownProps, DropdownItem, TabsProps, TabConfig, AvatarProps, AvatarGroupProps, StatCardProps, EmptyStateProps, BreadcrumbItem, BreadcrumbsProps, KPIMetric, DataTableToolbarProps };

export function Button({ variant = 'primary', size = 'md', children, className = '', loading = false, disabled, ...props }: {
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger' | 'outline';
  size?: 'sm' | 'md' | 'lg';
  children: React.ReactNode;
  className?: string;
  loading?: boolean;
  disabled?: boolean;
} & React.ButtonHTMLAttributes<HTMLButtonElement>) {
  const variantClasses = {
    primary: 'btn-primary',
    secondary: 'btn-secondary',
    ghost: 'btn-ghost',
    danger: 'btn-danger',
    outline: 'btn-outline',
  };

  const sizeClasses = {
    sm: 'btn-sm',
    md: '',
    lg: 'btn-lg',
  };

  return (
    <button
      className={`btn ${variantClasses[variant]} ${sizeClasses[size]} ${className}`}
      disabled={disabled || loading}
      {...props}
    >
      {loading ? <Loader2 className="loading-spinner w-4 h-4 mr-2" /> : null}
      {children}
    </button>
  );
}

export function KanbanBoard({ columns, onDragEnd, renderCard, onAddCard }: {
  columns: any[];
  onDragEnd: (item: any, sourceColumn: string, targetColumn: string) => Promise<void>;
  renderCard?: (item: any) => React.ReactNode;
  onAddCard?: (columnId: string) => void;
}) {
  const [dragging, setDragging] = useState<{ item: any; source: string } | null>(null);
  const [dropTarget, setDropTarget] = useState<string | null>(null);
  const [pending, setPending] = useState<string | null>(null);
  const [moveError, setMoveError] = useState('');
  const move = async (item: any, source: string, target: string) => {
    if (source === target || pending) return;
    setDragging(null); setDropTarget(null); setPending(item.id); setMoveError('');
    try { await onDragEnd(item, source, target); }
    catch { setMoveError('Could not move the record. Please try again.'); }
    finally { setPending(null); }
  };
  return <div>
    {moveError && <p className="form-error" role="alert">{moveError}</p>}
    <div className="kanban-board">{columns.map(column => <section key={column.id} className={'kanban-column ' + (dropTarget === column.id ? 'kanban-drop-active' : '')} aria-label={column.title}
      onDragOver={event => { if (dragging && !pending) { event.preventDefault(); event.dataTransfer.dropEffect = 'move'; setDropTarget(column.id); } }}
      onDragLeave={event => { if (!event.currentTarget.contains(event.relatedTarget as Node)) setDropTarget(null); }}
      onDrop={event => { event.preventDefault(); if (dragging) void move(dragging.item, dragging.source, column.id); }}>
      <div className="kanban-column-header"><div className="kanban-column-title"><h3 className="font-medium">{column.title}</h3><span className="kanban-column-count">{column.items?.length || 0}</span></div></div>
      <div className="kanban-cards">{column.items?.map((item: any) => <div key={item.id} className="kanban-card mb-2" draggable={!pending}
        onDragStart={event => { event.dataTransfer.effectAllowed = 'move'; event.dataTransfer.setData('text/plain', item.id); setDragging({item,source:column.id}); }}
        onDragEnd={() => { setDragging(null); setDropTarget(null); }} aria-busy={pending === item.id}>
        {renderCard ? renderCard(item) : <div className="p-3"><p className="font-medium">{item.title || item.name}</p><p className="text-sm text-gray-500">{item.description || ''}</p></div>}
        <select className="kanban-move" value="" aria-label={'Move ' + (item.title || item.name || 'record') + ' to column'} disabled={!!pending} onClick={event => event.stopPropagation()} onChange={event => { void move(item, column.id, event.target.value); }}><option value="">{pending === item.id ? 'Moving...' : 'Move to...'}</option>{columns.filter(target => target.id !== column.id).map(target => <option key={target.id} value={target.id}>{target.title}</option>)}</select>
      </div>)}{onAddCard && <button className="kanban-add-card btn btn-ghost btn-sm w-full justify-start" onClick={() => onAddCard(column.id)}><Plus size={14} />Add record</button>}</div>
    </section>)}</div>
  </div>;
}

export function ActivityFeed({ activities }: { activities: any[] }) {
  return (
    <div className="activity-feed">
      {activities.map((activity, index) => (
        <div key={index} className="activity-item">
          <div className="activity-dot bg-orange-600" />
          <div className="activity-content">
            <div className="activity-header">
              <span className="activity-user">{activity.user_name}</span>
              <span className="activity-action">{activity.action}</span>
              <a href={activity.target_url} className="activity-target">{activity.target}</a>
              <span className="activity-time">{activity.time}</span>
            </div>
            <p className="activity-description">{activity.description}</p>
          </div>
        </div>
      ))}
    </div>
  );
}
