import { Redirect } from 'expo-router';
import { useAuth } from '@/lib/auth';
import { ActivityIndicator, StyleSheet, View } from 'react-native';

import { colors } from '@/lib/theme';

export default function IndexScreen() {
  const { isLoading, user } = useAuth();
  if (!isLoading) return <Redirect href={!user ? '/login' : user.profileComplete ? '/home' : '/onboarding'} />;
  return (
    <View style={styles.container}>
      <ActivityIndicator color={colors.deepBlue} size="large" />
    </View>
  );
}

const styles = StyleSheet.create({
  container: {
    alignItems: 'center',
    backgroundColor: colors.fantasy,
    flex: 1,
    justifyContent: 'center',
  },
});
