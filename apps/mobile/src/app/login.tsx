import { SafeAreaView } from 'react-native-safe-area-context';
import { Link } from 'expo-router';
import { useState } from 'react';
import { KeyboardAvoidingView, Platform, ScrollView, StyleSheet, Text, TextInput, View } from 'react-native';

import { OutlineButton } from '@/components/OutlineButton';
import { colors, layout, radii, spacing, type } from '@/lib/theme';
import { postPublicJSON } from '@/lib/api';
import { AuthUser, useAuth } from '@/lib/auth';

export default function LoginScreen() {
  const { completeLogin } = useAuth();
  const [email, setEmail] = useState('');
  const [code, setCode] = useState('');
  const [codeRequested, setCodeRequested] = useState(false);
  const [message, setMessage] = useState('Enter your email to receive a login code.');
  const [isSubmitting, setIsSubmitting] = useState(false);

  async function requestCode() {
    if (isSubmitting) return;
    setIsSubmitting(true);
    setMessage('Requesting code...');
    try {
      await postPublicJSON('/auth/request-code', { email: email.trim().toLowerCase() });
      setCodeRequested(true);
      setMessage('Code sent. Check your inbox.');
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'Could not request code.');
    } finally {
      setIsSubmitting(false);
    }
  }

  async function verifyCode() {
    if (isSubmitting) return;
    setIsSubmitting(true);
    setMessage('Verifying code...');
    try {
      const result = await postPublicJSON<{ accessToken: string; refreshToken: string; user: AuthUser }>('/auth/verify-code', { email: email.trim().toLowerCase(), code });
      await completeLogin(result.accessToken, result.refreshToken, result.user);
    } catch (error) {
      setMessage(error instanceof Error ? error.message : 'Could not verify code.');
    } finally {
      setIsSubmitting(false);
    }
  }

  return (
    <SafeAreaView style={styles.safeArea}>
      <KeyboardAvoidingView behavior={Platform.OS === 'ios' ? 'padding' : undefined} style={styles.keyboardView}>
        <ScrollView
          contentContainerStyle={styles.container}
          keyboardShouldPersistTaps="handled"
          showsVerticalScrollIndicator={false}
        >
          <View style={styles.copy}>
            <Text style={styles.title}>A little Dutch, every day.</Text>
            <Text style={styles.subtitle}>Sign in with a code sent to your email.</Text>
          </View>
          <View style={styles.form}>
            <Text style={styles.label}>Email</Text>
            <TextInput
              autoCapitalize="none"
              accessibilityLabel="Email"
              autoComplete="email"
              editable={!codeRequested}
              inputMode="email"
              keyboardType="email-address"
              onChangeText={setEmail}
              placeholder="you@example.com"
              placeholderTextColor={colors.muted}
              returnKeyType="send"
              style={styles.input}
              value={email}
            />
            {codeRequested ? (
              <>
                <Text style={styles.label}>6-digit code</Text>
                <TextInput
                  accessibilityLabel="6-digit code"
                  autoComplete="one-time-code"
                  textContentType="oneTimeCode"
                  inputMode="numeric"
                  keyboardType="number-pad"
                  maxLength={6}
                  onChangeText={(value) => setCode(value.replace(/\D/g, ''))}
                  placeholder="123456"
                  placeholderTextColor={colors.muted}
                  style={styles.input}
                  value={code}
                />
              </>
            ) : null}
            <Text accessibilityLiveRegion="polite" style={styles.message}>{message}</Text>
            <OutlineButton disabled={isSubmitting || (codeRequested ? code.length !== 6 : !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim()))} onPress={codeRequested ? verifyCode : requestCode} label={isSubmitting ? 'Please wait…' : codeRequested ? 'Verify code' : 'Send code'} variant="primary" />
            {codeRequested ? <>
              <OutlineButton disabled={isSubmitting} label="Resend code" onPress={requestCode} variant="text" />
              <OutlineButton disabled={isSubmitting} label="Use another email" onPress={() => { setCodeRequested(false); setCode(''); setMessage('Enter your email to receive a login code.'); }} variant="text" />
            </> : null}
          </View>
          <Link href="/disclaimer" style={styles.legalLink}>Data sources and licenses</Link>
        </ScrollView>
      </KeyboardAvoidingView>
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  safeArea: layout.screen, keyboardView: { flex: 1 },
  container: { ...layout.page, flexGrow: 1, justifyContent: 'center', gap: spacing.section },
  copy: { gap: spacing.lg }, title: type.title, subtitle: type.body,
  form: { gap: spacing.md }, label: type.label,
  input: { borderColor: colors.line, borderRadius: radii.md, borderWidth: 1, color: colors.ink, fontSize: 17, minHeight: 56, paddingHorizontal: spacing.lg, paddingVertical: spacing.lg },
  message: type.caption, legalLink: { ...type.caption, paddingVertical: spacing.md },
});
