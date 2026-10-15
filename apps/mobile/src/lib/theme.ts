// Shared native tokens. See /design.md for screen and interaction rules.
import { StyleSheet } from 'react-native';

export const colors = {
  fantasy: '#FFFFFF', surface: '#FFFFFF', surfaceMuted: '#F4F4F4',
  ink: '#111111', inkSoft: '#383838', muted: '#686868', line: '#E5E5E5',
  white: '#FFFFFF', black: '#111111', onDark: '#FFFFFF', onDarkMuted: '#BDBDBD',
  gold: '#111111', orange: '#111111', blue: '#EEEEEE', deepBlue: '#111111',
  paleBlue: '#F4F4F4', selectedBlue: '#F4F4F4', teal: '#686868',
  green: '#FFFFFF', coral: '#383838', lavender: '#EEEEEE', pink: '#EEEEEE',
  known: '#C9E89B', learning: '#FFE047', unseen: '#FFFFFF', locked: '#F4F4F4',
  track: '#E5E5E5', border: '#E5E5E5', transparent: 'transparent',
};
export const semantic = {
  appBackground: colors.white, cardBorder: colors.line, controlBorder: colors.ink,
  controlFill: colors.transparent, controlPressed: '#0000000D', quietText: colors.muted,
};
export const radii = { sm: 8, md: 12, lg: 16, xl: 24, pill: 999 };
export const spacing = { xs: 4, sm: 8, md: 12, lg: 16, xl: 24, xxl: 32, section: 40 };
export const typography = {
  display: 'System', body: 'System', mono: 'Courier',
  weight: { regular: '400' as const, medium: '500' as const, bold: '600' as const, black: '700' as const },
};
export const type = StyleSheet.create({
  display: { color: colors.ink, fontSize: 48, lineHeight: 54, fontWeight: '600', letterSpacing: -2 },
  title: { color: colors.ink, fontSize: 36, lineHeight: 41, fontWeight: '600', letterSpacing: -1.3 },
  heading: { color: colors.ink, fontSize: 22, lineHeight: 28, fontWeight: '600', letterSpacing: -0.5 },
  body: { color: colors.inkSoft, fontSize: 16, lineHeight: 24 },
  caption: { color: colors.muted, fontSize: 14, lineHeight: 20 },
  label: { color: colors.ink, fontSize: 14, lineHeight: 20, fontWeight: '500' },
});
export const layout = StyleSheet.create({
  screen: { backgroundColor: colors.white, flex: 1 },
  page: { width: '100%', maxWidth: 600, alignSelf: 'center', padding: spacing.xl, paddingBottom: spacing.section, gap: spacing.xxl },
});
export const cardPalette = [colors.black];
export function colorForIndex(_index: number) { return colors.black; }
