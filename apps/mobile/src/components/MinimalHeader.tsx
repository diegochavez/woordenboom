import { Href, Link } from 'expo-router';
import { ReactNode } from 'react';
import { StyleSheet, Text, View } from 'react-native';

import { CircleButton } from '@/components/OutlineButton';
import { colors } from '@/lib/theme';

type MinimalHeaderProps = {
  backHref?: Href;
  eyebrow?: string;
  right?: ReactNode;
  title?: string;
};

export function MinimalHeader({ backHref, eyebrow, right, title }: MinimalHeaderProps) {
  return (
    <View style={styles.header}>
      {backHref ? (
        <Link href={backHref} dismissTo asChild>
          <CircleButton icon="arrow-left" />
        </Link>
      ) : <View style={styles.placeholder} />}
      <View style={styles.copy}>
        {eyebrow ? <Text style={styles.eyebrow}>{eyebrow}</Text> : null}
        {title ? <Text numberOfLines={1} style={styles.title}>{title}</Text> : null}
      </View>
      {right ?? <View style={styles.placeholder} />}
    </View>
  );
}

const styles = StyleSheet.create({
  header: { alignItems: 'center', flexDirection: 'row', gap: 12, justifyContent: 'space-between', minHeight: 48 },
  placeholder: { height: 44, width: 44 },
  copy: { alignItems: 'center', flex: 1, minWidth: 0 },
  eyebrow: { color: colors.muted, fontSize: 12, fontWeight: '600', letterSpacing: 0 },
  title: { color: colors.ink, fontSize: 15, fontWeight: '600', letterSpacing: 0.2, marginTop: 1, maxWidth: '100%' },
});
