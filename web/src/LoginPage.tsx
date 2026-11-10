import { useEffect, useRef, useState } from 'react';
import { Link, useLocation } from 'react-router-dom';
import { ArrowLeft, ArrowRight, ArrowUpRight, Eye, EyeOff, AlertCircle, Loader2 } from 'lucide-react';
import { api, ApiError } from './api';
import './LoginPage.css';

function WorkspaceDrawing() {
  return <svg className="tl-drawing" viewBox="0 0 520 340" fill="none" role="img" aria-labelledby="tl-drawing-title"><title id="tl-drawing-title">Contacts, deals, and tasks in one workspace</title>
    <path d="M53 38h331v242H53z" fill="#ff4f00" stroke="#000" strokeWidth="1.5" /><path d="M81 63h331v242H81z" fill="#fff" stroke="#000" strokeWidth="1.5" />
    <path d="M81 106h331" stroke="#000" /><circle cx="99" cy="85" r="3" fill="#ff4f00" /><text x="114" y="89" fontSize="10" letterSpacing="1" fill="#000">TORQ / WORKSPACE</text><path d="M374 81h18m-18 7h18" stroke="#000" />
    <g className="tl-drawing-row tl-drawing-row-1"><circle cx="114" cy="142" r="12" stroke="#000" /><circle cx="114" cy="139" r="3" fill="#000" /><path d="M108 150c0-9 12-9 12 0" fill="#000" /><text x="144" y="147" fontSize="15" fill="#000">Contacts</text><path d="M349 142h40m-6-6 6 6-6 6" stroke="#000" /></g>
    <path d="M101 168h291" stroke="#000" />
    <g className="tl-drawing-row tl-drawing-row-2"><path d="M104 204v-13h6v13m4 0v-19h6v19m4 0v-26h6v26" fill="#ff4f00" stroke="#000" /><text x="144" y="200" fontSize="15" fill="#000">Deals</text><path d="M349 195h40m-6-6 6 6-6 6" stroke="#000" /></g>
    <path d="M101 221h291" stroke="#000" />
    <g className="tl-drawing-row tl-drawing-row-3"><rect x="103" y="240" width="23" height="23" fill="#ff4f00" stroke="#000" /><path d="m108 251 5 5 9-11" stroke="#000" strokeWidth="1.5" /><text x="144" y="256" fontSize="15" fill="#000">Tasks</text></g>
    <circle cx="419" cy="267" r="42" fill="#000" /><path d="M400 267h38m-17-17 17 17-17 17" stroke="#fff" strokeWidth="2" /><path d="M32 319h69m364-281v40m-20-20h40" stroke="#000" />
  </svg>;
}

export function LoginPage() {
  const location = useLocation();
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [showPassword, setShowPassword] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const errorRef = useRef<HTMLDivElement>(null);
  const requestedPath = typeof location.state?.from === 'string' ? location.state.from : location.state?.from?.pathname;
  const from = typeof requestedPath === 'string' && requestedPath.startsWith('/') && !requestedPath.startsWith('//') && !requestedPath.includes('\\') && requestedPath !== '/login' ? requestedPath : '/dashboard';
  useEffect(() => {
    const previousTitle = document.title;
    document.title = 'Sign in - TORQ';
    return () => { document.title = previousTitle; };
  }, []);
  useEffect(() => { if (error) errorRef.current?.focus(); }, [error]);

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    if (loading) return;
    setError(null); setLoading(true);
    try {
      await api.post('/auth/login', { email: email.trim(), password });
      window.location.assign(from);
    } catch (err) {
      setError(err instanceof ApiError && err.status === 401 ? 'Email or password is incorrect.' : err instanceof ApiError && err.status === 400 ? err.message : 'Could not sign in. Please try again.');
      setLoading(false);
    }
  };

  return <div className="torq-login">
    <a className="tl-skip" href="#tl-form">Skip to sign in</a>
    <header className="tl-header"><Link to="/" className="tl-brand" aria-label="TORQ home"><svg viewBox="0 0 30 30" aria-hidden="true"><path d="M3 8h20v6H3zM9 20h20v6H9z" fill="currentColor" transform="rotate(-28 16 16)" /></svg><span>torq<span>.</span></span></Link><Link to="/" className="tl-back"><ArrowLeft size={15} />Back to home</Link></header>
    <div className="tl-layout"><aside className="tl-editorial" aria-label="Your CRM workspace"><div className="tl-editorial-copy"><span className="tl-eyebrow">TORQ CRM</span><h2>Contacts.<br />Deals.<br /><span>Follow-ups.</span></h2><p>One place to manage your customer relationships.</p></div><WorkspaceDrawing /><div className="tl-editorial-footer"><span>Your workspace, on your infrastructure.</span><ArrowUpRight size={16} /></div></aside>
      <main className="tl-main"><div className="tl-form-wrap"><span className="tl-eyebrow">WORKSPACE ACCESS</span><h1>Sign in.</h1><p className="tl-description">Use your workspace account to continue.</p>
        {error && <div className="tl-error" ref={errorRef} tabIndex={-1} id="tl-error" role="alert"><AlertCircle size={17} /><span>{error}</span></div>}
        <form id="tl-form" onSubmit={handleSubmit} className="tl-form" aria-label="Sign in to TORQ" aria-busy={loading} aria-describedby={error ? 'tl-error' : undefined}>
          <div className="tl-field"><label htmlFor="tl-email">Email address</label><input id="tl-email" type="email" name="email" value={email} onChange={event => setEmail(event.target.value)} placeholder="you@company.com" autoComplete="username" autoCapitalize="none" spellCheck={false} required disabled={loading} /></div>
          <div className="tl-field"><label htmlFor="tl-password">Password</label><div className="tl-password"><input id="tl-password" type={showPassword ? 'text' : 'password'} name="password" value={password} onChange={event => setPassword(event.target.value)} placeholder="Enter your password" autoComplete="current-password" required disabled={loading} /><button type="button" className="tl-password-toggle" onClick={() => setShowPassword(value => !value)} aria-label={showPassword ? 'Hide password' : 'Show password'} aria-pressed={showPassword} disabled={loading}>{showPassword ? <EyeOff size={18} /> : <Eye size={18} />}</button></div></div>
          <button type="submit" className="tl-submit" disabled={loading}>{loading ? 'Signing in...' : 'Sign in'}{loading ? <Loader2 size={18} className="tl-spinner" /> : <ArrowRight size={18} />}</button>
        </form><p className="tl-help">Need access? Contact your workspace administrator.</p>
      </div><footer className="tl-footer"><span>TORQ CRM</span><Link to="/">View product<ArrowUpRight size={13} /></Link></footer></main>
    </div>
  </div>;
}
