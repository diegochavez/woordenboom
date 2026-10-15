import { Animated, StyleSheet, View } from 'react-native';

import { motion, useMotionValue } from '@/hooks/useMotion';
import { colors, radii } from '@/lib/theme';

type ProgressBarProps = {
  value: number;
  max: number;
  color?: string;
  height?: number;
};

export function ProgressBar({ value, max, color = colors.ink, height = 4 }: ProgressBarProps) {
  const percent = Math.max(0, Math.min(100, max === 0 ? 0 : (value / max) * 100));

  const fill = useMotionValue(percent, motion.progress, false);
  return (
    <View aria-valuemin={0} aria-valuemax={100} aria-valuenow={Math.round(percent)} accessibilityRole="progressbar" accessibilityValue={{ min: 0, max: 100, now: Math.round(percent) }} style={[styles.track, { height }]}>
      <Animated.View style={[styles.fill, { width: fill.interpolate({ inputRange: [0, 100], outputRange: ['0%', '100%'] }), backgroundColor: color }]} />
    </View>
  );
}

const styles = StyleSheet.create({
  track: {
    backgroundColor: colors.track,
    borderRadius: radii.pill,
    overflow: 'hidden',
    width: '100%',
  },
  fill: {
    borderRadius: radii.pill,
    height: '100%',
  },
});
