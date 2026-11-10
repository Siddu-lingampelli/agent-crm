// API Client for TORQ Backend
const API_BASE = '/api/v1';

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = 'ApiError';
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const relativePath = path.startsWith(API_BASE + '/') ? path.slice(API_BASE.length) : path;
  const headers = new Headers(options.headers);
  if (!headers.has('Content-Type')) headers.set('Content-Type', 'application/json');
  const response = await fetch(API_BASE + relativePath, {
    ...options,
    headers,
    credentials: 'include',
  });

  if (!response.ok) {
    const error = await response.json().catch(() => ({ message: response.statusText }));
    throw new ApiError(response.status, error.error?.message || error.message || response.statusText);
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return response.json();
}

export const api = {
  // Generic HTTP methods for admin pages
  get: <T>(path: string, options?: { params?: Record<string, any> }) => {
    const searchParams = new URLSearchParams();
    if (options?.params) {
      Object.entries(options.params).forEach(([key, value]) => {
        if (value !== undefined && value !== null) searchParams.set(key, String(value));
      });
    }
    const query = searchParams.toString();
    return request<T>(`${path}${query ? `?${query}` : ''}`);
  },
  post: <T>(path: string, data?: any) => request<T>(path, { method: 'POST', body: JSON.stringify(data) }),
  patch: <T>(path: string, data?: any) => request<T>(path, { method: 'PATCH', body: JSON.stringify(data) }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),

  // Dashboard
  dashboard: {
    getMetrics: (day?: string) => request<DashboardMetrics>(`/dashboard/metrics${day ? `?day=${encodeURIComponent(day)}` : ""}`),
  },

  // Companies
  companies: {
    list: (params?: { q?: string; status?: string; is_client?: boolean; limit?: number; offset?: number; sort?: string; order?: string }) => {
      const searchParams = new URLSearchParams();
      if (params) {
        Object.entries(params).forEach(([key, value]) => {
          if (value !== undefined && value !== null) searchParams.set(key, String(value));
        });
      }
      return request<{ items: Company[]; total: number; limit: number; offset: number }>(`/companies?${searchParams}`);
    },
    get: (id: string) => request<Company>(`/companies/${id}`),
    create: (data: CompanyCreateInput) => request<Company>(`/companies/create`, { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: CompanyUpdateInput) => request<Company>(`/companies/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    archive: (id: string) => request<void>(`/companies/${id}`, { method: 'DELETE' }),
  },

  // Contacts
  contacts: {
    list: (params?: { q?: string; status?: string; company_id?: string; limit?: number; offset?: number }) => {
      const searchParams = new URLSearchParams();
      if (params) {
        Object.entries(params).forEach(([key, value]) => {
          if (value !== undefined && value !== null) searchParams.set(key, String(value));
        });
      }
      return request<{ items: Contact[]; total: number; limit: number; offset: number }>(`/contacts?${searchParams}`);
    },
    get: (id: string) => request<Contact>(`/contacts/${id}`),
    create: (data: ContactCreateInput) => request<Contact>(`/contacts/create`, { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: ContactUpdateInput) => request<Contact>(`/contacts/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    archive: (id: string) => request<void>(`/contacts/${id}`, { method: 'DELETE' }),
  },

  // Leads
  leads: {
    list: (params?: { q?: string; status?: string; source?: string; priority?: string; sort?: string; order?: string; limit?: number; offset?: number }) => {
      const searchParams = new URLSearchParams();
      if (params) {
        Object.entries(params).forEach(([key, value]) => {
          if (value !== undefined && value !== null) searchParams.set(key, String(value));
        });
      }
      return request<{ items: Lead[]; total: number; limit: number; offset: number }>(`/leads?${searchParams}`);
    },
    get: (id: string) => request<Lead>(`/leads/${id}`),
    create: (data: LeadCreateInput) => request<Lead>(`/leads/create`, { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: LeadUpdateInput) => request<Lead>(`/leads/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    archive: (id: string) => request<void>(`/leads/${id}`, { method: 'DELETE' }),
    convert: (id: string, data: LeadConvertInput) => request<Lead>(`/leads/${id}/convert`, { method: 'POST', body: JSON.stringify(data) }),
  },

  // Deals
  deals: {
    list: (params?: { q?: string; category?: string; close_from?: string; close_before?: string; inactive_before?: string; stage_id?: string; pipeline_id?: string; limit?: number; offset?: number; sort?: string; order?: string }) => {
      const searchParams = new URLSearchParams();
      if (params) {
        Object.entries(params).forEach(([key, value]) => {
          if (value !== undefined && value !== null) searchParams.set(key, String(value));
        });
      }
      return request<{ items: Deal[]; total: number; limit: number; offset: number }>(`/deals?${searchParams}`);
    },
    get: (id: string) => request<Deal>(`/deals/${id}`),
    create: (data: DealCreateInput) => request<Deal>(`/deals/create`, { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: DealUpdateInput) => request<Deal>(`/deals/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    archive: (id: string) => request<void>(`/deals/${id}`, { method: 'DELETE' }),
  },

  // Tasks
  tasks: {
    list: (params?: { q?: string; status?: string; priority?: string; project_id?: string; assignee_id?: string; active_only?: boolean; unscheduled?: boolean; due_after?: string; due_before?: string; due_on?: string; sort?: string; order?: string; limit?: number; offset?: number }) => {
      const searchParams = new URLSearchParams();
      if (params) {
        Object.entries(params).forEach(([key, value]) => {
          if (value !== undefined && value !== null) searchParams.set(key, String(value));
        });
      }
      return request<{ items: Task[]; total: number; limit: number; offset: number }>(`/tasks?${searchParams}`);
    },
    get: (id: string) => request<Task>(`/tasks/${id}`),
    create: (data: TaskCreateInput) => request<Task>(`/tasks/create`, { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: TaskUpdateInput) => request<Task>(`/tasks/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    archive: (id: string) => request<void>(`/tasks/${id}`, { method: 'DELETE' }),
    complete: (id: string) => request<Task>(`/tasks/${id}/complete`, { method: 'POST' }),
  },

  // Projects
  projects: {
    list: (params?: { q?: string; status?: string; priority?: string; company_id?: string; owner_id?: string; sort?: string; order?: string; limit?: number; offset?: number }) => {
      const searchParams = new URLSearchParams();
      if (params) {
        Object.entries(params).forEach(([key, value]) => {
          if (value !== undefined && value !== null) searchParams.set(key, String(value));
        });
      }
      return request<{ items: Project[]; total: number; limit: number; offset: number }>(`/projects?${searchParams}`);
    },
    get: (id: string) => request<Project>(`/projects/${id}`),
    create: (data: ProjectCreateInput) => request<Project>(`/projects/create`, { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: ProjectUpdateInput) => request<Project>(`/projects/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    archive: (id: string) => request<void>(`/projects/${id}`, { method: 'DELETE' }),
  },

  // Search
  search: (query: string, limit = 10) =>
    request<{ results: SearchResult[]; total: number; query: string; limit: number }>(`/search?q=${encodeURIComponent(query)}&limit=${limit}`),

  // Emails
  emails: {
    list: (params?: { q?: string; status?: string; related_type?: string; related_id?: string; limit?: number; offset?: number; sort?: string; order?: string }) => {
      const searchParams = new URLSearchParams();
      if (params) {
        Object.entries(params).forEach(([key, value]) => {
          if (value !== undefined && value !== null) searchParams.set(key, String(value));
        });
      }
      return request<{ items: Email[]; total: number; limit: number; offset: number }>(`/emails?${searchParams}`);
    },
    get: (id: string) => request<Email>(`/emails/${id}`),
    create: (data: EmailCreateInput) => request<Email>(`/emails/create`, { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: EmailUpdateInput) => request<Email>(`/emails/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    archive: (id: string) => request<void>(`/emails/${id}`, { method: 'DELETE' }),
    approve: (id: string) => request<Email>(`/emails/${id}/approve`, { method: 'POST' }),
    send: (id: string) => request<Email>(`/emails/${id}/send`, { method: 'POST' }),
    retryFailed: () => request<{ retried: number }>(`/emails/retry`, { method: 'POST' }),
  },

  // Email Templates
  emailTemplates: {
    list: () => request<{ items: EmailTemplate[] }>(`/email-templates`),
    get: (id: string) => request<EmailTemplate>(`/email-templates/${id}`),
    create: (data: EmailTemplateCreateInput) => request<EmailTemplate>(`/email-templates/create`, { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: Partial<EmailTemplateCreateInput>) => request<EmailTemplate>(`/email-templates/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    archive: (id: string) => request<void>(`/email-templates/${id}`, { method: 'DELETE' }),
    apply: (id: string, data: EmailApplyTemplateInput) => request<Email>(`/email-templates/${id}/apply`, { method: 'POST', body: JSON.stringify(data) }),
  },

  // Calendar
  calendar: {
    list: (params?: { q?: string; status?: string; range_start?: string; range_end?: string; start_from?: string; start_to?: string; limit?: number; offset?: number; sort?: string; order?: string }) => {
      const searchParams = new URLSearchParams();
      if (params) {
        Object.entries(params).forEach(([key, value]) => {
          if (value !== undefined && value !== null) searchParams.set(key, String(value));
        });
      }
      return request<{ items: CalendarEvent[]; total: number; limit: number; offset: number }>(`/calendar?${searchParams}`);
    },
    get: (id: string) => request<CalendarEvent>(`/calendar/${id}`),
    create: (data: CalendarCreateInput) => request<CalendarEvent>(`/calendar/create`, { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: CalendarUpdateInput) => request<CalendarEvent>(`/calendar/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    archive: (id: string) => request<void>(`/calendar/${id}`, { method: 'DELETE' }),
  },

  // Workflows
  workflows: {
    list: (params?: { q?: string; status?: string; limit?: number; offset?: number; sort?: string; order?: string }) => {
      const searchParams = new URLSearchParams();
      if (params) {
        Object.entries(params).forEach(([key, value]) => {
          if (value !== undefined && value !== null) searchParams.set(key, String(value));
        });
      }
      return request<{ items: Workflow[]; total: number; limit: number; offset: number }>(`/workflows?${searchParams}`);
    },
    get: (id: string) => request<Workflow>(`/workflows/${id}`),
    create: (data: WorkflowCreateInput) => request<Workflow>(`/workflows/create`, { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: WorkflowUpdateInput) => request<Workflow>(`/workflows/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    archive: (id: string) => request<void>(`/workflows/${id}`, { method: 'DELETE' }),
    execute: (id: string, triggerData?: Record<string, any>) => request<WorkflowExecution>(`/workflows/${id}/execute`, { method: 'POST', body: JSON.stringify(triggerData || {}) }),
    listExecutions: (params?: { workflow_id?: string; limit?: number; offset?: number }) => {
      const searchParams = new URLSearchParams();
      if (params) {
        Object.entries(params).forEach(([key, value]) => {
          if (value !== undefined && value !== null) searchParams.set(key, String(value));
        });
      }
      return request<WorkflowExecution[]>(`/workflow-executions?${searchParams}`);
    },
    getExecution: (id: string) => request<WorkflowExecution>(`/workflow-executions/${id}`),
  },

  // Integrations
  integrations: {
    list: (params?: { q?: string; provider?: string; status?: string; limit?: number; offset?: number; sort?: string; order?: string }) => {
      const searchParams = new URLSearchParams();
      if (params) {
        Object.entries(params).forEach(([key, value]) => {
          if (value !== undefined && value !== null) searchParams.set(key, String(value));
        });
      }
      return request<{ items: Integration[]; total: number; limit: number; offset: number }>(`/integrations?${searchParams}`);
    },
    get: (id: string) => request<Integration>(`/integrations/${id}`),
    create: (data: IntegrationCreateInput) => request<Integration>(`/integrations/create`, { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: IntegrationUpdateInput) => request<Integration>(`/integrations/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    archive: (id: string) => request<void>(`/integrations/${id}`, { method: 'DELETE' }),
    updateCredentials: (id: string, credentials: string) => request<void>(`/integrations/${id}/credentials`, { method: 'POST', body: JSON.stringify({ credentials }) }),
    sync: (id: string) => request<void>(`/integrations/${id}/sync`, { method: 'POST' }),
    test: (id: string) => request<void>(`/integrations/${id}/test`, { method: 'POST' }),
    listLogs: (id: string, params?: { limit?: number }) => {
      const searchParams = new URLSearchParams();
      if (params) {
        Object.entries(params).forEach(([key, value]) => {
          if (value !== undefined && value !== null) searchParams.set(key, String(value));
        });
      }
      return request<{ items: SyncLog[] }>(`/integrations/${id}/logs?${searchParams}`);
    },
  },

  // Webhooks
  webhooks: {
    list: (params?: { status?: string; limit?: number; offset?: number; sort?: string; order?: string }) => {
      const searchParams = new URLSearchParams();
      if (params) {
        Object.entries(params).forEach(([key, value]) => {
          if (value !== undefined && value !== null) searchParams.set(key, String(value));
        });
      }
      return request<{ items: Webhook[]; total: number; limit: number; offset: number }>(`/webhooks?${searchParams}`);
    },
    get: (id: string) => request<Webhook>(`/webhooks/${id}`),
    create: (data: WebhookCreateInput) => request<Webhook>(`/webhooks/create`, { method: 'POST', body: JSON.stringify(data) }),
    update: (id: string, data: WebhookUpdateInput) => request<Webhook>(`/webhooks/${id}`, { method: 'PATCH', body: JSON.stringify(data) }),
    archive: (id: string) => request<void>(`/webhooks/${id}`, { method: 'DELETE' }),
    test: (id: string) => request<void>(`/webhooks/${id}/test`, { method: 'POST' }),
  },
};

// Types
export interface Company {
  id: string;
  workspace_id: string;
  name: string;
  domain: string;
  industry: string;
  size_band: string;
  website: string;
  phone: string;
  email: string;
  address: string;
  is_client: boolean;
  status: string;
  owner_id: string;
  tags: string[] | null;
  custom_fields: Record<string, any> | null;
  notes: string;
  created_at: string;
  updated_at: string;
  archived_at: string | null;
}

export interface CompanyCreateInput {
  name: string;
  domain: string;
  industry: string;
  size_band?: string;
  website?: string;
  phone?: string;
  email?: string;
  address?: string;
  is_client?: boolean;
  status?: string;
  owner_id?: string;
  tags?: string[];
  custom_fields?: Record<string, any>;
  notes?: string;
}

export interface CompanyUpdateInput {
  name?: string;
  domain?: string;
  industry?: string;
  size_band?: string;
  website?: string;
  phone?: string;
  email?: string;
  address?: string;
  is_client?: boolean;
  status?: string;
  owner_id?: string;
  tags?: string[];
  custom_fields?: Record<string, any>;
  notes?: string;
}

export interface Contact {
  id: string;
  workspace_id: string;
  first_name: string;
  last_name: string;
  email: string;
  phone: string;
  job_title: string;
  company_id: string;
  owner_id: string;
  status: string;
  tags: string[] | null;
  custom_fields: Record<string, any> | null;
  notes: string;
  created_at: string;
  updated_at: string;
  archived_at: string | null;
}

export interface ContactCreateInput {
  first_name: string;
  last_name: string;
  email: string;
  phone?: string;
  job_title?: string;
  company_id?: string;
  owner_id?: string;
  status?: string;
  tags?: string[];
  custom_fields?: Record<string, any>;
  notes?: string;
}

export interface ContactUpdateInput {
  first_name?: string;
  last_name?: string;
  email?: string;
  phone?: string;
  job_title?: string;
  company_id?: string;
  owner_id?: string;
  status?: string;
  tags?: string[];
  custom_fields?: Record<string, any>;
  notes?: string;
}

export interface Lead {
  id: string;
  workspace_id: string;
  display_name: string;
  first_name: string;
  last_name: string;
  email: string;
  phone: string;
  company_name: string;
  company_id: string;
  job_title: string;
  status: string;
  source: string;
  owner_id: string;
  priority: string;
  estimated_value: number | null;
  currency: string;
  lead_score: number | null;
  expected_close: string;
  follow_up_date: string;
  tags: string[] | null;
  custom_fields: Record<string, any> | null;
  description: string;
  qualification: string;
  created_at: string;
  updated_at: string;
  archived_at: string | null;
}

export interface LeadCreateInput {
  display_name: string;
  first_name: string;
  last_name: string;
  email: string;
  phone?: string;
  company_name?: string;
  company_id?: string;
  job_title?: string;
  status?: string;
  source?: string;
  owner_id?: string;
  priority?: string;
  estimated_value?: number;
  currency?: string;
  lead_score?: number;
  expected_close?: string;
  follow_up_date?: string;
  tags?: string[];
  custom_fields?: Record<string, any>;
  description?: string;
  qualification?: string;
}

export interface LeadUpdateInput {
  display_name?: string;
  first_name?: string;
  last_name?: string;
  email?: string;
  phone?: string;
  company_name?: string;
  company_id?: string;
  job_title?: string;
  status?: string;
  source?: string;
  owner_id?: string;
  priority?: string;
  estimated_value?: number;
  currency?: string;
  lead_score?: number;
  expected_close?: string;
  follow_up_date?: string;
  tags?: string[];
  custom_fields?: Record<string, any>;
  description?: string;
  qualification?: string;
}

export interface LeadConvertInput {
  contact_id?: string;
  company_id?: string;
  deal_name: string;
  pipeline_id: string;
  stage_id: string;
  amount: number;
  currency: string;
  expected_close: string;
}

export interface PipelineChoice {
  id: string;
  name: string;
  is_default: boolean;
  stages: { id: string; name: string; position: number; probability: number; category: string }[];
}

export interface Deal {
  id: string;
  workspace_id: string;
  name: string;
  pipeline_id: string;
  stage_id: string;
  amount: number | null;
  currency: string;
  probability: number;
  expected_close: string;
  company_id: string;
  primary_contact_id: string;
  lead_id: string;
  owner_id: string;
  source: string;
  description: string;
  lost_reason: string;
  custom_fields: Record<string, any> | null;
  created_at: string;
  updated_at: string;
  archived_at: string | null;
}

export interface DealCreateInput {
  name: string;
  pipeline_id: string;
  stage_id: string;
  amount?: number;
  currency?: string;
  probability?: number;
  expected_close: string;
  company_id?: string;
  primary_contact_id?: string;
  lead_id?: string;
  owner_id?: string;
  source?: string;
  description?: string;
  lost_reason?: string;
  custom_fields?: Record<string, any>;
}

export interface DealUpdateInput {
  name?: string;
  pipeline_id?: string;
  stage_id?: string;
  amount?: number;
  currency?: string;
  probability?: number;
  expected_close?: string;
  company_id?: string;
  primary_contact_id?: string;
  lead_id?: string;
  owner_id?: string;
  source?: string;
  description?: string;
  lost_reason?: string;
  custom_fields?: Record<string, any>;
}

export interface Task {
  id: string;
  workspace_id: string;
  title: string;
  description: string;
  status: string;
  priority: string;
  due_date: string;
  assignee_id: string;
  related_type: string;
  related_id: string;
  project_id: string;
  created_by: string;
  custom_fields: Record<string, any> | null;
  completed_at: string | null;
  created_at: string;
  updated_at: string;
  archived_at: string | null;
}

export interface TaskCreateInput {
  title: string;
  description?: string;
  status?: string;
  priority?: string;
  due_date?: string;
  assignee_id?: string;
  related_type?: string;
  related_id?: string;
  project_id?: string;
  created_by?: string;
  custom_fields?: Record<string, any>;
}

export interface TaskUpdateInput {
  title?: string;
  description?: string;
  status?: string;
  priority?: string;
  due_date?: string;
  assignee_id?: string;
  related_type?: string;
  related_id?: string;
  project_id?: string;
  custom_fields?: Record<string, any>;
}

export interface Project {
  id: string;
  workspace_id: string;
  name: string;
  description: string;
  status: string;
  priority: string;
  owner_id: string;
  company_id: string;
  start_date: string;
  end_date: string;
  progress: number;
  custom_fields: Record<string, any> | null;
  created_at: string;
  updated_at: string;
  archived_at: string | null;
}

export interface ProjectCreateInput {
  name: string;
  description?: string;
  status?: string;
  priority?: string;
  owner_id?: string;
  company_id?: string;
  start_date?: string;
  end_date?: string;
  progress?: number;
  custom_fields?: Record<string, any>;
}

export interface ProjectUpdateInput {
  name?: string;
  description?: string;
  status?: string;
  priority?: string;
  owner_id?: string;
  company_id?: string;
  start_date?: string;
  end_date?: string;
  progress?: number;
  custom_fields?: Record<string, any>;
}

export interface SearchResult {
  entity_type: string;
  entity_id: string;
  title: string;
  subtitle: string;
  url: string;
  score: number;
  metadata: Record<string, any> | null;
}

// Email
export interface Email {
  id: string;
  workspace_id: string;
  to: string;
  cc: string;
  bcc: string;
  subject: string;
  body: string;
  body_html: string;
  status: 'draft' | 'queued' | 'sending' | 'sent' | 'failed';
  provider: string;
  provider_message_id: string;
  related_type: string;
  related_id: string;
  created_by: string;
  approved_by: string;
  approved_at: string | null;
  sent_at: string | null;
  failed_at: string | null;
  error_message: string;
  retry_count: number;
  created_at: string;
  updated_at: string;
}

export interface EmailCreateInput {
  to: string;
  cc?: string;
  bcc?: string;
  subject: string;
  body: string;
  body_html?: string;
  related_type?: string;
  related_id?: string;
}

export interface EmailUpdateInput {
  to?: string;
  cc?: string;
  bcc?: string;
  subject?: string;
  body?: string;
  body_html?: string;
}

export interface EmailTemplate {
  id: string;
  workspace_id: string;
  name: string;
  subject: string;
  body: string;
  body_html: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface EmailTemplateCreateInput {
  name: string;
  subject: string;
  body: string;
  body_html?: string;
}

export interface EmailApplyTemplateInput {
  to: string;
  cc?: string;
  bcc?: string;
  related_type?: string;
  related_id?: string;
}

// Calendar
export interface CalendarEvent {
  id: string;
  workspace_id: string;
  title: string;
  description: string;
  start_time: string;
  end_time: string;
  all_day: boolean;
  location: string;
  meeting_url: string;
  organizer_id: string;
  attendee_ids: string[];
  related_type: string;
  related_id: string;
  status: 'confirmed' | 'tentative' | 'cancelled';
  created_at: string;
  updated_at: string;
}

export interface CalendarCreateInput {
  title: string;
  description?: string;
  start_time: string;
  end_time: string;
  all_day?: boolean;
  location?: string;
  meeting_url?: string;
  attendee_ids?: string[];
  related_type?: string;
  related_id?: string;
  status?: string;
}

export interface CalendarUpdateInput {
  title?: string;
  description?: string;
  start_time?: string;
  end_time?: string;
  all_day?: boolean;
  location?: string;
  meeting_url?: string;
  attendee_ids?: string[];
  status?: string;
}

// Workflows
export interface Workflow {
  id: string;
  workspace_id: string;
  name: string;
  description: string;
  trigger_type: 'manual' | 'schedule' | 'event' | 'webhook';
  trigger_config: Record<string, any>;
  steps: WorkflowStep[];
  status: 'active' | 'inactive' | 'draft';
  created_by: string;
  updated_by: string;
  last_run_at: string | null;
  next_run_at: string | null;
  run_count: number;
  success_count: number;
  failure_count: number;
  created_at: string;
  updated_at: string;
}

export interface WorkflowStep {
  id: string;
  name: string;
  type: 'action' | 'condition' | 'delay' | 'webhook' | 'mcp_call';
  config: Record<string, any>;
  order: number;
}

export interface WorkflowCreateInput {
  name: string;
  description?: string;
  trigger_type: 'manual' | 'schedule' | 'event' | 'webhook';
  trigger_config?: Record<string, any>;
  steps: WorkflowStep[];
  status?: 'active' | 'inactive' | 'draft';
}

export interface WorkflowUpdateInput {
  name?: string;
  description?: string;
  trigger_type?: 'manual' | 'schedule' | 'event' | 'webhook';
  trigger_config?: Record<string, any>;
  steps?: WorkflowStep[];
  status?: 'active' | 'inactive' | 'draft';
}

export interface WorkflowExecution {
  id: string;
  workflow_id: string;
  workspace_id: string;
  status: 'running' | 'completed' | 'failed' | 'cancelled';
  trigger_data: Record<string, any>;
  steps_run: number;
  steps_total: number;
  error_message: string;
  started_at: string;
  completed_at: string | null;
  created_at: string;
}

// Integrations
export interface Integration {
  id: string;
  workspace_id: string;
  name: string;
  provider: string;
  config: Record<string, any>;
  credentials: string;
  status: 'connected' | 'disconnected' | 'error';
  last_sync_at: string | null;
  last_error: string;
  enabled: boolean;
  created_at: string;
  updated_at: string;
}

export interface IntegrationCreateInput {
  name: string;
  provider: string;
  config?: Record<string, any>;
  enabled?: boolean;
}

export interface IntegrationUpdateInput {
  name?: string;
  config?: Record<string, any>;
  enabled?: boolean;
  status?: string;
}

// Webhooks
export interface Webhook {
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

export interface WebhookCreateInput {
  name: string;
  url: string;
  events: string[];
  secret: string;
}

export interface WebhookUpdateInput {
  name?: string;
  url?: string;
  events?: string[];
  secret?: string;
  status?: string;
}

// Dashboard
export interface DashboardMetrics {
  insights: DashboardInsights;
  open_pipeline_value: number;
  weighted_pipeline_value: number;
  pipeline_stages: PipelineStageMetric[];
  new_leads_count: number;
  new_leads_this_week: number;
  tasks_due_count: number;
  tasks_overdue_count: number;
  win_rate: number;
  avg_deal_size: number;
  sales_cycle_days: number;
  lead_conversion_rate: number;
  recent_activity: ActivityItem[];
}

export interface PipelineStageMetric {
  stage_id: string;
  stage_name: string;
  count: number;
  value: number;
  color: string;
  position: number;
}

export interface ActivityItem {
  id: string;
  user_name: string;
  user_avatar: string;
  action: string;
  target: string;
  target_url: string;
  time: string;
  description: string;
}

// Sync Log
export interface SyncLog {
  id: string;
  integration_id: string;
  status: 'success' | 'failed' | 'partial';
  records_processed: number;
  records_failed: number;
  started_at: string;
  completed_at: string | null;
  error_message: string | null;
}

export interface DashboardCurrency {currency:string;open_value:number;weighted_value:number;won_this_month:number;won_last_month:number;average_won:number;open_deals:number;won_deals:number;lost_deals:number}
export interface DashboardStage {id:string;name:string;category:string;position:number;currency:string;count:number;value:number}
export interface DashboardRevenue {month:string;currency:string;value:number;count:number}
export interface DashboardInsights {day:string;pipeline_id:string;pipeline_name:string;open_deals:number;won_deals:number;lost_deals:number;past_close_deals:number;inactive_deals:number;total_leads:number;converted_leads:number;companies:number;contacts:number;active_tasks:number;unscheduled_tasks:number;currencies:DashboardCurrency[];stages:DashboardStage[];revenue:DashboardRevenue[]}
