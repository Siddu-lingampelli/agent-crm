import { useState } from 'react';
import { Link } from 'react-router-dom';
import { ArrowUpRight, Check, Eye, EyeOff, KeyRound, Loader2, LockKeyhole, Mail, Plug, RefreshCw, ShieldCheck, Webhook } from 'lucide-react';
import { api } from './api';
import { FormField, Input } from './components';
import { AdminHeader, AdminPanel, AdminLoading, AdminError, AdminLink } from './AdminArea';
import { useAdminData, type AccountIdentity } from './admin-data';

export function SettingsPage() {
  const identity = useAdminData(() => api.get<AccountIdentity>('/auth/me'), 'settings-identity');
  const [current,setCurrent] = useState('');
  const [password,setPassword] = useState('');
  const [confirm,setConfirm] = useState('');
  const [show,setShow] = useState(false);
  const [saving,setSaving] = useState(false);
  const [error,setError] = useState('');
  const [success,setSuccess] = useState('');
  const changePassword = async (event: React.FormEvent) => {
    event.preventDefault(); setError(''); setSuccess('');
    if(password!==confirm){setError('The new passwords do not match.');return;}
    if(password.length<8){setError('Use at least 8 characters for your new password.');return;}
    setSaving(true);
    try {await api.post('/admin/settings/password',{current_password:current,new_password:password});setCurrent('');setPassword('');setConfirm('');setSuccess('Password updated. Other login sessions have been signed out.');}
    catch(caught){setError(caught instanceof Error?caught.message:'Could not change your password.');}
    finally{setSaving(false);}
  };
  return <div className="ad-page"><AdminHeader title="Settings" description="Your account, sign-in security, and connected workspace tools." actions={<button className="ad-button" onClick={identity.refresh} disabled={identity.loading}><RefreshCw size={14} className={identity.loading?'loading-spinner':''} />Refresh account</button>} />
    {identity.error && <AdminError message={identity.error} onRetry={identity.refresh} />}
    <div className="ad-settings-layout"><div className="ad-settings-side"><AdminPanel title="Your account" description="Current workspace sign-in">{identity.loading && !identity.data ? <AdminLoading /> : <div className="ad-account-profile"><span className="ad-account-avatar">{identity.data?.email.slice(0,2).toUpperCase() || 'T'}</span><strong>{identity.data?.email || 'Account unavailable'}</strong><span className="ad-status ad-status-healthy"><ShieldCheck size={11} />Workspace account</span><div><label htmlFor="account-email">Sign-in email</label><div className="ad-readonly-input"><Mail size={14} /><input id="account-email" value={identity.data?.email || ''} readOnly /></div><p>Your email is managed by your workspace account.</p></div><details><summary>Account details</summary><dl><dt>Account ID</dt><dd>{identity.data?.id || '—'}</dd><dt>Workspace ID</dt><dd>{identity.data?.workspace_id || '—'}</dd></dl></details></div>}</AdminPanel>
    <div className="ad-admin-note"><ShieldCheck size={20} /><h3>Scoped access for every agent.</h3><p>Keep agent permissions and API keys aligned with the work they need to do.</p><AdminLink to="/agents">Manage access</AdminLink></div></div>
    <div className="ad-settings-main"><AdminPanel title="Sign-in security" description="Update the password you use to access this workspace"><form className="ad-password-form" onSubmit={changePassword} aria-busy={saving}>
      <div className="ad-security-intro"><span><LockKeyhole size={19} /></span><div><h3>Change password</h3><p>Your current session stays signed in. Other sessions are revoked after a successful change.</p></div></div>
      {error && <div className="ad-error" role="alert"><span>{error}</span></div>}{success && <div className="ad-success" role="status"><Check size={15} />{success}</div>}
      <FormField label="Current password" required><Input name="current_password" type={show?'text':'password'} autoComplete="current-password" value={current} onChange={event=>setCurrent(event.target.value)} required disabled={saving} maxLength={4096} /></FormField>
      <div className="ad-two-columns"><FormField label="New password" helpText="Use at least 8 characters." required><Input name="new_password" type={show?'text':'password'} autoComplete="new-password" value={password} onChange={event=>setPassword(event.target.value)} required disabled={saving} minLength={8} maxLength={4096} /></FormField><FormField label="Confirm new password" required><Input name="confirm_password" type={show?'text':'password'} autoComplete="new-password" value={confirm} onChange={event=>setConfirm(event.target.value)} required disabled={saving} minLength={8} maxLength={4096} /></FormField></div>
      <div className="ad-form-footer"><button type="button" className="ad-show-password" aria-pressed={show} onClick={()=>setShow(!show)}>{show?<EyeOff size={14} />:<Eye size={14} />}{show?'Hide passwords':'Show passwords'}</button><button type="submit" className="ad-button ad-button-primary" disabled={saving}>{saving?<Loader2 size={14} className="loading-spinner" />:<KeyRound size={14} />}{saving?'Updating...':'Update password'}</button></div>
    </form></AdminPanel>
    <AdminPanel title="Connected tools" description="Configure the services and credentials used by your workspace"><div className="ad-settings-links">{[{href:'/integrations',title:'Integrations',text:'Connected providers and service configuration.',icon:Plug},{href:'/webhooks',title:'Webhooks',text:'Event destinations and delivery configuration.',icon:Webhook},{href:'/agents',title:'Agents & API keys',text:'Automation access, scopes, and credentials.',icon:KeyRound}].map(item=><Link key={item.href} to={item.href}><span><item.icon size={17} /></span><div><strong>{item.title}</strong><p>{item.text}</p></div><ArrowUpRight size={14} /></Link>)}</div></AdminPanel></div></div>
  </div>;
}
