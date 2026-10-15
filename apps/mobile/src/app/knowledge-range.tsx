import { router, useLocalSearchParams } from 'expo-router';
import { useCallback } from 'react';
import { FullscreenWordCard, WordMeaning } from '@/components/FullscreenWordCard';
import { ReviewInterval } from '@/components/ReviewTiming';
import { StackState } from '@/components/StackState';
import { useWordStack } from '@/hooks/useWordStack';
import { getJSON, postJSON } from '@/lib/api';

type RangeWord = { meanings?: WordMeaning[]; cefrLevel?: string; englishExample: string; example: string; id: string; topic: string; translation: string; word: string };
export default function KnowledgeRangeScreen() {
  const params = useLocalSearchParams<{ start?: string }>();
  const parsedStart = Number(params.start ?? '1');
  const start = Number.isInteger(parsedStart) && parsedStart > 0 ? parsedStart : 1;
  const load = useCallback(() => getJSON<{ words: RangeWord[] }>(`/knowledge-map/ranges/${start}`), [start]);
  const stack = useWordStack(load);
  const map = () => router.dismissTo('/knowledge-map');
  if (stack.error) return <StackState title="Couldn’t load this range." message={stack.error} actionLabel="Try again" onAction={stack.retry} secondaryLabel="Back to map" onSecondary={map} />;
  if (stack.empty) return <StackState title="No words yet." message="Practice words aren’t available for this range yet. Choose another range on your map." actionLabel="Back to map" onAction={map} />;
  if (stack.complete) return <StackState complete title="Range complete." message="Your progress is saved on your knowledge map." total={stack.total} known={stack.known} actionLabel="Back to map" onAction={map} secondaryLabel="Practice again" onSecondary={stack.retry} />;
  const word = stack.word;
  if (!word) return <StackState loading title="Getting ready." message="Loading your word cards." secondaryLabel="Back to map" onSecondary={map} />;
  const mark = (known: boolean, reviewInterval: ReviewInterval = 'auto') => stack.mark(async () => { await postJSON(`/knowledge-map/words/${word.id}/mark`, { action: known ? 'already_knew' : 'should_learn', reviewInterval }); }, known);
  return <FullscreenWordCard meanings={word.meanings} wordId={word.id} cefrLevel={word.cefrLevel} example={[word.example, word.englishExample].filter(Boolean).join('\n')}
    feedback={stack.feedback} flipped={stack.flipped} isSubmitting={stack.submitting} onAlreadyKnew={(interval) => mark(true, interval)} onShouldLearn={() => mark(false)}
    onBack={() => router.canGoBack() ? router.back() : map()} onFlip={stack.flip}
    progressLabel={`${stack.index + 1} / ${stack.total}`} topic={word.topic} translation={word.translation} word={word.word} />;
}
