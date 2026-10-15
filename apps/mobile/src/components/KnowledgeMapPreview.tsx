import { Animated, Pressable, StyleSheet, Text, useWindowDimensions, View } from 'react-native';
import { LineIcon } from '@/components/LineIcon';
import { motion, useMotionValue } from '@/hooks/useMotion';
import { colors, radii, spacing, type } from '@/lib/theme';
type RangePreview = { available: number; endRank: number; known: number; learning: number; startRank: number; total: number };
type Props = { onRangePress?: (startRank: number) => void; ranges: RangePreview[] };
export function KnowledgeMapPreview({ onRangePress, ranges }: Props) {
  const { width, fontScale } = useWindowDimensions();
  const columns = width < 360 || fontScale > 1.3 ? 4 : 5;
  const rows = Array.from({ length: Math.ceil(ranges.length / columns) }, (_, i) => ranges.slice(i * columns, (i + 1) * columns));
  return (
    <View style={styles.panel}>
      <View style={styles.header}>
        <View style={styles.heading}>
          <LineIcon name="map" size={20} />
          <Text accessibilityRole="header" style={styles.title}>Knowledge map</Text>
        </View>
        <Text style={styles.rangeCount}>{ranges.length} ranges</Text>
      </View>
      <View style={[styles.grid, onRangePress && styles.interactiveGrid]}>
        {rows.map((row, rowIndex) => <View key={rowIndex} style={styles.row}>
          {row.map((range) => {
            const unavailable = range.available === 0;
            const contents = <>
              <RangeFill range={range} value={range.known + range.learning} color={colors.learning} />
              <RangeFill range={range} value={range.known} color={colors.known} />
              <Text style={[styles.cellText, unavailable && styles.lockedText]}>{range.endRank}</Text>
            </>;
            const cellStyle = [styles.cell, unavailable && styles.locked];
            return onRangePress ? <Pressable key={range.startRank} accessibilityRole="button"
              accessibilityLabel={`Words ${range.startRank} to ${range.endRank}, ${range.known} known, ${range.learning} learning${unavailable ? ', not available yet' : ''}`}
              aria-disabled={unavailable} accessibilityState={{ disabled: unavailable }} disabled={unavailable} onPress={() => onRangePress(range.startRank)}
              style={({ pressed }) => [styles.rangeTarget, pressed && styles.pressed]}><View style={cellStyle}>{contents}</View></Pressable>
              : <View key={range.startRank} style={cellStyle}>{contents}</View>;
          })}
          {Array.from({ length: columns - row.length }, (_, i) => <View key={`space-${i}`} style={styles.spacer} />)}
        </View>)}
      </View>
      <View style={styles.legend}>
        {([['Known', colors.known], ['Learning', colors.learning], ['Not available', colors.locked]] as const).map(([label, color]) => <View key={label} style={styles.legendItem}>
          <View style={[styles.dot, { backgroundColor: color }]} /><Text style={type.caption}>{label}</Text>
        </View>)}
      </View>
    </View>
  );
}
function RangeFill({ range, value, color }: { range: RangePreview; value: number; color: string }) {
  const percent = Math.max(0, Math.min(100, range.total > 0 ? value / range.total * 100 : 0));
  const width = useMotionValue(percent, motion.progress, false);
  return <Animated.View style={[styles.knownFill, { backgroundColor: color, width: width.interpolate({ inputRange: [0, 100], outputRange: ['0%', '100%'] }) }]} />;
}
const styles = StyleSheet.create({
  // Center the icon/text group instead of deriving a native baseline from the SVG.
  // Let both labels wrap and give the title's descenders room on iOS.
  heading: { flexDirection: 'row', alignItems: 'center', gap: spacing.sm, flexShrink: 1, minWidth: 0, maxWidth: '100%' },
  title: { ...type.heading, lineHeight: 30, paddingVertical: 2, flexShrink: 1 },
  rangeCount: { ...type.caption, flexShrink: 0 },
  panel: { gap: spacing.lg }, header: { flexDirection: 'row', flexWrap: 'wrap', alignItems: 'center', justifyContent: 'space-between', gap: spacing.sm },
  grid: { gap: spacing.xs }, interactiveGrid: { gap: 0 },
  rangeTarget: { flex: 1, minHeight: 44, paddingVertical: 8 }, row: { flexDirection: 'row', gap: spacing.sm }, spacer: { flex: 1 },
  cell: { backgroundColor: colors.white, borderColor: colors.line, borderRadius: radii.sm, borderWidth: 1, minHeight: 28, paddingVertical: 3, flex: 1, justifyContent: 'center', overflow: 'hidden' },
  locked: { backgroundColor: colors.locked }, lockedText: { color: colors.muted }, pressed: { opacity: 0.6 },
  learningFill: { backgroundColor: colors.learning, bottom: 0, left: 0, position: 'absolute', top: 0 },
  knownFill: { backgroundColor: colors.known, bottom: 0, left: 0, position: 'absolute', top: 0 },
  cellText: { ...type.label, fontSize: 12, textAlign: 'center', fontVariant: ['tabular-nums'] },
  legend: { flexDirection: 'row', flexWrap: 'wrap', gap: spacing.md }, legendItem: { flexDirection: 'row', alignItems: 'center', gap: spacing.xs },
  dot: { width: 10, height: 10, borderRadius: 5, borderWidth: 1, borderColor: colors.line },
});
