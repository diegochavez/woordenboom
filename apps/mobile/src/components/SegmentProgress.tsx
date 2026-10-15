import { StyleSheet, View } from 'react-native';

import { colors, radii } from '@/lib/theme';

type SegmentProgressProps = {
  total: number;
  current: number;
};

export function SegmentProgress({ total, current }: SegmentProgressProps) {
  return (
    <View style={styles.row}>
      {Array.from({ length: total }).map((_, index) => (
        <View key={index} style={[styles.segment, index < current && styles.segmentActive]} />
      ))}
    </View>
  );
}

const styles = StyleSheet.create({
  row: {
    flexDirection: 'row',
    gap: 5,
  },
  segment: {
    backgroundColor: colors.track,
    borderRadius: radii.pill,
    flex: 1,
    height: 4,
  },
  segmentActive: {
    backgroundColor: colors.ink,
  },
});
