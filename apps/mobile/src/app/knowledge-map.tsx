import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useFocusEffect } from 'expo-router';
import { useCallback, useState } from 'react';
import { ActivityIndicator, AppState, ScrollView, StyleSheet, Text, View } from 'react-native';

import { KnowledgeMapPreview } from '@/components/KnowledgeMapPreview';
import { MinimalHeader } from '@/components/MinimalHeader';
import { OutlineButton } from '@/components/OutlineButton';
import { getJSON } from '@/lib/api';
import { subscribeToProgressChanges } from '@/lib/progressEvents';
import { colors, layout, spacing, type } from '@/lib/theme';

type KnowledgeMap = {
  availableWords: number;
  knownWords: number;
  learningWords: number;
  ranges: { available: number; endRank: number; known: number; learning: number; startRank: number; total: number }[];
  totalWords: number;
  unseenWords: number;
};

export default function KnowledgeMapScreen() {
  const [knowledgeMap, setKnowledgeMap] = useState<KnowledgeMap | null>(null);
  const [error, setError] = useState('');

  const [loadAttempt, setLoadAttempt] = useState(0);
  useFocusEffect(useCallback(() => {
    let active = true;
    let latestRequest = 0;
    async function loadMap() {
      const request = ++latestRequest;
      try {
        setError('');
        const result = await getJSON<KnowledgeMap>('/knowledge-map');
        if (active && request === latestRequest) setKnowledgeMap(result);
      } catch (loadError) {
        if (!active || request !== latestRequest) return;
        setError(loadError instanceof Error ? loadError.message : 'Could not load your knowledge map.');
      }
    }

    const refresh = () => { void loadMap(); };
    const unsubscribe = subscribeToProgressChanges(refresh);
    let previousState = AppState.currentState;
    const subscription = AppState.addEventListener('change', (state) => {
      if (state === 'active' && previousState !== 'active') refresh();
      previousState = state;
    });
    refresh();
    return () => { active = false; unsubscribe(); subscription.remove(); };
  // The retry token restarts the focused subscription.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loadAttempt]));

  return (
    <SafeAreaView style={styles.safeArea}>
      <ScrollView contentContainerStyle={styles.container} showsVerticalScrollIndicator={false}>
        <MinimalHeader backHref="/home" title="Your progress" />

        <View style={styles.hero}>

          <Text style={styles.title}>Your Dutch, word by word.</Text>
          <Text style={styles.subtitle}>Each range contains 100 Dutch words, ordered by frequency. Tap a range to practice.</Text>
        </View>

        {!knowledgeMap && !error ? (
          <View style={styles.stateCard}>
            <ActivityIndicator color={colors.ink} />
            <Text style={styles.stateText}>Building your map</Text>
          </View>
        ) : error ? (
          <View style={styles.stateCard}>
            <Text style={styles.stateTitle}>Map unavailable</Text>
            <Text style={styles.stateText}>We couldn’t load your map. Please try again.</Text>
            <OutlineButton label="Try again" onPress={() => setLoadAttempt((value) => value + 1)} />
          </View>
        ) : knowledgeMap ? (
          <>
            <View style={styles.statsGrid}>
              <Metric label="available" value={knowledgeMap.availableWords} />
              <Metric label="known" value={knowledgeMap.knownWords} />
              <Metric label="learning" value={knowledgeMap.learningWords} />
              <Metric label="total" value={knowledgeMap.totalWords} />
            </View>
            <KnowledgeMapPreview onRangePress={(startRank) => router.push({ pathname: '/knowledge-range', params: { start: String(startRank) } })} ranges={knowledgeMap.ranges} />
          </>
        ) : null}
      </ScrollView>
    </SafeAreaView>
  );
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <View style={styles.metric}>
      <Text style={styles.metricValue}>{value.toLocaleString()}</Text>
      <Text style={styles.metricLabel}>{label}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  safeArea: layout.screen, container: layout.page, hero: { gap: spacing.lg },
  title: type.title, subtitle: type.body,
  statsGrid: { flexDirection: 'row', flexWrap: 'wrap', gap: spacing.xl },
  metric: { flexBasis: '40%', flexGrow: 1, gap: spacing.xs }, metricValue: type.title, metricLabel: type.caption,
  stateCard: { gap: spacing.lg, paddingVertical: spacing.section }, stateTitle: type.heading, stateText: type.body,
});
