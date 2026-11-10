export function crmMoney(amount: number, currency: string) {
  try { return new Intl.NumberFormat(undefined, { style: 'currency', currency, minimumFractionDigits: 0, maximumFractionDigits: 2 }).format(amount); }
  catch { return `${currency} ${amount.toLocaleString()}`; }
}

export function crmValueSummary(items: { amount: number | null; currency: string }[]) {
  const values = new Map<string, number>();
  for (const item of items) values.set(item.currency || 'USD', (values.get(item.currency || 'USD') || 0) + (item.amount || 0));
  return values.size ? Array.from(values, ([currency, amount]) => crmMoney(amount, currency)).join(' / ') : '\u2014';
}

export function exportCrmPage<T extends { id: string }>(noun: string, records: T[], fields: { key: keyof T; label: string }[]) {
  const cell = (value: unknown) => { const raw = String(value ?? ''); return `"${(/^[\s]*[=+@-]/.test(raw) ? "'" + raw : raw).replaceAll('"', '""')}"`; };
  const csv = '\uFEFF' + [fields.map(field => cell(field.label)).join(','), ...records.map(record => fields.map(field => cell(record[field.key])).join(','))].join('\r\n');
  const url = URL.createObjectURL(new Blob([csv], { type: 'text/csv;charset=utf-8' }));
  const anchor = document.createElement('a'); anchor.href = url; anchor.download = `torq-${noun}-${new Date().toISOString().slice(0, 10)}.csv`; document.body.appendChild(anchor); anchor.click(); anchor.remove(); setTimeout(() => URL.revokeObjectURL(url), 1000);
}
