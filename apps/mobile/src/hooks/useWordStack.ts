import * as Haptics from 'expo-haptics';
import { notifyProgressChanged } from '@/lib/progressEvents';
import { useEffect, useRef, useState } from 'react';

export type WordStack<T> = { id?: string; words: T[] };
// A synchronous lock prevents two taps in the same render from recording two answers.
// Generation checks discard network responses after leaving or reloading a stack.
export function useWordStack<T>(load: () => Promise<WordStack<T>>) {
  const [stack, setStack] = useState<WordStack<T> | null>(null);
  const [index, setIndex] = useState(0);
  const [known, setKnown] = useState(0);
  const [flipped, setFlipped] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState('');
  const [feedback, setFeedback] = useState('');
  const [attempt, setAttempt] = useState(0);
  const generation = useRef(0);
  const locked = useRef(false);
  useEffect(() => {
    const current = ++generation.current;
    Promise.resolve().then(async () => {
      if (generation.current !== current) return;
      locked.current = false;
      setStack(null); setIndex(0); setKnown(0); setFlipped(false);
      setSubmitting(false); setError(''); setFeedback('');
      const result = await load();
      if (generation.current === current) setStack(result);
    }).catch(() => {
      if (generation.current === current) setError('We could not load these words. Check your connection and try again.');
    });
    return () => { generation.current = current + 1; };
  }, [load, attempt]);

  async function mark(save: () => Promise<{ correctCount?: number } | void>, isKnown: boolean) {
    if (locked.current || !stack?.words[index]) return;
    locked.current = true;
    const current = generation.current;
    setSubmitting(true); setFeedback('Saving…');
    try {
      const result = await save();
      notifyProgressChanged();
      if (generation.current !== current) return;
      setKnown((value) => result?.correctCount ?? value + (isKnown ? 1 : 0));
      setIndex((value) => value + 1); setFlipped(false); setFeedback('');
      void Haptics.selectionAsync().catch(() => {});
    } catch {
      if (generation.current === current) setFeedback('Your answer was not saved. Check your connection and tap your choice again.');
    } finally {
      if (generation.current === current) { locked.current = false; setSubmitting(false); }
    }
  }
  const total = stack?.words.length ?? 0;
  return { stack, word: stack?.words[index], index, total, known, flipped, submitting, error, feedback,
    complete: Boolean(stack && total > 0 && index >= total), empty: Boolean(stack && total === 0),
    flip: () => { if (!locked.current) setFlipped((value) => !value); },
    retry: () => setAttempt((value) => value + 1), mark };
}
