import { SafeAreaView } from 'react-native-safe-area-context';
import { router } from 'expo-router';
import { useState } from 'react';
import { KeyboardAvoidingView, Platform, Pressable, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';

import { OutlineButton } from '@/components/OutlineButton';
import { colors, layout, radii, spacing, type } from '@/lib/theme';
import { postJSON } from '@/lib/api';
import { useAuth } from '@/lib/auth';

const ageRanges = ['Under 18', '18-24', '25-34', '35-44', '45-54', '55+'];

export default function OnboardingScreen() {
  const { refreshUser } = useAuth();
  const [displayName, setDisplayName] = useState('');
  const [ageRange, setAgeRange] = useState('25-34');
  const [message, setMessage] = useState('Tell us how to address you.');
  const [isSaving, setIsSaving] = useState(false);

  async function saveProfile() {
    if (isSaving) return;
    setIsSaving(true);
    setMessage('Saving profile...');
    try {
      await postJSON('/profile', { displayName: displayName.trim(), ageRange });
      await refreshUser();
      router.replace('/home');
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'Could not save profile.');
    } finally {
      setIsSaving(false);
    }
  }

  return (
    <SafeAreaView style={styles.safeArea}>
      <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : undefined} style={{ flex: 1 }}>
      <ScrollView contentContainerStyle={styles.container} keyboardShouldPersistTaps="handled">
        <View style={styles.header}>
          <Text style={styles.title}>Make yourself at home.</Text>
          <Text style={styles.subtitle}>Add your name and age range to your learning profile.</Text>
        </View>
        <View style={styles.form}>
          <Text style={styles.label}>Display name</Text>
          <TextInput
            accessibilityLabel="Display name"
            autoComplete="given-name"
            onChangeText={setDisplayName}
            placeholder="Alex"
            placeholderTextColor={colors.muted}
            style={styles.input}
            value={displayName}
          />
          <Text style={styles.label}>Age range</Text>
          <View style={styles.ageGrid}>
            {ageRanges.map((range) => (
              <Pressable aria-checked={ageRange === range} accessibilityRole="radio" accessibilityState={{ checked: ageRange === range }} key={range} onPress={() => setAgeRange(range)} style={[styles.ageButton, ageRange === range && styles.ageButtonSelected]}>
                <Text style={[styles.ageText, ageRange === range && styles.ageTextSelected]}>{range}</Text>
              </Pressable>
            ))}
          </View>
          <Text accessibilityLiveRegion="polite" style={styles.message}>{message}</Text>
          <OutlineButton disabled={isSaving || !displayName.trim()} onPress={saveProfile} label={isSaving ? 'Saving…' : 'Continue'} variant="primary" />
        </View>
      </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: layout.screen, container: { ...layout.page, flexGrow: 1, justifyContent: 'center', gap: spacing.section },
  header: { gap: spacing.lg }, title: type.title, subtitle: type.body, form: { gap: spacing.lg }, label: type.label,
  input: { borderColor: colors.line, borderRadius: radii.md, borderWidth: 1, color: colors.ink, fontSize: 17, minHeight: 56, padding: spacing.lg },
  ageGrid: { flexDirection: 'row', flexWrap: 'wrap', gap: spacing.sm },
  ageButton: { borderColor: colors.line, borderRadius: radii.pill, borderWidth: 1, minHeight: 48, paddingHorizontal: spacing.lg, paddingVertical: spacing.md },
  ageButtonSelected: { backgroundColor: colors.ink, borderColor: colors.ink }, ageText: type.label, ageTextSelected: { color: colors.white }, message: type.caption,
});
