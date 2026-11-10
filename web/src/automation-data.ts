import { useEffect, useState } from 'react';

export function useDebouncedSearch(value: string) {
  const [settled, setSettled] = useState(value);
  useEffect(() => { const timer = window.setTimeout(() => setSettled(value.trim()), 250); return () => window.clearTimeout(timer); }, [value]);
  return settled;
}
export function objectConfig(value: string): Record<string, unknown> {
  const result: unknown = JSON.parse(value);
  if (!result || typeof result !== 'object' || Array.isArray(result)) throw new Error('Enter a JSON object, for example {}.');
  return result as Record<string, unknown>;
}
export function automationDate(value: string | null | undefined) {
  if (!value || value.startsWith('0001-')) return '\u2014';
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? '\u2014' : date.toLocaleString(undefined, { month:'short',day:'numeric',hour:'numeric',minute:'2-digit' });
}
export function errorMessage(error: unknown) { return error instanceof Error ? error.message : 'The request could not be completed.'; }
