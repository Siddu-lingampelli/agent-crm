import { createBrowserRouter, RouterProvider, Navigate, useRouteError, useLocation } from 'react-router-dom'
import { lazy, Suspense, useState, useEffect } from 'react'
import { api } from './api'
import { LayoutDashboard, Users, DollarSign, Contact, Building2, CheckSquare, FolderKanban, Mail, Calendar as CalendarIcon, Bot, GitBranch, Plug, Key, Database, ClipboardList, Monitor, Settings, Zap, Loader2 } from 'lucide-react'
import './index.css'
import { WorkspaceShell } from './WorkspaceShell'
import { AdminArea } from './AdminArea'
import { isAdminPath } from './admin-navigation'
const AdministrationPage = lazy(() => import('./AdministrationPage').then(module => ({ default: module.AdministrationPage })))
const CompaniesPage = lazy(() => import('./CompaniesPage').then(module => ({ default: module.CompaniesPage })))
const ContactsPage = lazy(() => import('./ContactsPage').then(module => ({ default: module.ContactsPage })))
const LeadsPage = lazy(() => import('./LeadsPage').then(module => ({ default: module.LeadsPage })))
const DealsPage = lazy(() => import('./DealsPage').then(module => ({ default: module.DealsPage })))
const TasksPage = lazy(() => import('./TasksPage').then(module => ({ default: module.TasksPage })))
const ProjectsPage = lazy(() => import('./ProjectsPage').then(module => ({ default: module.ProjectsPage })))
const AgentConsolePage = lazy(() => import('./AgentConsolePage').then(module => ({ default: module.AgentConsolePage })))
const WorkflowsPage = lazy(() => import('./WorkflowsPage').then(module => ({ default: module.WorkflowsPage })))
const IntegrationsPage = lazy(() => import('./IntegrationsPage').then(module => ({ default: module.IntegrationsPage })))
const AgentsPage = lazy(() => import('./AgentsPage').then(module => ({ default: module.AgentsPage })))
const SchemaPage = lazy(() => import('./SchemaPage').then(module => ({ default: module.SchemaPage })))
const AuditPage = lazy(() => import('./AuditPage').then(module => ({ default: module.AuditPage })))
const SystemPage = lazy(() => import('./SystemPage').then(module => ({ default: module.SystemPage })))
const SettingsPage = lazy(() => import('./SettingsPage').then(module => ({ default: module.SettingsPage })))
const CalendarPage = lazy(() => import('./CalendarPage').then(module => ({ default: module.CalendarPage })))
const EmailsPage = lazy(() => import('./EmailsPage').then(module => ({ default: module.EmailsPage })))
const WebhooksPage = lazy(() => import('./WebhooksPage').then(module => ({ default: module.WebhooksPage })))
const LoginPage = lazy(() => import('./LoginPage').then(module => ({ default: module.LoginPage })))
const HomePage = lazy(() => import('./HomePage').then(module => ({ default: module.HomePage })))
const DashboardPage = lazy(() => import('./DashboardPage').then(module => ({ default: module.DashboardPage })))


const navigation: Array<{ name: string; href: string; icon: React.ReactNode }> = [
  { name: 'Dashboard', href: '/dashboard', icon: <LayoutDashboard className="w-5 h-5" /> },
  { name: 'Leads', href: '/leads', icon: <Users className="w-5 h-5" /> },
  { name: 'Deals', href: '/deals', icon: <DollarSign className="w-5 h-5" /> },
  { name: 'Contacts', href: '/contacts', icon: <Contact className="w-5 h-5" /> },
  { name: 'Companies', href: '/companies', icon: <Building2 className="w-5 h-5" /> },
  { name: 'Tasks', href: '/tasks', icon: <CheckSquare className="w-5 h-5" /> },
  { name: 'Projects', href: '/projects', icon: <FolderKanban className="w-5 h-5" /> },
  { name: 'Inbox', href: '/inbox', icon: <Mail className="w-5 h-5" /> },
  { name: 'Calendar', href: '/calendar', icon: <CalendarIcon className="w-5 h-5" /> },
]

const adminNavigation: Array<{ name: string; href: string; icon: React.ReactNode }> = [
  { name: 'Administration', href: '/admin', icon: <Settings className="w-5 h-5" /> },
  { name: 'Agent Console', href: '/agent-console', icon: <Bot className="w-5 h-5" /> },
  { name: 'Workflows', href: '/workflows', icon: <GitBranch className="w-5 h-5" /> },
  { name: 'Integrations', href: '/integrations', icon: <Plug className="w-5 h-5" /> },
  { name: 'Webhooks', href: '/webhooks', icon: <Zap className="w-5 h-5" /> },
  { name: 'Agents & API', href: '/agents', icon: <Key className="w-5 h-5" /> },
  { name: 'Data Model', href: '/schema', icon: <Database className="w-5 h-5" /> },
  { name: 'Audit Log', href: '/audit', icon: <ClipboardList className="w-5 h-5" /> },
  { name: 'System & Health', href: '/system', icon: <Monitor className="w-5 h-5" /> },
  { name: 'Settings', href: '/settings', icon: <Settings className="w-5 h-5" /> },
]

function LegacyActivityRedirect() {
  const location = useLocation();
  return <Navigate to={{ pathname: '/audit', search: location.search, hash: location.hash }} replace />;
}

function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { pathname } = useLocation()
  return <WorkspaceShell navigation={navigation} adminNavigation={adminNavigation}><Suspense fallback={<div className="ws-route-loading" role="status"><Loader2 className="loading-spinner" size={20} />Loading workspace...</div>}>{isAdminPath(pathname) ? <AdminArea>{children}</AdminArea> : children}</Suspense></WorkspaceShell>
}

// Protected route wrapper - checks if user is authenticated
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  const [authChecked, setAuthChecked] = useState(false);
  const [isAuthenticated, setIsAuthenticated] = useState(false);

  useEffect(() => {
    // Check authentication status by trying to fetch a protected endpoint
    async function checkAuth() {
      try {
        await api.get('/auth/me');
        setIsAuthenticated(true);
      } catch {
        setIsAuthenticated(false);
      } finally {
        setAuthChecked(true);
      }
    }
    checkAuth();
  }, []);

  if (!authChecked) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-surface-bg">
        <Loader2 className="loading-spinner w-8 h-8 text-black" />
      </div>
    );
  }

  if (!isAuthenticated) {
    return <Navigate to="/login" replace state={{ from: window.location.pathname + window.location.search }} />;
  }

  return <>{children}</>;
}

const routes = [
  { path: '/admin', element: <DashboardLayout><ProtectedRoute><AdministrationPage /></ProtectedRoute></DashboardLayout> },
  { path: '/administration', element: <Navigate to="/admin" replace /> },
  {
    path: '/',
    element: <HomePage />,
  },
  {
    path: '/dashboard',
    element: (
      <DashboardLayout>
        <ProtectedRoute>
          <DashboardPage />
        </ProtectedRoute>
      </DashboardLayout>
    ),
  },
  {
    path: '/leads',
    element: (
      <DashboardLayout>
        <ProtectedRoute>
          <LeadsPage />
        </ProtectedRoute>
      </DashboardLayout>
    ),
  },
  {
    path: '/deals',
    element: (
      <DashboardLayout>
        <ProtectedRoute>
          <DealsPage />
        </ProtectedRoute>
      </DashboardLayout>
    ),
  },
  {
    path: '/contacts',
    element: (
      <DashboardLayout>
        <ProtectedRoute>
          <ContactsPage />
        </ProtectedRoute>
      </DashboardLayout>
    ),
  },
  {
    path: '/companies',
    element: (
      <DashboardLayout>
        <ProtectedRoute>
          <CompaniesPage />
        </ProtectedRoute>
      </DashboardLayout>
    ),
  },
  {
    path: '/tasks',
    element: (
      <DashboardLayout>
        <ProtectedRoute>
          <TasksPage />
        </ProtectedRoute>
      </DashboardLayout>
    ),
  },
  {
    path: '/projects',
    element: (
      <DashboardLayout>
        <ProtectedRoute>
          <ProjectsPage />
        </ProtectedRoute>
      </DashboardLayout>
    ),
  },
  {
    path: '/inbox',
    element: (
      <DashboardLayout>
        <ProtectedRoute>
          <EmailsPage />
        </ProtectedRoute>
      </DashboardLayout>
    ),
  },
  {
    path: '/calendar',
    element: (
      <DashboardLayout>
        <ProtectedRoute>
          <CalendarPage />
        </ProtectedRoute>
      </DashboardLayout>
    ),
  },
  {
    path: '/agent-console',
    element: (
      <DashboardLayout>
        <ProtectedRoute>
          <AgentConsolePage />
        </ProtectedRoute>
      </DashboardLayout>
    ),
  },
  {
    path: '/workflows',
    element: (
      <DashboardLayout>
        <ProtectedRoute>
          <WorkflowsPage />
        </ProtectedRoute>
      </DashboardLayout>
    ),
  },
  {
    path: '/integrations',
    element: (
      <DashboardLayout>
        <ProtectedRoute>
          <IntegrationsPage />
        </ProtectedRoute>
      </DashboardLayout>
    ),
  },
  {
    path: '/webhooks',
    element: (
      <DashboardLayout>
        <ProtectedRoute>
          <WebhooksPage />
        </ProtectedRoute>
      </DashboardLayout>
    ),
  },
  {
    path: '/agents',
    element: (
      <DashboardLayout>
        <ProtectedRoute>
          <AgentsPage />
        </ProtectedRoute>
      </DashboardLayout>
    ),
  },
  {
    path: '/schema',
    element: (
      <DashboardLayout>
        <ProtectedRoute>
          <SchemaPage />
        </ProtectedRoute>
      </DashboardLayout>
    ),
  },
  {
    path: '/audit',
    element: (
      <DashboardLayout>
        <ProtectedRoute>
          <AuditPage />
        </ProtectedRoute>
      </DashboardLayout>
    ),
  },
  {
    path: '/system',
    element: (
      <DashboardLayout>
        <ProtectedRoute>
          <SystemPage />
        </ProtectedRoute>
      </DashboardLayout>
    ),
  },
  {
    path: '/settings',
    element: (
      <DashboardLayout>
        <ProtectedRoute>
          <SettingsPage />
        </ProtectedRoute>
      </DashboardLayout>
    ),
  },
  { path: '/activity', element: <LegacyActivityRedirect /> },
  { path: '*', element: <Navigate to="/dashboard" replace /> },
  {
    path: '/login',
    element: <LoginPage />,
  },
]

function RouteError() {
  const error = useRouteError()
  const unavailable = error instanceof Error && /fetch|import|chunk|network/i.test(error.message)
  return <div className="ws-route-error" role="alert"><img src="/torq-mark.svg" alt="" /><h1>{unavailable ? 'This page could not be loaded.' : 'Something interrupted this page.'}</h1><p>Refresh the page to reconnect to your workspace.</p><button onClick={() => window.location.reload()}>Refresh page</button><a href="/dashboard">Return to dashboard</a></div>
}

const router = createBrowserRouter(routes.map(route => ({ ...route, errorElement: <RouteError /> })))

export default function App() {
  return <Suspense fallback={<div className="ws-route-loading" role="status"><Loader2 className="loading-spinner" size={20} />Loading TORQ...</div>}><RouterProvider router={router} /></Suspense>
}
