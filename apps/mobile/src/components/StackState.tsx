import { Animated, ActivityIndicator, ScrollView, StyleSheet, Text, View } from 'react-native';
import { SafeAreaView } from 'react-native-safe-area-context';
import { OutlineButton } from '@/components/OutlineButton';
import { LineIcon } from '@/components/LineIcon';
import { motion, useReveal } from '@/hooks/useMotion';
import { colors, layout, spacing, type } from '@/lib/theme';

type Props = { title: string; message: string; loading?: boolean; complete?: boolean; total?: number; known?: number;
  actionLabel?: string; onAction?: () => void; secondaryLabel?: string; onSecondary?: () => void };
export function StackState({ title, message, loading, complete, total = 0, known = 0, actionLabel, onAction, secondaryLabel, onSecondary }: Props) {
  const opacity = useReveal(complete ? motion.complete : 0);
  return (
    <SafeAreaView style={layout.screen}>
      <ScrollView contentContainerStyle={styles.content}>
        <Animated.View key={complete ? 'complete' : 'state'} style={[styles.copy, { opacity: complete ? opacity : 1 }]}>
          {complete ? <LineIcon name="check" size={24} /> : null}
          {loading ? <ActivityIndicator color={colors.ink} /> : null}
          {complete ? <Text style={type.label}>{total} words practiced</Text> : null}
          <Text accessibilityRole="header" style={type.display}>{title}</Text>
          <Text accessibilityLiveRegion="polite" style={type.body}>{message}</Text>
          {complete ? <View style={styles.stats}>
            <View style={styles.stat}><Text style={type.title}>{known}</Text><Text style={type.caption}>known</Text></View>
            <View style={styles.stat}><Text style={type.title}>{Math.max(0, total - known)}</Text><Text style={type.caption}>still learning</Text></View>
          </View> : null}
        </Animated.View>
        <View style={styles.actions}>
          {actionLabel && onAction ? <OutlineButton label={actionLabel} onPress={onAction} variant="primary" /> : null}
          {secondaryLabel && onSecondary ? <OutlineButton label={secondaryLabel} onPress={onSecondary} /> : null}
        </View>
      </ScrollView>
    </SafeAreaView>
  );
}
const styles = StyleSheet.create({
  content: { ...layout.page, flexGrow: 1, paddingBottom: spacing.xl },
  copy: { flexGrow: 1, justifyContent: 'center', gap: spacing.xl, paddingVertical: spacing.section },
  stats: { flexDirection: 'row', gap: spacing.xxl, marginTop: spacing.xl },
  stat: { flex: 1, gap: spacing.xs }, actions: { gap: spacing.md },
});
