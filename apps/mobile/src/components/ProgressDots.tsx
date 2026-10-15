import { StyleSheet, View } from 'react-native';

import { colors } from '@/lib/theme';

type ProgressDotsProps = {
  current: number;
  total: number;
};

export function ProgressDots({ current, total }: ProgressDotsProps) {
  const visible = Array.from({ length: Math.min(total, 14) });
  return (
    <View style={styles.row}>
      {visible.map((_, index) => {
        const active = index < current;
        const currentDot = index === current;
        return <View key={index} style={[styles.dot, active && styles.active, currentDot && styles.current]} />;
      })}
    </View>
  );
}

const styles = StyleSheet.create({
  row: { alignItems: 'center', flexDirection: 'row', gap: 6, justifyContent: 'center' },
  dot: { borderColor: colors.ink, borderRadius: 999, borderWidth: 1, height: 8, width: 8 },
  active: { backgroundColor: colors.ink },
  current: { width: 20 },
});
