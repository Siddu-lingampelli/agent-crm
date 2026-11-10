import { useEffect, useEffectEvent } from 'react';
import { useSearchParams } from 'react-router-dom';

/** Open existing record editors from dashboard/search links without adding duplicate forms. */
export function useRecordRoute<T>({ getRecord, onCreate, onRecord, onError }: {
  getRecord: (id: string) => Promise<T>;
  onCreate: () => void;
  onRecord: (record: T) => void;
  onError: (message: string) => void;
}) {
  const [params, setParams] = useSearchParams();
  const create = params.get('create');
  const record = params.get('record');
  const openCreate = useEffectEvent(onCreate);
  const openRecord = useEffectEvent(onRecord);
  const reportError = useEffectEvent(onError);

  useEffect(() => {
    const controller = new AbortController();
    const consume = () => setParams(previous => {
      const next = new URLSearchParams(previous);
      next.delete('create'); next.delete('record');
      return next;
    }, { replace: true });
    if (create === '1') { openCreate(); consume(); }
    else if (record) {
      getRecord(record).then(data => {
        if (!controller.signal.aborted) { openRecord(data); consume(); }
      }).catch(error => {
        if (!controller.signal.aborted) { reportError(error instanceof Error ? error.message : 'Could not open this record.'); consume(); }
      });
    }
    return () => { controller.abort(); };
  }, [create, record, getRecord, setParams]);
}
