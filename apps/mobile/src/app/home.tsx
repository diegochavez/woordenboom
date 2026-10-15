import { SafeAreaView } from 'react-native-safe-area-context';
import { Link, useFocusEffect } from 'expo-router';
import { useCallback, useRef, useState } from 'react';
import { ActivityIndicator, AppState, Pressable, RefreshControl, ScrollView, StyleSheet, Text, View } from 'react-native';

import { iconForTopic } from '@/components/LineIcon';
import { CategoryListItem } from '@/components/CategoryListItem';
import { KnowledgeMapPreview } from '@/components/KnowledgeMapPreview';
import { ProgressBar } from '@/components/ProgressBar';
import { OutlineButton } from '@/components/OutlineButton';
import { getJSON } from '@/lib/api';
import { useAuth } from '@/lib/auth';
import { LearningFilters, Topic } from '@/lib/learning';
import { subscribeToProgressChanges } from '@/lib/progressEvents';
import { colors, layout, spacing, type } from '@/lib/theme';

type Progress = {
  totalWords: number;
  reviewedWords: number;
  masteredWords: number;
  dueWords: number;
  topicsTotal: number;
  topicsStarted: number;
  dailyGoal: number;
  dailyDone: number;
  strengthScore: number;
};

type KnowledgeMap = {
  availableWords: number;
  knownWords: number;
  learningWords: number;
  ranges: { available: number; endRank: number; known: number; learning: number; startRank: number; total: number }[];
  totalWords: number;
  unseenWords: number;
};

export default function HomeScreen() {
  const { logout, user } = useAuth();
  const [progress, setProgress] = useState<Progress | null>(null);
  const [knowledgeMap, setKnowledgeMap] = useState<KnowledgeMap | null>(null);
  const [topics, setTopics] = useState<Topic[]>([]);
  const [error, setError] = useState('');

  const [refreshing, setRefreshing] = useState(false);
  const refreshDashboard = useRef<((showSpinner?: boolean) => void) | null>(null);
  useFocusEffect(useCallback(() => {
    let active = true;
    let latestRequest = 0;
    async function loadDashboard(showSpinner = false) {
      const request = ++latestRequest;
      const isCurrent = () => active && request === latestRequest;
      // Native RefreshControl should only appear after a pull gesture.
      setRefreshing(showSpinner);
      setError('');
      // The daily total can update immediately, even if map/category reads are slow.
      const results = await Promise.allSettled([
        getJSON<Progress>('/progress/summary').then((result) => { if (isCurrent()) setProgress(result); }),
        getJSON<LearningFilters>('/topics').then((result) => { if (isCurrent()) setTopics(result.topics.filter((topic) => topic.isAvailable)); }),
        getJSON<KnowledgeMap>('/knowledge-map').then((result) => { if (isCurrent()) setKnowledgeMap(result); }),
      ]);
      if (!isCurrent()) return;
      if (results.some((result) => result.status === 'rejected')) setError('Could not refresh all your progress. Try again.');
      setRefreshing(false);
    }
    refreshDashboard.current = (showSpinner = false) => { void loadDashboard(showSpinner); };
    const refresh = () => refreshDashboard.current?.();
    const unsubscribe = subscribeToProgressChanges(refresh);
    let previousState = AppState.currentState;
    const subscription = AppState.addEventListener('change', (state) => {
      if (state === 'active' && previousState !== 'active') refresh();
      previousState = state;
    });
    refresh();
    return () => { active = false; refreshDashboard.current = null; unsubscribe(); subscription.remove(); };
  // A different account must start a fresh dashboard request.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.id]));

  return (
    <SafeAreaView style={styles.safeArea}>
      <ScrollView contentContainerStyle={styles.container} showsVerticalScrollIndicator={false}
        refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => refreshDashboard.current?.(true)} tintColor={colors.ink} colors={[colors.ink]} />}>
        <View style={styles.topBar}>
          <View style={styles.greetingBlock}>
            <Text style={styles.kicker}>Woordenboom</Text>
            <Text style={styles.greeting}>{user?.displayName ? `Hello, ${user.displayName}.` : 'Hello there.'}</Text>
          </View>
          <OutlineButton label="Log out" variant="text" onPress={logout} style={styles.logoutButton} />
        </View>

        {!progress && !error ? (
          <View style={styles.stateCard}>
            <ActivityIndicator color={colors.deepBlue} />
            <Text style={styles.stateText}>Loading your learning plan</Text>
          </View>
        ) : error && !progress ? (
          <View style={styles.stateCard}>
            <Text style={styles.stateTitle}>Dashboard unavailable</Text>
            <Text style={styles.stateText}>We couldn’t load your progress. Please try again.</Text>
            <OutlineButton label="Try again" onPress={() => refreshDashboard.current?.()} />
          </View>
        ) : progress ? (
          <>
            {error ? <View style={styles.refreshError}>
              <Text accessibilityLiveRegion="polite" style={type.caption}>{error}</Text>
              <OutlineButton label="Retry refresh" variant="text" onPress={() => refreshDashboard.current?.()} />
            </View> : null}
            <View style={styles.heroCard}>
              <Text style={styles.heroBadgeText}>Your daily practice</Text>
              <Text testID="daily-practice-count" accessibilityLiveRegion="polite" style={styles.heroTitle}>{progress.dailyDone}<Text style={styles.heroGoal}> / {progress.dailyGoal}</Text></Text>
              <Text style={styles.heroText}>words practiced today</Text>
              <ProgressBar value={progress.dailyDone} max={progress.dailyGoal} color={colors.ink} height={4} />
              <View style={styles.practiceActions}>
                <Text style={styles.practiceCategory}>General vocabulary</Text>
                <Link href={{ pathname: '/review', params: { topics: 'general-vocabulary' } }} asChild>
                  <OutlineButton icon="learning" variant="primary" label="Start practice" accessibilityHint="Practice General vocabulary" />
                </Link>
                <Link href="/topics" asChild>
                  <OutlineButton icon="categories" variant="text" label="Choose categories" />
                </Link>
              </View>
            </View>

            <View style={styles.statsRow}>
              <View style={styles.statCard}>
                <Text style={styles.statValue}>{progress.dueWords}</Text>
                <Text style={styles.statLabel}>due now</Text>
              </View>
              <View style={styles.statCard}>
                <Text style={styles.statValue}>{progress.masteredWords}</Text>
                <Text style={styles.statLabel}>mastered</Text>
              </View>
              <View style={styles.statCard}>
                <Text style={styles.statValue}>{progress.topicsStarted}</Text>
                <Text style={styles.statLabel}>active categories</Text>
              </View>
            </View>

            {knowledgeMap ? (
              <Link href="/knowledge-map" asChild>
                <Pressable accessibilityRole="link" accessibilityLabel="Open knowledge map">
                  <KnowledgeMapPreview ranges={knowledgeMap.ranges} />
                </Pressable>
              </Link>
            ) : null}

            <View style={styles.progressPanel}>
              <View style={styles.panelHeader}>
                <Text style={styles.sectionTitle}>Learning strength</Text>
                <Text style={styles.panelNumber}>{progress.strengthScore}%</Text>
              </View>
              <Text style={styles.panelText}>{progress.reviewedWords} of {progress.totalWords} available words practiced.</Text>
            </View>

            <View style={styles.sectionHeader}>
              <Text style={styles.sectionTitle}>Categories</Text>
              <Link href="/topics" style={styles.sectionLink}>See all</Link>
            </View>

            <View style={styles.topicList}>
              {topics.slice(0, 6).map((topic) => (
                <Link href={{ pathname: "/topics", params: { topic: topic.slug } }} key={topic.slug} asChild>
                  <CategoryListItem icon={iconForTopic(topic.slug)} meta={`${topic.wordCount} practice words`} title={topic.title} />
                </Link>
              ))}
            </View>
          </>
        ) : null}

        <Link href="/disclaimer" style={styles.legalLink}>Data sources and licenses</Link>
      </ScrollView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  refreshError: { gap: spacing.sm, alignItems: 'flex-start' },
  safeArea: layout.screen, container: layout.page,
  topBar: { alignItems: 'flex-start', flexDirection: 'row', gap: spacing.md, justifyContent: 'space-between' },
  greetingBlock: { flex: 1, minWidth: 0 }, kicker: type.caption,
  greeting: { ...type.heading, marginTop: spacing.sm }, logoutButton: { minHeight: 44 },
  heroCard: { gap: spacing.lg, paddingVertical: spacing.lg }, heroBadgeText: type.body,
  heroTitle: { ...type.display, fontSize: 88, lineHeight: 96, letterSpacing: -5, fontVariant: ['tabular-nums'] },
  heroGoal: { fontSize: 32, letterSpacing: -1, color: colors.muted },
  heroText: { ...type.body, marginTop: -spacing.sm },
  practiceActions: { gap: spacing.sm, marginTop: spacing.sm }, practiceCategory: type.caption,
  statsRow: { flexDirection: 'row', gap: spacing.lg, paddingVertical: spacing.sm },
  statCard: { flex: 1, gap: spacing.xs }, statValue: { ...type.title, fontSize: 32 }, statLabel: type.caption,
  progressPanel: { gap: spacing.md, borderTopWidth: 1, borderTopColor: colors.line, paddingTop: spacing.xl },
  panelHeader: { alignItems: 'baseline', flexDirection: 'row', flexWrap: 'wrap', gap: spacing.sm, justifyContent: 'space-between' },
  panelNumber: type.heading, panelText: type.caption,
  sectionHeader: { alignItems: 'center', flexDirection: 'row', justifyContent: 'space-between', gap: spacing.md },
  sectionTitle: type.heading, sectionLink: { ...type.label, paddingVertical: spacing.md },
  topicList: { marginTop: -spacing.xl },
  stateCard: { gap: spacing.lg, paddingVertical: spacing.section }, stateTitle: type.heading, stateText: type.body,
  legalLink: { ...type.caption, paddingVertical: spacing.md },
});
