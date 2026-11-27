import { useEffect, useRef, useState, useSyncExternalStore, type KeyboardEvent, type ReactNode } from 'react';
import { AnimatePresence, MotionConfig, motion } from 'framer-motion';
import { Link } from 'react-router-dom';
import { ArrowDown, ArrowRight, ArrowUpRight, Check, CheckCheck, ChevronDown, CircleHelp, Code2, Contact, GitBranch, LayoutGrid, ListTodo, Menu, MoreHorizontal, Pause, Play, Plus, Search, Users, Webhook, X } from 'lucide-react';
import './HomePage.css';

const ease = [0.22, 1, 0.36, 1] as const;
const year = new Date().getFullYear();

const reducedMotionQuery = '(prefers-reduced-motion: reduce)';
function subscribeMotionPreference(onChange: () => void) {
  const media = window.matchMedia(reducedMotionQuery);
  media.addEventListener('change', onChange);
  return () => media.removeEventListener('change', onChange);
}
function useMotionPreference() {
  return useSyncExternalStore(subscribeMotionPreference, () => window.matchMedia(reducedMotionQuery).matches, () => true);
}

function Brand({ inverse = false }: { inverse?: boolean }) {
  return <span className={`th-brand${inverse ? ' th-brand-inverse' : ''}`}><svg viewBox="0 0 30 30" fill="none" aria-hidden="true"><path d="M3 8h20v6H3zM9 20h20v6H9z" fill="currentColor" transform="rotate(-28 16 16)" /></svg><span>torq<span>.</span></span></span>;
}

function Reveal({ children, className = '', delay = 0 }: { children: ReactNode; className?: string; delay?: number }) {
  const reduce = useMotionPreference();
  return <motion.div className={className} initial={reduce ? false : { opacity: 0, y: 18 }} whileInView={{ opacity: 1, y: 0 }} viewport={{ once: true, amount: 0.08 }} transition={{ duration: reduce ? 0 : 0.6, delay: reduce ? 0 : delay, ease }}>{children}</motion.div>;
}

function Eyebrow({ children }: { children: ReactNode }) {
  return <span className="th-eyebrow"><span aria-hidden="true">/</span>{children}</span>;
}

function HeroIllustration() {
  const reduce = useMotionPreference();
  const [playing, setPlaying] = useState(true);
  return <div className={`th-hero-art${!playing || reduce ? ' is-paused' : ''}`}>
    <div className="th-art-meta"><span>CONTACT / OPPORTUNITY</span><button type="button" className="th-motion-toggle" onClick={() => setPlaying(value => !value)} disabled={!!reduce} aria-label={reduce ? 'Illustration motion disabled by your preference' : playing ? 'Pause illustration animation' : 'Play illustration animation'} aria-pressed={playing && !reduce}>{playing && !reduce ? <Pause size={12} /> : <Play size={12} />}<span>{reduce ? 'Motion off' : playing ? 'Pause motion' : 'Play motion'}</span></button></div>
    <svg className="th-hero-drawing" viewBox="0 0 600 510" role="img" aria-labelledby="th-art-title th-art-desc">
      <title id="th-art-title">Contact and deal tracking</title><desc id="th-art-desc">A contact record, an open deal, and an assigned follow-up task.</desc>
      <defs><pattern id="th-dot-pattern" width="18" height="18" patternUnits="userSpaceOnUse"><circle cx="2" cy="2" r="1" fill="#000" /></pattern></defs>
      <circle cx="336" cy="257" r="181" fill="none" stroke="#000" strokeWidth="1" />
      <path d="M155 257a181 181 0 0 1 181-181" fill="none" stroke="#FF5A1F" strokeWidth="9" />
      <path className="th-route-line" d="M68 160v215h105M449 112h94v232" fill="none" stroke="#000" strokeWidth="1.5" strokeDasharray="5 7" />
      <path d="m168 369 8 6-8 6m369-43 6 8 6-8" fill="none" stroke="#000" strokeWidth="1.5" />
      <path d="M374 408h108v65H374z" fill="url(#th-dot-pattern)" />
      <g className="th-growth-arrow"><path d="M241 369V227h-65L346 57l170 170h-65v142Z" fill="#FF5A1F" stroke="#000" strokeWidth="2" strokeLinejoin="round" /><path d="M262 347V211h-46L346 81" fill="none" stroke="#fff" strokeWidth="2" /></g>
      <g className="th-art-contact"><path d="M22 85h168v104H22z" fill="#fff" stroke="#000" strokeWidth="1.5" /><path d="M22 112h168" stroke="#000" /><circle cx="35" cy="99" r="2" fill="#FF5A1F" /><text x="46" y="102" fontSize="8" letterSpacing="1.1" fill="#000">CONTACT RECORD</text><circle cx="52" cy="148" r="20" fill="#000" /><circle cx="52" cy="142" r="6" fill="#fff" /><path d="M41 162c0-17 22-17 22 0" fill="#fff" /><text x="85" y="145" fontSize="12" fontWeight="600" fill="#000">Alex Morgan</text><text x="85" y="162" fontSize="9" fill="#000">Studio North</text></g>
      <g className="th-art-deal"><path d="M380 310h197v105H380z" fill="#fff" stroke="#000" strokeWidth="1.5" /><path d="M380 338h197" stroke="#000" /><text x="394" y="328" fontSize="8" letterSpacing="1" fill="#000">OPEN DEAL</text><text x="394" y="376" fontSize="30" fontWeight="600" letterSpacing="-1" fill="#000">$12,000</text><text x="395" y="398" fontSize="9" fill="#000">Proposal sent</text><path d="M538 353v28m-14-14h28m-24 10 20-20m-20 0 20 20" stroke="#FF5A1F" strokeWidth="4" /></g>
      <g className="th-illustrated-hand" fill="#fff" stroke="#000" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="m108 505 30-97c5-20 12-38 24-55l51-77c5-8 15-9 21-3 5 5 4 13 0 20l-24 39 57-69c5-7 14-8 20-3 6 5 5 14 0 20l-35 47 47-49c6-6 15-5 20 1 4 5 2 13-3 19l-43 49 36-29c7-6 15-4 19 2 4 6 1 13-5 19l-48 45c-10 11-12 25-8 36l15 37-4 93" /><path d="m173 400 19-43 18-25m12 30 30-35m-10 49 31-30m-58 78c-9-21-4-39 7-52m-76 67 96 36m-101-23 98 36" /><path d="m108 505 12-41 128 38-1 3" fill="#FF5A1F" /></g>
      <g transform="translate(505 42)" stroke="#000" strokeWidth="2"><path d="M0 17h34M17 0v34" /><circle cx="17" cy="17" r="10" fill="none" /></g>
      <g className="th-art-check"><rect x="25" y="272" width="133" height="40" fill="#000" /><circle cx="46" cy="292" r="10" fill="#FF5A1F" /><path d="m41 292 4 4 7-8" fill="none" stroke="#000" strokeWidth="1.5" /><text x="63" y="295" fontSize="9" fill="#fff">Follow-up assigned</text></g>
      <path d="M313 460h24m-12-12v24" stroke="#000" /><text x="377" y="493" fontSize="8" letterSpacing="1.2" fill="#000">CONTACTS / DEALS / TASKS</text>
    </svg>
    <span className="th-art-caption">Contact details and deal tracking.</span>
  </div>;
}

type PreviewView = 'pipeline' | 'contacts' | 'tasks';
const previewViews: { id: PreviewView; label: string; icon: typeof LayoutGrid; description: string }[] = [
  { id: 'pipeline', label: 'Pipeline', icon: LayoutGrid, description: 'Track opportunities by sales stage.' },
  { id: 'contacts', label: 'Contacts', icon: Users, description: 'Contact and company records.' },
  { id: 'tasks', label: 'Tasks', icon: ListTodo, description: 'Assign tasks and track follow-up dates.' },
];
const pipeline = [
  { name: 'Discovery', total: '$18,000', deals: [{ company: 'Form Studio', name: 'Brand partnership', value: '$6,000', initials: 'FS', note: 'Introduction call' }, { company: 'Arbor & Co.', name: 'Growth consulting', value: '$12,000', initials: 'AC', note: 'Send follow-up' }] },
  { name: 'In conversation', total: '$17,000', deals: [{ company: 'Northline', name: 'Workspace expansion', value: '$9,000', initials: 'N', note: 'Review proposal' }, { company: 'Paperplane', name: 'Platform rollout', value: '$8,000', initials: 'P', note: 'Meet the team' }] },
  { name: 'Proposal', total: '$19,000', deals: [{ company: 'Sonder', name: 'Annual partnership', value: '$15,000', initials: 'S', note: 'Confirm details' }, { company: 'Common Ground', name: 'A new collaboration', value: '$4,000', initials: 'CG', note: 'Proposal review' }] },
];
const contacts = [
  { name: 'Alex Morgan', role: 'Founder', company: 'Studio North', email: 'alex@studionorth.example', initials: 'AM' },
  { name: 'Jamie Chen', role: 'Creative director', company: 'Form Studio', email: 'jamie@formstudio.example', initials: 'JC' },
  { name: 'Sam Rivera', role: 'Partnerships', company: 'Arbor & Co.', email: 'sam@arbor.example', initials: 'SR' },
  { name: 'Taylor Ellis', role: 'Co-founder', company: 'Paperplane', email: 'taylor@paperplane.example', initials: 'TE' },
];
const sampleTasks = [
  { id: 'proposal', title: 'Send the Studio North proposal', company: 'Studio North', due: 'Today', priority: 'High priority' },
  { id: 'intro', title: 'Introduce Jamie to the team', company: 'Form Studio', due: 'Today', priority: 'Follow-up' },
  { id: 'review', title: 'Review the partnership details', company: 'Arbor & Co.', due: 'Tomorrow', priority: 'Review' },
  { id: 'thanks', title: 'Send a thank-you note', company: 'Paperplane', due: 'Tomorrow', priority: 'Follow-up' },
];

function ProductPreview() {
  const [active, setActive] = useState<PreviewView>('pipeline');
  const [completed, setCompleted] = useState<string[]>(['thanks']);
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([]);
  const reduce = useMotionPreference();
  const current = previewViews.find(view => view.id === active)!;
  const onTabKey = (event: KeyboardEvent<HTMLButtonElement>, index: number) => {
    let next = index;
    if (event.key === 'ArrowRight') next = (index + 1) % previewViews.length;
    else if (event.key === 'ArrowLeft') next = (index + previewViews.length - 1) % previewViews.length;
    else if (event.key === 'Home') next = 0;
    else if (event.key === 'End') next = previewViews.length - 1;
    else return;
    event.preventDefault(); setActive(previewViews[next].id); tabRefs.current[next]?.focus();
  };
  return <>
    <div className="th-preview-controls"><div className="th-preview-tabs" role="tablist" aria-label="Explore the CRM preview">{previewViews.map((view, index) => <button key={view.id} id={`th-tab-${view.id}`} type="button" role="tab" aria-selected={active === view.id} aria-controls={`th-panel-${view.id}`} tabIndex={active === view.id ? 0 : -1} ref={element => { tabRefs.current[index] = element; }} onKeyDown={event => onTabKey(event, index)} onClick={() => setActive(view.id)}><view.icon size={17} />{view.label}</button>)}</div><span className="th-preview-hint">Preview CRM views <ArrowDown size={15} /></span></div>
    <figure className="th-product-figure">
      <div className="th-product-window">
        <aside className="th-product-sidebar" aria-hidden="true"><Brand /><span className="th-demo-workspace">Studio workspace <ChevronDown size={12} /></span><div className="th-demo-nav">{previewViews.map(view => <span key={view.id} className={active === view.id ? 'is-active' : ''}><view.icon size={16} />{view.label}</span>)}<span><GitBranch size={16} />Workflows</span></div><span className="th-demo-help"><CircleHelp size={15} />Help & resources</span><div className="th-demo-user"><span>AM</span><div>Alex Morgan<small>Your workspace</small></div></div></aside>
        <div className="th-product-main"><div className="th-preview-topbar"><span>Workspace <span>/</span> {current.label}</span><span className="th-preview-top-icons" aria-hidden="true"><Search size={16} /><span>AM</span></span></div>
          <div className="th-preview-heading"><div><span className="th-preview-kicker">STUDIO WORKSPACE</span><h3>{active === 'pipeline' ? 'Sales pipeline.' : active === 'contacts' ? 'Contacts.' : 'Tasks.'}</h3><p>{active === 'pipeline' ? '$54,000 in pipeline / 6 open deals' : active === 'contacts' ? '4 contact records' : `${sampleTasks.length - completed.length} open tasks`}</p></div><Link className="th-preview-open" aria-label={`Open your ${current.label.toLowerCase()} workspace`} to={active === 'contacts' ? '/contacts' : active === 'tasks' ? '/tasks' : '/deals'}><span>Open CRM</span><ArrowUpRight size={15} /></Link></div>
          <AnimatePresence mode="wait" initial={false}><motion.div key={active} role="tabpanel" id={`th-panel-${active}`} aria-labelledby={`th-tab-${active}`} tabIndex={0} className="th-preview-panel" initial={{ opacity: 0, y: reduce ? 0 : 6 }} animate={{ opacity: 1, y: 0 }} exit={{ opacity: 0, y: reduce ? 0 : -4 }} transition={{ duration: reduce ? 0 : 0.16 }}>
            {active === 'pipeline' && <><p className="th-swipe-hint">Scroll to explore the pipeline <ArrowRight size={12} aria-hidden="true" /></p><div className="th-demo-pipeline">{pipeline.map((column, index) => <div className="th-pipeline-column" key={column.name}><div className="th-column-heading"><span><i className={index === 2 ? 'is-orange' : ''} />{column.name}<small>2</small></span><MoreHorizontal size={15} /></div>{column.deals.map(deal => <article className="th-deal" key={deal.company}><div className="th-deal-company"><span className="th-company-mark">{deal.initials}</span>{deal.company}<MoreHorizontal size={14} /></div><h4>{deal.name}</h4><strong>{deal.value}</strong><div className="th-deal-footer"><span><CheckCheck size={12} />{deal.note}</span><ArrowUpRight size={13} /></div></article>)}<span className="th-column-total">Stage value <strong>{column.total}</strong></span></div>)}</div></>}
            {active === 'contacts' && <div className="th-contacts-scroll"><table className="th-demo-contacts"><thead><tr><th>Name</th><th>Company</th><th>Email</th><th>Status</th></tr></thead><tbody>{contacts.map(contact => <tr key={contact.name}><td><div><span className="th-person-initial">{contact.initials}</span><span><strong>{contact.name}</strong><small>{contact.role}</small></span></div></td><td>{contact.company}</td><td>{contact.email}</td><td><span className="th-relationship-tag"><Check size={12} />Active</span></td></tr>)}</tbody></table></div>}
            {active === 'tasks' && <div className="th-demo-tasks"><div className="th-task-list-caption"><span>YOUR NEXT STEPS</span><span>Try checking one off <ArrowDown size={12} /></span></div>{sampleTasks.map(task => <label className={`th-task-row${completed.includes(task.id) ? ' is-complete' : ''}`} key={task.id}><input type="checkbox" checked={completed.includes(task.id)} onChange={() => setCompleted(previous => previous.includes(task.id) ? previous.filter(id => id !== task.id) : [...previous, task.id])} /><span className="th-task-description"><strong>{task.title}</strong><small>{task.company}</small></span><span className="th-task-priority">{task.priority}</span><span className="th-task-due">{task.due}</span></label>)}</div>}
          </motion.div></AnimatePresence>
        </div>
      </div><figcaption><span><span className="th-sample-mark" />Interactive product preview</span><span>Sample data only. No changes are made to your CRM.</span></figcaption>
    </figure><p className="th-preview-description" aria-live="polite">{current.description}</p>
  </>;
}

function StepIllustration({ step }: { step: number }) {
  return <svg viewBox="0 0 300 175" fill="none" aria-hidden="true" className="th-step-art">
    {step === 0 ? <><path d="M41 22h183v130H41z" fill="#000" stroke="#fff" /><path d="M41 45h183" stroke="#fff" /><circle cx="54" cy="34" r="2" fill="#FF5A1F" /><circle cx="88" cy="91" r="28" fill="#FF5A1F" /><circle cx="88" cy="81" r="9" fill="#000" /><path d="M71 110c0-25 34-25 34 0" fill="#000" /><path d="M133 76h62m-62 17h45m-45 17h54" stroke="#fff" /><path d="M213 119h53v40h-53z" fill="#FF5A1F" /><path d="m229 138 7 7 15-17" stroke="#000" strokeWidth="2" /></> : step === 1 ? <><path d="M30 132h240M52 132V91h45v41M124 132V64h45v68" stroke="#fff" /><path d="M196 132V35h45v97" fill="#FF5A1F" /><path className="th-step-dash" d="m43 67 63-37 54 9 70-25" stroke="#fff" strokeDasharray="4 5" /><path d="m216 10 15 3-7 13" stroke="#fff" /><circle cx="106" cy="30" r="5" fill="#000" stroke="#fff" /><circle cx="160" cy="39" r="5" fill="#000" stroke="#fff" /></> : <><circle cx="153" cy="87" r="65" stroke="#fff" /><circle cx="153" cy="87" r="44" stroke="#fff" /><circle cx="153" cy="87" r="22" fill="#FF5A1F" /><path d="m152 88 88-60m-35 0h35v34" stroke="#FF5A1F" strokeWidth="5" /><path d="M65 126h31m-15-15v30M65 40h12M71 34v12" stroke="#fff" /><path d="M228 132h24m-12-12v24" stroke="#FF5A1F" /></>}
  </svg>;
}

function ConnectedIllustration() {
  return <div className="th-connected-art" role="img" aria-label="CRM records connect to agent tools and webhooks, with each step recorded in an audit log">
    <span className="th-connected-caption">CRM / CONNECTIONS</span><svg viewBox="0 0 520 340" fill="none" aria-hidden="true"><path className="th-connector-dash" d="M129 102h131v134h131M260 102h131" stroke="#000" strokeWidth="1.5" strokeDasharray="5 6" /><circle cx="260" cy="102" r="6" fill="#fff" stroke="#000" /><circle cx="260" cy="236" r="6" fill="#fff" stroke="#000" /><path d="m378 96 8 6-8 6m0 122 8 6-8 6" stroke="#000" /><circle cx="130" cy="102" r="55" fill="#FF5A1F" stroke="#000" /><path d="M108 80h44v45h-44z" stroke="#000" strokeWidth="2" /><circle cx="130" cy="96" r="6" fill="#000" /><path d="M119 116c0-16 22-16 22 0" fill="#000" /><rect x="390" y="48" width="100" height="108" fill="#fff" stroke="#000" /><path d="m427 78-15 15 15 15m27-30 15 15-15 15m-8-35-12 40" stroke="#000" strokeWidth="2" /><text x="404" y="140" fill="#000" fontSize="10">AGENT TOOLS</text><rect x="390" y="190" width="100" height="100" fill="#000" /><circle cx="424" cy="223" r="9" stroke="#fff" /><circle cx="458" cy="223" r="9" stroke="#fff" /><path d="m424 223 17 20 17-20m-17 20v10" stroke="#fff" /><text x="405" y="276" fill="#fff" fontSize="10">WEBHOOKS</text><text x="83" y="184" fill="#000" fontSize="10" letterSpacing="1">YOUR CRM DATA</text><rect x="47" y="233" width="194" height="51" fill="#fff" stroke="#000" /><path d="m63 257 5 5 10-11" stroke="#FF5A1F" strokeWidth="2" /><text x="89" y="262" fill="#000" fontSize="11">Context. Action. Audit trail.</text><path d="M31 36h20m-10-10v20M307 299h20m-10-10v20" stroke="#000" /></svg><span className="th-connected-footer">CRM records, agent tools, and webhooks.</span>
  </div>;
}

const questions = [
  { question: 'Which records does TORQ support?', answer: 'Manage contacts, companies, leads, deals, tasks, and projects. The sales pipeline tracks deal stages, values, and expected close dates.' },
  { question: 'How do I access my workspace?', answer: 'Select Log in and use the account created by your workspace administrator. Contact the administrator if you need an account.' },
  { question: 'Does the preview use my CRM data?', answer: 'No. The preview uses sample records. Changing views or checking off tasks in the preview does not affect your workspace.' },
  { question: 'Can I self-host TORQ and connect other tools?', answer: 'Yes. TORQ can run on your infrastructure and supports API access, agent tools, workflows, and webhooks. Configure connections and permissions in the workspace.' },
];

export function HomePage() {
  const [mobileOpen, setMobileOpen] = useState(false);
  const [openQuestion, setOpenQuestion] = useState<number | null>(0);
  const reduce = useMotionPreference();
  const menuRef = useRef<HTMLButtonElement>(null);
  useEffect(() => {
    const previousTitle = document.title;
    document.title = 'TORQ - Contacts, deals, and tasks';
    return () => { document.title = previousTitle; };
  }, []);
  useEffect(() => {
    if (!mobileOpen) return;
    const close = (event: globalThis.KeyboardEvent) => { if (event.key === 'Escape') { setMobileOpen(false); menuRef.current?.focus(); } };
    window.addEventListener('keydown', close);
    return () => window.removeEventListener('keydown', close);
  }, [mobileOpen]);
  const heroMotion = (delay: number) => ({ initial: reduce ? false as const : { opacity: 0, y: 18 }, animate: { opacity: 1, y: 0 }, transition: { duration: reduce ? 0 : 0.7, delay: reduce ? 0 : delay, ease } });

  return <MotionConfig reducedMotion={reduce ? 'always' : 'never'}><div className="torq-home" id="th-top">
    <a href="#th-main" className="th-skip-link">Skip to content</a>
    <header className="th-header"><div className="th-container th-header-inner"><Link to="/" className="th-brand-link" aria-label="TORQ home"><Brand /></Link><nav className="th-desktop-nav" aria-label="Main navigation"><a href="#th-product">Product</a><a href="#th-how">How it works</a><a href="#th-connect">Connections</a></nav><div className="th-header-actions"><Link to="/login" className="th-sign-in">Log in <ArrowUpRight size={14} /></Link><Link to="/dashboard" className="th-button th-button-black th-header-cta">Open workspace <ArrowUpRight size={15} /></Link><button type="button" ref={menuRef} className="th-menu-toggle" onClick={() => setMobileOpen(value => !value)} aria-label={mobileOpen ? 'Close navigation' : 'Open navigation'} aria-expanded={mobileOpen} aria-controls="th-mobile-nav">{mobileOpen ? <X size={23} /> : <Menu size={23} />}</button></div></div>
      <nav className={`th-mobile-nav${mobileOpen ? ' is-open' : ''}`} id="th-mobile-nav" aria-label="Mobile navigation" inert={!mobileOpen}>{[['Product', '#th-product'], ['How it works', '#th-how'], ['Connections', '#th-connect'], ['Questions', '#th-questions']].map(([label, href]) => <a key={href} href={href} onClick={() => setMobileOpen(false)}>{label}<ArrowUpRight size={17} /></a>)}<Link to="/dashboard" onClick={() => setMobileOpen(false)}>Open workspace<ArrowUpRight size={17} /></Link></nav>
    </header>
    <main id="th-main" tabIndex={-1}>
      <section className="th-hero th-container" aria-labelledby="th-hero-title"><div className="th-hero-copy"><motion.div {...heroMotion(0)}><Eyebrow>SELF-HOSTED CRM</Eyebrow></motion.div><motion.h1 id="th-hero-title" {...heroMotion(0.07)}>Self-hosted<br /><span>CRM for teams.</span></motion.h1><motion.p className="th-hero-description" {...heroMotion(0.14)}>Manage contacts, companies, leads, deals, tasks, and projects in one place.<br className="th-desktop-break" /> Run TORQ on your own infrastructure.</motion.p><motion.div className="th-hero-actions" {...heroMotion(0.21)}><Link to="/dashboard" className="th-button th-button-orange">Open workspace <ArrowUpRight size={18} /></Link><a href="#th-product" className="th-text-link">View product <ArrowDown size={16} /></a></motion.div><motion.div className="th-hero-footnote" {...heroMotion(0.28)}><span><Check size={13} />Self-hosted</span><span><Check size={13} />CRM and project tracking</span></motion.div></div><motion.div className="th-hero-visual" {...heroMotion(0.15)}><HeroIllustration /></motion.div><div className="th-hero-bottom"><span>TORQ / CUSTOMER RELATIONSHIP MANAGEMENT</span><a href="#th-product">PRODUCT OVERVIEW <ArrowDown size={13} /></a></div></section>

      <div className="th-manifesto" aria-label="Contacts, deals, and tasks"><div className="th-container"><span>Contacts.</span><span className="th-band-separator" aria-hidden="true">/</span><span>Deals.</span><span className="th-band-separator" aria-hidden="true">/</span><span>Tasks.</span><ArrowUpRight /></div></div>

      <section className="th-product-section th-container" id="th-product" aria-labelledby="th-product-title"><Reveal><Eyebrow>01 / PRODUCT</Eyebrow><div className="th-section-heading"><h2 id="th-product-title">Customer records<br />and sales activity.</h2><p>View contacts, companies, deal stages, and assigned work together. Use the preview to see the main CRM views.</p></div></Reveal><Reveal><ProductPreview /></Reveal><div className="th-feature-columns">{[{icon:Contact,title:'Contacts & companies',copy:'Store contact details, company information, and related activity.'},{icon:LayoutGrid,title:'Deal pipeline',copy:'Review deal values and stages from discovery through proposal.'},{icon:CheckCheck,title:'Tasks & follow-ups',copy:'Assign owners, set due dates, and review upcoming work.'}].map((item,index)=><Reveal key={item.title} delay={index*0.06}><div className="th-feature-top"><span>0{index+1}</span><item.icon size={21} strokeWidth={1.5} /></div><h3>{item.title}</h3><p>{item.copy}</p></Reveal>)}</div></section>

      <section className="th-how-section" id="th-how" aria-labelledby="th-how-title"><div className="th-container"><Reveal><Eyebrow>HOW IT WORKS</Eyebrow><div className="th-section-heading"><h2 id="th-how-title">A straightforward<br />sales workflow.</h2><p>Keep customer records current, update deal stages, and assign follow-up tasks.</p></div></Reveal><div className="th-step-grid">{[{title:'Add customer records',copy:'Create contact and company records with the relevant details.'},{title:'Update deal stages',copy:'Record the deal value and move each opportunity through your pipeline.'},{title:'Assign follow-up tasks',copy:'Set an owner and due date for the next action.'}].map((item,index)=><Reveal className="th-step" key={item.title} delay={index*0.08}><div className="th-step-label"><span>STEP 0{index+1}</span><ArrowUpRight size={17} /></div><StepIllustration step={index} /><h3>{item.title}</h3><p>{item.copy}</p></Reveal>)}</div><div className="th-how-bottom"><span>Contacts, deals, and assigned tasks.</span><Link to="/dashboard">Open workspace <ArrowRight size={17} /></Link></div></div></section>

      <section className="th-connect-section th-container" id="th-connect" aria-labelledby="th-connect-title"><Reveal className="th-connect-visual"><ConnectedIllustration /></Reveal><Reveal className="th-connect-copy" delay={0.08}><Eyebrow>02 / CONNECTIONS</Eyebrow><h2 id="th-connect-title">Integrate with<br /><span>your tools.</span></h2><p>Use APIs, agent tools, workflows, and webhooks to connect TORQ with your existing systems. Configure access and permissions in the workspace.</p><div className="th-connection-links"><Link to="/agent-console"><Code2 size={18} />Agent console<ArrowUpRight size={17} /></Link><Link to="/webhooks"><Webhook size={18} />Webhooks<ArrowUpRight size={17} /></Link><Link to="/workflows"><GitBranch size={18} />Workflows<ArrowUpRight size={17} /></Link></div></Reveal></section>

      <section className="th-questions th-container" id="th-questions" aria-labelledby="th-questions-title"><Reveal className="th-questions-intro"><Eyebrow>03 / FAQ</Eyebrow><h2 id="th-questions-title">Frequently<br />asked questions.</h2><p>Product, access, and hosting.</p></Reveal><div className="th-faq-list">{questions.map((item,index)=><Reveal key={item.question} delay={index*0.03}><article className={`th-faq${openQuestion===index?' is-open':''}`}><h3><button type="button" onClick={()=>setOpenQuestion(openQuestion===index?null:index)} aria-expanded={openQuestion===index} aria-controls={`th-answer-${index}`} id={`th-question-${index}`}><span>{item.question}</span><Plus size={19} aria-hidden="true" /></button></h3><div className="th-faq-answer" id={`th-answer-${index}`} role="region" aria-labelledby={`th-question-${index}`} inert={openQuestion!==index}><div><p>{item.answer}</p></div></div></article></Reveal>)}</div></section>

      <section className="th-final-cta" aria-labelledby="th-cta-title"><div className="th-container"><Reveal><Eyebrow>TORQ CRM</Eyebrow><h2 id="th-cta-title">Open your<br />workspace.</h2><div className="th-final-actions"><Link to="/dashboard" className="th-button th-button-black">Open your workspace<ArrowUpRight size={19} /></Link><Link to="/login" className="th-text-link">Already here? Log in <ArrowRight size={16} /></Link></div></Reveal><svg className="th-cta-arrow" viewBox="0 0 300 300" fill="none" aria-hidden="true"><circle cx="150" cy="150" r="146" stroke="#000" /><path d="M76 224 224 76M76 76h148v148" stroke="#000" strokeWidth="22" /><path d="M51 248 248 51" stroke="#fff" strokeWidth="2" /></svg></div></section>
    </main>
    <footer className="th-footer"><div className="th-container"><div className="th-footer-top"><div><Link to="/" className="th-brand-link" aria-label="TORQ home"><Brand inverse /></Link><p>Contacts, deals, and tasks.</p></div><nav aria-label="Footer navigation"><a href="#th-product">Product</a><a href="#th-how">How it works</a><a href="#th-questions">Questions</a><Link to="/login">Log in<ArrowUpRight size={14} /></Link></nav><a className="th-back-top" href="#th-top">Back to top<ArrowUpRight size={18} /></a></div><div className="th-footer-bottom"><span>&copy; {year} TORQ</span><span>Self-hosted CRM</span><span>Your infrastructure. Your workspace.</span></div></div></footer>
  </div></MotionConfig>;
}
