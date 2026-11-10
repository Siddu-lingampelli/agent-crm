# TORQ User Guide

## Overview

This TORQ (Customer Relationship Management) system helps you manage contacts, companies, deals, tasks, projects, and more. Built as a modular monolith with a modern React frontend and Go backend.

## Getting Started

### Accessing the Application

1. Navigate to `https://yourdomain.com` (or `http://localhost:8080` for local development)
2. On first access, you'll be redirected to the setup page to create the owner account
3. Enter your email and a strong password
4. Click "Create Owner Account"
5. You'll be automatically logged in and taken to the dashboard

### First-Time Setup

After creating the owner account:
1. **Configure your workspace** - Set your company name, timezone, and preferences
2. **Add team members** - Navigate to Settings → Team → Invite Members
3. **Import existing data** - Use Settings → Import to bring in contacts from CSV
4. **Set up integrations** - Connect email, calendar, and other tools

## Core Features

### Dashboard

The dashboard provides an at-a-glance view of your business:
- **Pipeline value** - Total value of open deals
- **Recent activity** - Latest interactions across all records
- **Upcoming tasks** - Tasks due today and this week
- **Key metrics** - Conversion rates, average deal size, cycle time

### Contacts

Manage people you interact with:
- **Create contacts** - Click "New Contact" and fill in details
- **Import contacts** - Upload CSV with columns: first_name, last_name, email, phone, company, title
- **Contact details** - View interaction history, deals, tasks, emails, notes
- **Tags & custom fields** - Organize with tags and add custom data fields
- **Merge duplicates** - Find and merge duplicate contacts

### Companies

Track organizations:
- **Company records** - Name, domain, industry, size, website, address
- **Associated contacts** - See all contacts at a company
- **Deal pipeline** - View all deals for a company
- **Custom fields** - Add company-specific data

### Deals

Manage your sales pipeline:
- **Pipeline stages** - Customize stages (Lead → Qualified → Proposal → Negotiation → Won/Lost)
- **Deal cards** - Drag and drop between stages
- **Deal details** - Value, expected close date, probability, owner
- **Activities** - Log calls, meetings, emails directly on deals
- **Forecasting** - Weighted pipeline by stage probability

### Tasks

Stay organized with task management:
- **Create tasks** - Assign to yourself or team members
- **Due dates & reminders** - Set deadlines and get notifications
- **Recurring tasks** - Set up daily, weekly, monthly recurrences
- **Task views** - My tasks, team tasks, overdue, completed
- **Link to records** - Attach tasks to contacts, companies, deals, projects

### Projects

Track project work:
- **Project phases** - Plan, execute, monitor, close
- **Milestones** - Key deliverables with dates
- **Team assignments** - Assign members with roles
- **Time tracking** - Log hours against projects
- **Project health** - Status indicators (on track, at risk, off track)

### Calendar

Schedule and view events:
- **Multiple views** - Day, week, month, agenda
- **Event types** - Meetings, calls, deadlines, reminders
- **Attendees** - Invite team members and external contacts
- **Recurring events** - Set up recurring meetings
- **Integration** - Sync with Google Calendar, Outlook

### Search

Powerful full-text search across all records:
- **Global search** - Search contacts, companies, deals, tasks, projects, emails
- **Filters** - Narrow by type, date range, owner, tags
- **Saved searches** - Save common queries for quick access
- **Advanced syntax** - Use `field:value`, `"exact phrase"`, `AND`, `OR`, `-exclude`

### Email

Manage communications:
- **Email sync** - Connect Gmail/Outlook to sync emails automatically
- **Threaded view** - See conversation history on contact/company/deal records
- **Send from TORQ** - Compose and send emails without leaving the app
- **Templates** - Create reusable email templates
- **Tracking** - See opens, clicks, replies

### Reports & Analytics

Gain insights with built-in reports:
- **Pipeline report** - Deals by stage, value, velocity
- **Activity report** - Calls, meetings, emails by team member
- **Conversion funnel** - Lead to opportunity to close rates
- **Forecast vs actual** - Compare predictions to reality
- **Custom reports** - Build your own with the report builder

## Settings & Administration

### User Management

- **Roles** - Admin, Manager, User, Viewer
- **Permissions** - Granular control per feature
- **Teams** - Organize users into teams
- **SSO** - SAML/OIDC integration (Enterprise)

### Workspace Settings

- **Branding** - Logo, colors, custom domain
- **Timezone & locale** - Date/time formats, language
- **Custom fields** - Define fields for each record type
- **Pipeline configuration** - Stages, probabilities, requirements
- **Automation** - Workflow rules, triggers, actions

### Integrations

- **Email** - Gmail, Outlook, IMAP/SMTP
- **Calendar** - Google Calendar, Outlook Calendar
- **Webhooks** - Real-time event notifications
- **API** - REST API for custom integrations
- **MCP** - Model Context Protocol for AI agents

### Data Management

- **Import/Export** - CSV, JSON for all record types
- **Backup** - Automated daily backups, manual snapshots
- **Audit log** - Track all changes for compliance
- **Data retention** - Configure retention policies

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `g` then `d` | Go to Dashboard |
| `g` then `c` | Go to Contacts |
| `g` then `o` | Go to Companies |
| `g` then `l` | Go to Deals (Leads) |
| `g` then `t` | Go to Tasks |
| `g` then `p` | Go to Projects |
| `g` then `k` | Go to Calendar |
| `/` | Focus search |
| `n` | New record (context-aware) |
| `?` | Show help dialog |

## Tips & Best Practices

1. **Keep data clean** - Regularly merge duplicates, archive old records
2. **Use tags consistently** - Establish a tagging taxonomy for your team
3. **Log activities promptly** - Don't let calls/meetings go unrecorded
4. **Set up automation** - Reduce manual work with workflow rules
5. **Review pipeline weekly** - Keep deals moving, update stages
6. **Use custom fields** - Capture business-specific data without custom code

## Troubleshooting

### Common Issues

**Can't log in**
- Check email/password
- Clear browser cache/cookies
- Contact admin for password reset

**Emails not syncing**
- Verify email integration settings
- Check IMAP/SMTP credentials
- Review sync logs in Settings → Integrations

**Search not returning results**
- Check FTS index status in Admin → System
- Rebuild search index if needed
- Verify workspace permissions

**Performance issues**
- Check Admin → System → Memory/CPU
- Reduce dashboard widgets
- Archive old data

### Getting Help

- **Documentation** - Check docs.yourdomain.com
- **In-app help** - Click `?` icon or press `?` key
- **Support** - Email support@yourdomain.com
- **Community** - Join our Discord/Slack community

## Mobile Access

TORQ is a Progressive Web App (PWA):
- **Install** - "Add to Home Screen" from browser menu
- **Offline** - View cached data when offline
- **Notifications** - Push notifications for tasks, mentions
- **Responsive** - Works on phone, tablet, desktop

## Security Best Practices

1. **Use strong passwords** - Minimum 12 characters
2. **Enable 2FA** - TOTP authenticator app recommended
3. **Review sessions** - Check active sessions in Settings → Security
4. **API keys** - Rotate regularly, use least-privilege scopes
5. **Audit log** - Regularly review for suspicious activity

---

*For administrator documentation, see [Admin Guide](admin-guide.md). For API reference, see [API Documentation](api-docs.md). For developer setup, see [Developer Guide](developer-guide.md).*