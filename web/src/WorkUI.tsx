import { useId, useRef, useState, type ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { ArrowDownToLine, ArrowUpFromLine, ChevronDown, Loader2, Search } from 'lucide-react';
import { api } from './api';
import { useAdminData } from './admin-data';
import { useDebouncedSearch } from './automation-data';
import './WorkManagement.css';

export function WorkHeader({
  title,
  description,
  children,
  onExport,
  onImport,
}: { title: 'Projects' | 'Tasks'; description: string; children: ReactNode; onExport?: () => void; onImport?: (file: File) => Promise<void> }) {
  const [importFile, setImportFile] = useState<File | null>(null);
  const [importing, setImporting] = useState(false);
  const [importError, setImportError] = useState<string | null>(null);
  const [importSuccess, setImportSuccess] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const handleImport = async () => {
    if (!importFile || importing || !onImport) return;
    setImporting(true);
    setImportError(null);
    setImportSuccess(null);
    try {
      await onImport(importFile);
      setImportSuccess('Import complete — list refreshed.');
      setImportFile(null);
      if (fileInputRef.current) fileInputRef.current.value = '';
    } catch (e) {
      setImportError(e instanceof Error ? e.message : 'Import failed');
    } finally {
      setImporting(false);
    }
  };

  return (
    <>
      <header className="au-header" aria-describedby="wm-header-desc">
        <span id="wm-header-desc" className="sr-only">{description}</span>
        <div>
          <h1>{title}</h1>
        </div>
        <div className="au-header-actions">
          {onExport && (
            <button className="crm-button" onClick={onExport}>
              <ArrowDownToLine size={14} /> Export
            </button>
          )}
          {onImport && (
            <>
              <input type="file" ref={fileInputRef} accept=".csv" hidden onChange={e => setImportFile(e.target.files?.[0] || null)} disabled={importing} />
              <button className="crm-button" onClick={() => fileInputRef.current?.click()} disabled={importing}>
                <ArrowUpFromLine size={14} /> Import
              </button>
              {importFile && (
                <div className="wm-import-status">
                  <span>{importFile.name}</span>
                  <button className="wm-import-upload crm-button" onClick={handleImport} disabled={importing}>
                    {importing ? <Loader2 size={12} className="loading-spinner" /> : 'Upload'}
                  </button>
                  {importError && <span className="wm-import-error">{importError}</span>}
                  {importSuccess && <span className="wm-import-success">{importSuccess}</span>}
                </div>
              )}
            </>
          )}
          {children}
        </div>
      </header>
    </>
  );
}
export function WorkEmpty({title,description,children}:{title:string;description:string;children?:ReactNode}){return <div className="wm-empty"><strong>{title}</strong><span>{description}</span>{children}</div>;}
export function WorkPriority({value}:{value:string}){return <span className={`wm-priority wm-priority-${value}`}>{['high','urgent'].includes(value)&&<i/>}{value||'Not set'}</span>;}
export function WorkProgress({value,label='Progress'}:{value:number;label?:string}){const safe=Number.isFinite(value)?Math.max(0,Math.min(100,value)):0;return <div className="wm-progress"><div><span>{label}</span><strong>{safe}%</strong></div><div role="meter" aria-label={label} aria-valuemin={0} aria-valuemax={100} aria-valuenow={safe}><span style={{width:`${safe}%`}}/></div></div>;}
export function RecordPicker({kind,value,onChange,disabled=false,label}:{kind:'company'|'project';value:string;onChange:(id:string)=>void;disabled?:boolean;label:string}){
 const id=useId();const [query,setQuery]=useState('');const search=useDebouncedSearch(query);const path=kind==='company'?'companies':'projects';
 const records=useAdminData(()=>api.get<{items:{id:string;name:string}[];total:number}>(`/${path}`,{params:{q:search,limit:50,sort:'name',order:'asc'}}),`picker:${path}:${search}`);const selected=useAdminData<{id:string;name:string}|null>(()=>value?api.get(`/${path}/${encodeURIComponent(value)}`):Promise.resolve(null),`picker-selected:${path}:${value}`);const items=records.data?.items??[];
 return <div className="form-group wm-record-picker"><label className="form-label" htmlFor={id}>{label}</label><div className="wm-picker-search"><Search size={14}/><input type="search" value={query} onChange={e=>setQuery(e.target.value)} placeholder={`Find ${kind==='company'?'companies':'projects'}`} aria-label={`Search ${label.toLowerCase()}`} disabled={disabled}/></div><select id={id} value={value} onChange={e=>onChange(e.target.value)} disabled={disabled} className="form-select"><option value="">None</option>{value&&!items.some(item=>item.id===value)&&<option value={value}>{selected.data?.name??'Selected record'}</option>}{items.map(item=><option key={item.id} value={item.id}>{item.name}</option>)}</select>{records.loading&&<p className="form-help">Loading matches...</p>}{records.error&&<p className="form-help" role="alert">Could not load choices. Change the search to try again.</p>}{(records.data?.total??0)>items.length&&<p className="form-help">Search to narrow {records.data?.total} records.</p>}</div>;
}
export function RecordFilter({kind,value,onChange}:{kind:'company'|'project';value:string;onChange:(value:string)=>void}){const ref=useRef<HTMLDetailsElement>(null);const label=kind==='company'?'Company':'Project';return <details ref={ref} className="wm-record-filter"><summary>{label}{value&&<i/>}<ChevronDown size={13}/></summary><div><RecordPicker kind={kind} value={value} onChange={id=>{onChange(id);if(ref.current)ref.current.open=false;}} label={label}/></div></details>;}
export function WorkSelection({count,onClear,children}:{count:number;onClear:()=>void;children:ReactNode}){return <div className="wm-selection" role="status"><strong>{count} selected</strong><div>{children}<button onClick={onClear}>Clear selection</button></div></div>;}
