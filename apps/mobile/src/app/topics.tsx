import { SafeAreaView } from 'react-native-safe-area-context';
import { router, useLocalSearchParams } from 'expo-router';
import { useEffect, useState } from 'react';
import { ActivityIndicator, ScrollView, StyleSheet, Text, View } from 'react-native';

import { iconForTopic } from '@/components/LineIcon';
import { CategoryListItem } from '@/components/CategoryListItem';
import { MinimalHeader } from '@/components/MinimalHeader';
import { OutlineButton } from '@/components/OutlineButton';
import { getJSON } from '@/lib/api';
import { LearningFilters, LearningWord, Topic } from '@/lib/learning';
import { colors, layout, spacing, type } from '@/lib/theme';
const maxSelectedTopics = 3;

export default function TopicsScreen() {
  const params = useLocalSearchParams<{ topic?: string }>();
  const [filters, setFilters] = useState<LearningFilters | null>(null);
  const [selectedSlugs, setSelectedSlugs] = useState<string[]>([]);
  const [preview, setPreview] = useState<{ selection: string; words: LearningWord[]; count: number } | null>(null);
  const [previewError, setPreviewError] = useState('');
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [loadAttempt, setLoadAttempt] = useState(0);

  useEffect(() => {
    let active = true;
    async function loadFilters() {
      try {
        const result = await getJSON<LearningFilters>('/topics');
        if (!active) return;
        setFilters(result);
        setSelectedSlugs(params.topic && result.topics.some((topic) => topic.slug === params.topic && topic.isAvailable) ? [params.topic] : []);
      } catch (loadError) {
        if (!active) return;
        setError(loadError instanceof Error ? loadError.message : 'Could not load practice categories.');
      } finally {
        if (active) setIsLoading(false);
      }
    }

    void loadFilters();
    return () => { active = false; };
  }, [loadAttempt, params.topic]);

  useEffect(() => {
    let active = true;
    if (selectedSlugs.length === 0) return;

    async function loadPreview() {
      try {
        const result = await getJSON<{ words: LearningWord[]; wordCount: number }>(`/topics/preview?topics=${encodeURIComponent(selectedSlugs.join(','))}`);
        if (!active) return;
        setPreview({ selection: selectedSlugs.join(','), words: result.words, count: result.wordCount });
      } catch {
        if (active) setPreviewError('Preview unavailable. You can still start practice.');
      }
    }

    void loadPreview();
    return () => { active = false; };
  }, [selectedSlugs]);

  function toggleTopic(topic: Topic) {
    if (!topic.isAvailable) {
      return;
    }
    setPreview(null);
    setPreviewError('');
    setSelectedSlugs((current) => {
      if (current.includes(topic.slug)) {
        return current.filter((slug) => slug !== topic.slug);
      }
      if (topic.slug === 'general-vocabulary') return [topic.slug];
      current = current.filter((slug) => slug !== 'general-vocabulary');
      if (current.length >= maxSelectedTopics) {
        return current;
      }
      return [...current, topic.slug];
    });
  }

  function startPractice() {
    if (selectedSlugs.length === 0) {
      return;
    }
    router.push({ pathname: '/review', params: { topics: selectedSlugs.join(',') } });
  }

  const selectedTopics = filters?.topics.filter((topic) => selectedSlugs.includes(topic.slug)) ?? [];
  const currentPreview = preview?.selection === selectedSlugs.join(',') ? preview : null;
  const previewWords = currentPreview?.words ?? [];
  const allWordsSelected = selectedSlugs.includes('general-vocabulary');
  const general = filters?.topics.find((topic) => topic.slug === 'general-vocabulary');

  return (
    <SafeAreaView style={styles.safeArea}>
      <ScrollView contentContainerStyle={styles.container} showsVerticalScrollIndicator={false}>
        <MinimalHeader backHref="/home" eyebrow="Categories" title={allWordsSelected ? 'All words' : `${selectedSlugs.length}/${maxSelectedTopics} selected`} />

        <View style={styles.header}>
          <Text style={styles.title}>Choose your categories.</Text>
          <Text style={styles.subtitle}>Practice all words or choose up to three categories.</Text>
        </View>

        {isLoading ? (
          <View style={styles.stateCard}>
            <ActivityIndicator color={colors.deepBlue} />
            <Text style={styles.stateText}>Loading categories</Text>
          </View>
        ) : error ? (
          <View style={styles.stateCard}>
            <Text style={styles.stateTitle}>Categories are unavailable</Text>
            <Text style={styles.stateText}>We couldn’t load the categories. Please try again.</Text>
            <OutlineButton label="Try again" onPress={() => { setIsLoading(true); setError(''); setLoadAttempt((value) => value + 1); }} />
          </View>
        ) : (
          <>
            {filters?.cefrLevels.some((level) => level.isAvailable) ? (
              <View style={styles.levelSection}>
                <Text style={styles.sectionTitle}>Available levels</Text>
                <View style={styles.levelRow}>
                  {filters.cefrLevels.filter((level) => level.isAvailable).map((level) => (
                    <View key={level.level} style={styles.levelChip}>
                      <Text style={styles.levelText}>{level.level}</Text>
                    </View>
                  ))}
                </View>
              </View>
            ) : null}

            {general ? <View style={styles.collection}>
              <Text style={type.label}>All words</Text>
              <CategoryListItem icon="book" title={general.title} description="Practice the complete vocabulary, across every category."
                meta={`${general.wordCount.toLocaleString()} practice words`} selected={allWordsSelected}
                disabled={!general.isAvailable} onPress={() => toggleTopic(general)} />
            </View> : null}
            <View style={styles.topicList}>
              <Text style={type.label}>Choose a focus</Text>
              {filters?.topics.filter((topic) => topic.slug !== 'general-vocabulary').map((topic) => {
                const isSelected = selectedSlugs.includes(topic.slug);
                const selectionIsFull = selectedSlugs.length >= maxSelectedTopics && !isSelected;
                const isDisabled = !topic.isAvailable || selectionIsFull;
                return (
                  <CategoryListItem
                    accessibilityRole="checkbox"
                    accessibilityState={{ checked: isSelected, disabled: isDisabled }}
                    icon={iconForTopic(topic.slug)}
                    description={topic.description}
                    disabled={isDisabled}
                    key={topic.slug}
                    meta={topic.startedCount > 0 ? `${topic.startedCount} of ${topic.wordCount} started` : topic.isAvailable ? `${topic.wordCount} practice words` : 'Coming soon'}
                    onPress={() => toggleTopic(topic)}
                    selected={isSelected}
                    title={topic.title}
                  />
                );
              })}
            </View>

            {selectedTopics.length > 0 ? (
              <View style={styles.selectionPanel}>
                <Text style={styles.selectionLabel}>{allWordsSelected ? 'Your practice' : 'Your mix'}</Text>
                <Text style={styles.selectionTitle}>{selectedTopics.map((topic) => topic.title).join(' + ')}</Text>
                <Text style={styles.selectionMeta}>{currentPreview ? `${currentPreview.count.toLocaleString()} unique words available` : previewError || 'Loading word preview…'}</Text>

                {previewWords.length > 0 ? (
                  <View style={styles.previewList}>
                    {previewWords.map((word) => (
                      <View key={word.id} style={styles.wordRow}>
                        <View>
                          <Text style={styles.word}>{word.word}</Text>
                          <Text style={styles.translation}>{word.translation}</Text>
                        </View>
                        <Text style={styles.wordTopic}>{word.topic}</Text>
                      </View>
                    ))}
                  </View>
                ) : null}


              </View>
            ) : (
              <View style={styles.emptySelection}>
                <Text style={styles.emptySelectionText}>Select a category to see your practice mix.</Text>
              </View>
            )}
          </>
        )}
      </ScrollView>
      {!isLoading && !error ? <View style={styles.dock}>
        <Text accessibilityLiveRegion="polite" style={styles.selectionMeta}>{allWordsSelected ? 'General Vocabulary · all words' : selectedSlugs.length === 3 ? 'Three selected. Deselect one to change your mix.' : `${selectedSlugs.length} of 3 categories selected`}</Text>
        <OutlineButton icon="learning" disabled={!selectedSlugs.length} label="Start practice" onPress={startPractice} variant="primary" />
      </View> : null}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  collection: { gap: spacing.sm },
  safeArea: layout.screen, container: layout.page, header: { gap: spacing.md }, title: type.title, subtitle: type.body,
  sectionTitle: type.label, levelSection: { gap: spacing.sm }, levelRow: { flexDirection: 'row', flexWrap: 'wrap', gap: spacing.lg },
  levelChip: { paddingVertical: spacing.xs }, levelText: type.caption, topicList: { marginTop: -spacing.lg },
  selectionPanel: { gap: spacing.md }, selectionLabel: type.caption, selectionTitle: type.heading, selectionMeta: type.caption,
  previewList: { marginTop: spacing.sm }, wordRow: { borderBottomColor: colors.line, borderBottomWidth: 1, flexDirection: 'row', justifyContent: 'space-between', gap: spacing.md, paddingVertical: spacing.md },
  word: type.body, translation: type.caption, wordTopic: { ...type.caption, flexShrink: 1, textAlign: 'right' },
  emptySelection: { paddingVertical: spacing.lg }, emptySelectionText: type.caption,
  stateCard: { gap: spacing.lg, paddingVertical: spacing.xl }, stateTitle: type.heading, stateText: type.body,
  dock: { ...layout.page, paddingTop: spacing.md, paddingBottom: spacing.lg, gap: spacing.sm, borderTopColor: colors.line, borderTopWidth: 1 },
});
