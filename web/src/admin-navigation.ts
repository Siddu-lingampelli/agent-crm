import { Database, KeyRound, LayoutGrid, Monitor, ScrollText, Settings2, Webhook } from 'lucide-react';

export const adminSections = [
  { href: '/admin', title: 'Overview', description: 'A clear view of your workspace administration.', icon: LayoutGrid },
  { href: '/agents', title: 'Access & agents', description: 'Manage agents, API keys, and scoped workspace access.', icon: KeyRound },
  { href: '/webhooks', title: 'Webhooks', description: 'Connect workspace events to the services you use.', icon: Webhook },
  { href: '/audit', title: 'Audit log', description: 'Review recorded events, actors, and affected records.', icon: ScrollText },
  { href: '/schema', title: 'Data model', description: 'Inspect database tables, columns, and applied migrations.', icon: Database },
  { href: '/system', title: 'System health', description: 'Monitor your runtime, database, and background jobs.', icon: Monitor },
  { href: '/settings', title: 'Settings', description: 'Manage your account and sign-in security.', icon: Settings2 },
];
export function isAdminPath(path: string) { return adminSections.some(section => section.href === path); }
