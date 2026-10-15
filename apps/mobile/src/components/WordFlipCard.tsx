import { Pressable, StyleSheet, Text, View } from 'react-native';

import { LineIcon } from '@/components/LineIcon';
import { colors, radii, typography } from '@/lib/theme';

type WordFlipCardProps = {
  cefrLevel?: string;
  example?: string;
  flipped: boolean;
  onFlip: () => void;
  topic: string;
  translation: string;
  word: string;
};

export function WordFlipCard({ cefrLevel, example, flipped, onFlip, topic, translation, word }: WordFlipCardProps) {
  return (
    <Pressable accessibilityRole="button" onPress={onFlip} style={({ pressed }) => [styles.card, pressed && styles.pressed]}>
      <View style={styles.headerRow}>
        <Text style={styles.eyebrow}>{topic}</Text>
        {cefrLevel ? <Text style={styles.badge}>{cefrLevel}</Text> : null}
      </View>

      {!flipped ? (
        <View style={styles.frontContent}>
          <Text style={styles.word}>{word}</Text>
          <Text style={styles.hint}>Tap anywhere to reveal.</Text>
        </View>
      ) : (
        <View style={styles.backContent}>
          <Text style={styles.translation}>{translation}</Text>
          {example ? <Text style={styles.example}>{example}</Text> : null}
        </View>
      )}

      <View style={styles.flipPill}>
        <Text style={styles.flipText}>{flipped ? 'show word' : 'flip'}</Text>
        <LineIcon color={colors.onDark} name="flip" size={18} strokeWidth={1.8} />
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  card: { backgroundColor: colors.black, borderColor: colors.onDark, borderRadius: radii.xl, borderWidth: 0, flex: 1, gap: 20, justifyContent: 'space-between', minHeight: 0, padding: 26 },
  pressed: { opacity: 0.96 },
  headerRow: { alignItems: 'center', flexDirection: 'row', justifyContent: 'space-between' },
  eyebrow: { color: colors.onDark, fontSize: 12, fontWeight: '600', letterSpacing: 1.1, maxWidth: '70%', textTransform: 'uppercase' },
  badge: { borderColor: colors.onDark, borderRadius: radii.pill, borderWidth: 1, color: colors.onDark, fontSize: 13, fontWeight: '600', overflow: 'hidden', paddingHorizontal: 12, paddingVertical: 6 },
  frontContent: { gap: 18, justifyContent: 'center' },
  backContent: { gap: 24, justifyContent: 'center' },
  word: { color: colors.onDark, fontFamily: typography.body, fontSize: 58, fontWeight: '600', letterSpacing: -2.4, lineHeight: 60 },
  hint: { color: colors.onDarkMuted, fontSize: 19, fontWeight: '700', lineHeight: 27, maxWidth: 260 },
  translation: { color: colors.onDark, fontSize: 42, fontWeight: '600', letterSpacing: -1.4, lineHeight: 47 },
  example: { borderColor: colors.onDark, borderRadius: 0, borderWidth: 0, color: colors.onDarkMuted, fontSize: 18, fontWeight: '600', lineHeight: 27, padding: 0 },
  flipPill: { alignItems: 'center', borderColor: colors.onDark, borderRadius: radii.pill, borderWidth: 1.5, flexDirection: 'row', gap: 8, justifyContent: 'center', paddingVertical: 15 },
  flipText: { color: colors.onDark, fontSize: 16, fontWeight: '600' },
});
