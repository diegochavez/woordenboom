import { Stack } from 'expo-router';
import { ActivityIndicator, View } from 'react-native';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { StatusBar } from 'expo-status-bar';

import { AuthProvider, useAuth } from '@/lib/auth';
import { colors } from '@/lib/theme';

function AuthenticatedStack() {
  const { isLoading, user } = useAuth();

  if (isLoading) return <View style={{ flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: colors.white }}><ActivityIndicator color={colors.ink} /></View>;

  return (
    <>
      <Stack screenOptions={{ headerShown: false, contentStyle: { backgroundColor: colors.fantasy } }}>
        <Stack.Screen name="index" />
        <Stack.Protected guard={!isLoading && !user}>
          <Stack.Screen name="login" />
        </Stack.Protected>
        <Stack.Protected guard={!isLoading && Boolean(user) && !user?.profileComplete}>
          <Stack.Screen name="onboarding" />
        </Stack.Protected>
        <Stack.Protected guard={!isLoading && Boolean(user?.profileComplete)}>
          <Stack.Screen name="home" />
          <Stack.Screen name="knowledge-map" />
          <Stack.Screen name="knowledge-range" />
          <Stack.Screen name="topics" />
          <Stack.Screen name="review" />
        </Stack.Protected>
        <Stack.Screen name="disclaimer" />
      </Stack>
      <StatusBar style="dark" />
    </>
  );
}

export default function RootLayout() {
  return (
    <SafeAreaProvider>
    <AuthProvider>
      <AuthenticatedStack />
    </AuthProvider>
    </SafeAreaProvider>
  );
}
