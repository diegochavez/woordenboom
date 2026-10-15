import { StatusBar } from 'expo-status-bar';
import { PropsWithChildren } from 'react';
import { Animated, ScrollView, StyleSheet, Text, useWindowDimensions, View } from 'react-native';
import { SafeAreaView, useSafeAreaInsets } from 'react-native-safe-area-context';
import { CircleButton, OutlineButton } from '@/components/OutlineButton';
import { ReviewInterval, ReviewTimingControl, useReviewTiming } from '@/components/ReviewTiming';
import { motion, useReveal } from '@/hooks/useMotion';
import { colors, layout, spacing, type } from '@/lib/theme';

export type WordMeaning = { english: string; dutch: string; partOfSpeech?: string };

type FullscreenWordCardProps = {
  meanings?: WordMeaning[];
  cefrLevel?: string; example?: string; feedback?: string;
  flipped: boolean; isSubmitting?: boolean; onAlreadyKnew: (interval?: ReviewInterval) => void; onBack: () => void;
  onFlip: () => void; onShouldLearn: () => void; progressLabel?: string;
  topic: string; translation: string; word: string; wordId: string;
};

function WordReveal({ children }: PropsWithChildren) {
  const opacity = useReveal(motion.word);
  return <Animated.View testID="word-reveal" style={[styles.wordBlock, { opacity }]}>{children}</Animated.View>;
}

export function FullscreenWordCard({ cefrLevel, example, feedback, meanings = [], flipped, isSubmitting, onAlreadyKnew, onBack, onFlip, onShouldLearn, progressLabel, topic, translation, word, wordId }: FullscreenWordCardProps) {
  const { width, height } = useWindowDimensions();
  const insets = useSafeAreaInsets();
  const reviewTiming = useReviewTiming(wordId);
  const displayWord = flipped ? translation : word;
  // Both sides use the same size, even when the translation is longer.
  const longestWord = Math.max(...`${word} ${translation}`.split(/\s+/).map((part) => part.length), 1);
  const wordSize = Math.min(46, Math.max(28, Math.floor((Math.min(width, 600) - spacing.xl * 2) / (longestWord * 0.56))));
  // Anchor from the top, independent of example length or the controls below.
  // Large metadata may expand this region instead of clipping accessible text.
  const wordOffset = Math.max(112, Math.min(224, (height - insets.top - insets.bottom) * 0.26));
  const revealKey = `${progressLabel}:${word}:${flipped}`;
  return (
    <SafeAreaView style={styles.screen}>
      <StatusBar style="light" />
      <View style={styles.inset}>
        <View style={styles.topBar}>
          <CircleButton inverse accessibilityLabel="Leave word stack" icon="arrow-left" onPress={onBack} />
          <Text style={styles.progress}>{progressLabel}</Text>
        </View>
        <ScrollView key={revealKey} style={styles.scroll} contentContainerStyle={styles.content} showsVerticalScrollIndicator={false}>
          <View style={[styles.metadata, { minHeight: wordOffset }]}>
            <Text style={styles.language}>{flipped ? 'English' : 'Dutch'}</Text>
            <Text style={styles.topic}>{[cefrLevel, topic].filter(Boolean).join(' · ')}</Text>
          </View>
          <WordReveal>
            <Text testID="study-word" selectable style={[styles.word, { fontSize: wordSize }]}>{displayWord}</Text>
            {flipped ? <Text style={styles.original}>{word}</Text> : null}
            {flipped && meanings.length ? <View testID="dutch-definitions" style={styles.definitions}>
              <Text accessibilityRole="header" style={styles.definitionHeading}>Betekenis in het Nederlands</Text>
              {meanings.map((meaning, index) => <View key={`${index}:${meaning.english}`} style={styles.meaning}>
                {meanings.length > 1 ? <Text style={styles.senseLabel}>{meaning.english}</Text> : null}
                <Text selectable accessibilityLanguage="nl-NL" style={styles.definition}>{meaning.dutch}</Text>
              </View>)}
            </View> : null}
            {flipped && example ? <Text selectable style={styles.example}>{example}</Text> : null}
          </WordReveal>
        </ScrollView>
        <View style={styles.footer}>
          {feedback ? <Text accessibilityLiveRegion="polite" style={styles.feedback}>{feedback}</Text> : null}
          <View style={styles.secondaryAction}>
            {flipped ? <OutlineButton inverse disabled={isSubmitting} label="Show Dutch" onPress={onFlip} variant="text" /> : null}
          </View>
          {flipped ? <ReviewTimingControl key={wordId} word={word} {...reviewTiming} disabled={isSubmitting}
            onRetry={reviewTiming.retry} onConfirm={(interval) => { reviewTiming.choose(interval); onAlreadyKnew(interval); }} /> : null}
          {flipped ? (
            <View style={styles.actions}>
              <OutlineButton inverse disabled={isSubmitting} label="Not yet" accessibilityLabel="Not yet, keep learning this word" onPress={onShouldLearn} style={styles.action} />
              <OutlineButton inverse disabled={isSubmitting} label="Know" accessibilityLabel="Know this word" onPress={() => onAlreadyKnew(reviewTiming.selected)} style={styles.action} variant="primary" />
            </View>
          ) : <OutlineButton inverse label="Flip" onPress={onFlip} />}
        </View>
      </View>
    </SafeAreaView>
  );
}
const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: colors.black },
  inset: { ...layout.page, flex: 1, paddingTop: spacing.sm, paddingBottom: spacing.xl, gap: 0 },
  topBar: { alignItems: 'center', flexDirection: 'row', justifyContent: 'space-between', minHeight: 48 },
  progress: { ...type.label, color: colors.onDark, fontVariant: ['tabular-nums'] },
  scroll: { flex: 1 },
  content: { paddingTop: spacing.xl, paddingBottom: spacing.xxl },
  metadata: { paddingBottom: spacing.xl },
  language: { ...type.label, color: colors.onDark },
  topic: { ...type.caption, color: colors.onDarkMuted, marginTop: spacing.xs },
  wordBlock: { gap: spacing.lg },
  word: { ...type.display, color: colors.onDark, lineHeight: 54 },
  original: { ...type.body, color: colors.onDarkMuted },
  definitions: { gap: spacing.lg, marginTop: spacing.sm },
  definitionHeading: { ...type.caption, color: colors.onDarkMuted },
  meaning: { gap: spacing.xs },
  senseLabel: { ...type.label, color: colors.onDarkMuted },
  definition: { ...type.body, color: colors.onDark, fontSize: 18, lineHeight: 28 },
  example: { ...type.body, color: colors.onDark, fontSize: 18, lineHeight: 28, marginTop: spacing.xl },
  footer: { gap: spacing.sm },
  secondaryAction: { minHeight: 44 },
  actions: { flexDirection: 'row', gap: spacing.md }, action: { flex: 1 },
  feedback: { ...type.caption, color: colors.onDarkMuted, textAlign: 'center', paddingVertical: spacing.sm },
});
