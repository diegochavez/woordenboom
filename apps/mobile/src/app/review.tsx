import { router, useLocalSearchParams } from 'expo-router';
import { useCallback } from 'react';
import { FullscreenWordCard, WordMeaning } from '@/components/FullscreenWordCard';
import { ReviewInterval } from '@/components/ReviewTiming';
import { StackState } from '@/components/StackState';
import { useWordStack } from '@/hooks/useWordStack';
import { postJSON } from '@/lib/api';

type Question = { meanings?: WordMeaning[]; id: string; wordId: string; topic: string; word: string; example: string; translation: string };
type Session = { id: string; questions: Question[] };
export default function ReviewScreen() {
  const params = useLocalSearchParams<{ topic?: string; topics?: string }>();
  const load = useCallback(async () => {
    const topicSlugs = params.topics?.split(',').filter(Boolean) ?? [];
    const result = await postJSON<Session>('/sessions/next', {
      topicSlugs: topicSlugs.length ? topicSlugs : [params.topic || 'general-vocabulary'],
      cefrLevels: [], mode: 'balanced',
    });
    return { id: result.id, words: result.questions };
  }, [params.topic, params.topics]);
  const stack = useWordStack(load);
  const home = () => router.dismissTo('/home');
  if (stack.error) return <StackState title="Couldn’t load your stack." message={stack.error} actionLabel="Try again" onAction={stack.retry} secondaryLabel="Back to dashboard" onSecondary={home} />;
  if (stack.empty) return <StackState title="You’re up to date." message="Your words are scheduled for later. Come back when they’re due, or choose another category to learn something new." actionLabel="Choose categories" onAction={() => router.dismissTo('/topics')} secondaryLabel="Back to dashboard" onSecondary={home} />;
  if (stack.complete) return <StackState complete title="Stack complete." message="Your progress is saved. Keep practicing the words you’re still learning." total={stack.total} known={stack.known} actionLabel="Back to dashboard" onAction={home} secondaryLabel="Practice again" onSecondary={stack.retry} />;
  const question = stack.word;
  if (!question) return <StackState loading title="Getting ready." message="Preparing your word stack." secondaryLabel="Back to dashboard" onSecondary={home} />;
  const mark = (known: boolean, reviewInterval: ReviewInterval = 'auto') => stack.mark(() => postJSON<{ correctCount: number }>(`/sessions/${stack.stack?.id}/answer`, {
    questionId: question.id, wordId: question.wordId, answer: known ? 'Already Knew' : 'Should Learn', reviewInterval,
  }), known);
  return <FullscreenWordCard meanings={question.meanings} wordId={question.wordId} example={question.example} feedback={stack.feedback}
    flipped={stack.flipped} isSubmitting={stack.submitting} onAlreadyKnew={(interval) => mark(true, interval)} onShouldLearn={() => mark(false)}
    onBack={() => router.canGoBack() ? router.back() : home()} onFlip={stack.flip}
    progressLabel={`${stack.index + 1} / ${stack.total}`} topic={question.topic} translation={question.translation} word={question.word} />;
}
