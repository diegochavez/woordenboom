import { useEffect, useState } from 'react';
import { Animated, Modal, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { OutlineButton } from '@/components/OutlineButton';
import { getJSON } from '@/lib/api';
import { LineIcon } from '@/components/LineIcon';
import { motion, useReveal } from '@/hooks/useMotion';
import { colors, radii, spacing, type } from '@/lib/theme';

export type ReviewInterval = 'auto' | '10m' | '1d' | '2d' | '3d' | '7d' | '14d' | '30d' | '60d' | '90d' | '180d' | '365d' | 'paused';
type Timing = { rememberedCount: number; recommendedInterval: ReviewInterval; nextReviewAt: string | null; paused: boolean };
const options: { value: ReviewInterval; label: string }[] = [
  { value: '10m', label: 'In 10 minutes' }, { value: '1d', label: 'In 1 day' },
  { value: '2d', label: 'In 2 days' }, { value: '3d', label: 'In 3 days' },
  { value: '7d', label: 'In a week' }, { value: '14d', label: 'In 2 weeks' },
  { value: '30d', label: 'In a month' }, { value: '60d', label: 'In 2 months' },
  { value: '90d', label: 'In 3 months' }, { value: '180d', label: 'In 6 months' },
  { value: '365d', label: 'In a year' }, { value: 'paused', label: 'Pause reviews' },
];
export function useReviewTiming(wordId: string) {
  const [state, setState] = useState<{ wordId: string; timing?: Timing; error?: boolean } | null>(null);
  const [choice, setChoice] = useState<{ wordId: string; value: ReviewInterval } | null>(null);
  const [attempt, setAttempt] = useState(0);
  useEffect(() => {
    let active = true;
    getJSON<Timing>(`/words/${encodeURIComponent(wordId)}/review-timing`).then((timing) => {
      if (typeof timing.rememberedCount !== 'number' || !options.some((option) => option.value === timing.recommendedInterval)) throw new Error('Invalid review timing');
      if (active) setState({ wordId, timing });
    }).catch(() => { if (active) setState({ wordId, error: true }); });
    return () => { active = false; };
  }, [wordId, attempt]);
  const current = state?.wordId === wordId ? state : null;
  const selected = choice?.wordId === wordId ? choice.value : current?.timing?.paused ? 'paused' : 'auto';
  return { timing: current?.timing, error: current?.error, selected,
    choose: (value: ReviewInterval) => setChoice({ wordId, value }),
    retry: () => { setState(null); setAttempt((value) => value + 1); },
  };
}

type Props = {
  word: string; timing?: Timing; error?: boolean; selected: ReviewInterval; disabled?: boolean;
  onRetry: () => void; onConfirm: (value: ReviewInterval) => void;
};
export function ReviewTimingControl({ word, timing, error, selected, disabled, onRetry, onConfirm }: Props) {
  const [visible, setVisible] = useState(false);
  const [draft, setDraft] = useState<ReviewInterval>('1d');
  const insets = useSafeAreaInsets();
  const effective = selected === 'auto' ? timing?.recommendedInterval ?? '1d' : selected;
  const label = options.find((option) => option.value === effective)?.label ?? 'In 1 day';
  // Keep legacy two-day recommendations usable during API/client rollouts.
  const visibleOptions = options.filter((option) => option.value !== '2d' || effective === '2d');
  const count = timing?.rememberedCount ?? 0;
  const close = () => setVisible(false);
  function open() { setDraft(effective); setVisible(true); }
  return <>
    <View style={styles.summary}>
      <Text style={styles.count}>{timing ? `Remembered ${count} ${count === 1 ? 'time' : 'times'}` : 'Review timing'}</Text>
      {error ? <OutlineButton inverse variant="text" label="Retry timing" onPress={onRetry} /> :
        <Pressable accessibilityRole="button" accessibilityLabel="Change review timing" disabled={!timing || disabled}
          accessibilityState={{ disabled: !timing || disabled }} onPress={open} style={styles.timingButton}>
          <LineIcon name="clock" size={20} color={colors.white} />
          <Text style={styles.timingLabel}>{!timing ? 'Loading…' : effective === 'paused' ? 'Reviews paused' : `Review ${label.toLowerCase()}`}</Text>
          {timing ? <Text style={styles.change}>Change</Text> : null}
        </Pressable>}
    </View>
    <Modal visible={visible} transparent animationType="none" onRequestClose={close}>
      {visible ? <TimingOverlay>
        <View accessibilityViewIsModal style={[styles.sheet, { paddingBottom: Math.max(insets.bottom, spacing.lg) }]}>
          <View style={styles.header}>
            <View style={styles.headingBlock}>
              <Text accessibilityRole="header" style={styles.title}>Next review</Text>
              <Text style={styles.word}>{word}</Text>
              <Text style={styles.detail}>Remembered {count} {count === 1 ? 'time' : 'times'}</Text>
            </View>
            <OutlineButton label="Cancel" variant="text" onPress={close} />
          </View>
          <ScrollView style={styles.options} contentContainerStyle={styles.optionContent}>
            {visibleOptions.map((option) => <Pressable key={option.value} accessibilityRole="radio"
              accessibilityLabel={`${option.label}${option.value === timing?.recommendedInterval ? ', recommended' : ''}`}
              aria-checked={draft === option.value} onPress={() => setDraft(option.value)}
              style={({ pressed }) => [styles.option, pressed && styles.pressed]}>
              <View style={styles.radio}>{draft === option.value ? <View style={styles.radioFill} /> : null}</View>
              <View style={styles.optionLabel}>
                <Text style={styles.optionText}>{option.label}</Text>
                {option.value === timing?.recommendedInterval ? <Text style={styles.detail}>Recommended</Text> : null}
              </View>
            </Pressable>)}
          </ScrollView>
          <Text style={styles.detail}>{draft === 'paused' ? 'This word stays on your map. Choose a review time there to resume.' : 'This records a successful recall and schedules your next review.'}</Text>
          <OutlineButton label={draft === 'paused' ? 'Remember & pause' : 'Remember & continue'} variant="primary" disabled={disabled}
            onPress={() => { close(); onConfirm(draft); }} />
        </View>
      </TimingOverlay> : null}
    </Modal>
  </>;
}
function TimingOverlay({ children }: React.PropsWithChildren) {
  const opacity = useReveal(motion.sheet);
  return <Animated.View style={[styles.overlay, { opacity }]}>{children}</Animated.View>;
}
const styles = StyleSheet.create({
  summary: { gap: spacing.xs }, count: { ...type.caption, color: colors.onDarkMuted },
  timingButton: { minHeight: 44, flexDirection: 'row', alignItems: 'center', justifyContent: 'space-between', gap: spacing.md },
  timingLabel: { flex: 1, ...type.label, color: colors.onDark, flexShrink: 1 }, change: { ...type.caption, color: colors.onDarkMuted, textDecorationLine: 'underline' },
  overlay: { flex: 1, backgroundColor: '#00000099', justifyContent: 'flex-end', alignItems: 'center' },
  sheet: { backgroundColor: colors.white, width: '100%', maxWidth: 600, maxHeight: '92%', borderTopLeftRadius: radii.xl, borderTopRightRadius: radii.xl, padding: spacing.xl, gap: spacing.lg },
  header: { flexDirection: 'row', alignItems: 'flex-start', gap: spacing.sm }, headingBlock: { flex: 1, gap: spacing.xs },
  title: type.heading, word: { ...type.body, color: colors.ink }, detail: type.caption,
  options: { flexShrink: 1 }, optionContent: { gap: spacing.xs },
  option: { flexDirection: 'row', gap: spacing.lg, minHeight: 48, alignItems: 'center', paddingVertical: spacing.sm },
  radio: { width: 22, height: 22, borderRadius: 11, borderWidth: 1, borderColor: colors.ink, alignItems: 'center', justifyContent: 'center' },
  radioFill: { width: 12, height: 12, borderRadius: 6, backgroundColor: colors.ink },
  optionLabel: { flex: 1 }, optionText: { ...type.body, color: colors.ink }, pressed: { opacity: 0.6 },
});
