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
      setImportSuccess('Import complete. List refreshed.');
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
        <h1>{title}</h1>
        <p>{description}</p>
      </div>
      <div className="au-header-actions">
        {onExport && (
          <button className="au-button" onClick={onExport}>
            <ArrowDownToLine size={14} /> Export
          </button>
        )}
        {onImport && (
          <>
            <input type="file" ref={fileInputRef} accept=".csv" hidden onChange={e => setImportFile(e.target.files?.[0] || null)} disabled={importing} />
            <button className="au-button" onClick={() => fileInputRef.current?.click()} disabled={importing}>
              <ArrowUpFromLine size={14} /> Import
            </button>
            {importFile && (
              <div className="cm-import-status">
                <span>{importFile.name}</span>
                <button className="au-button cm-upload-button" onClick={handleImport} disabled={importing}>
                  {importing ? <Loader2 size={12} className="loading-spinner" /> : 'Upload'}
                </button>
                {importError && <span className="is-error">{importError}</span>}
                {importSuccess && <span className="is-success">{importSuccess}</span>}
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
