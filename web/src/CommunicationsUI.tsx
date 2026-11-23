import { useRef, useState, type ReactNode } from 'react';
import { ArrowDownToLine, ArrowUpFromLine, Loader2 } from 'lucide-react';
import './Communications.css';
export function CommunicationsHeader({
  title,
  description,
  children,
  onExport,
  onImport,
}: { title: string; description: string; children?: ReactNode; onExport?: () => void; onImport?: (file: File) => Promise<void> }) {
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
    <header className="au-header cm-header">
      <div>
        <span className="au-eyebrow">WORKSPACE</span>
        <h1>{title}</h1>
        <p>{description}</p>
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
              <div className="crm-import-status" style={{ display: 'flex', alignItems: 'center', gap: 8, fontSize: 11, color: '#775334' }}>
                <span>{importFile.name}</span>
                <button className="crm-button" onClick={handleImport} disabled={importing} style={{ padding: '4px 8px', fontSize: 11, minHeight: 28 }}>
                  {importing ? <Loader2 size={12} className="loading-spinner" /> : 'Upload'}
                </button>
                {importError && <span style={{ color: '#b8562e' }}>{importError}</span>}
                {importSuccess && <span style={{ color: '#46673a' }}>{importSuccess}</span>}
              </div>
            )}
          </>
        )}
        {children}
      </div>
    </header>
  );
}
export function CommunicationsEmpty({icon,title,description,children}:{icon:ReactNode;title:string;description:string;children?:ReactNode}){return <div className="cm-empty"><span>{icon}</span><h3>{title}</h3><p>{description}</p>{children}</div>;}
