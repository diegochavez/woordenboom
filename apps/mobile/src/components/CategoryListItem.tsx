import { forwardRef } from 'react';
import { Animated, Pressable, PressableProps, StyleProp, StyleSheet, Text, View, ViewStyle } from 'react-native';
import { IconName, LineIcon } from '@/components/LineIcon';
import { motion, useMotionValue } from '@/hooks/useMotion';
import { colors, radii, spacing, type } from '@/lib/theme';

type Props = Omit<PressableProps, 'style'> & { icon?: IconName; accent?: string; description?: string; meta: string; selected?: boolean; style?: StyleProp<ViewStyle>; title: string };
export const CategoryListItem = forwardRef<View, Props>(function CategoryListItem({ accent: _accent, icon, description, disabled, meta, selected, title, style, ...props }, ref) {
  const opacity = useMotionValue(selected ? 1 : 0, motion.selection);
  const selectable = typeof selected === 'boolean';
  return (
    <Pressable accessibilityRole={selectable ? 'checkbox' : 'button'} accessibilityState={selectable ? { checked: selected, disabled: Boolean(disabled) } : { disabled: Boolean(disabled) }}
      aria-checked={selectable ? selected : undefined} aria-disabled={Boolean(disabled)} disabled={disabled} ref={ref} style={({ pressed }) => [styles.item, pressed && styles.pressed, disabled && styles.disabled, style]} {...props}>
      {icon ? <LineIcon name={icon} size={20} /> : null}
      <View style={styles.copy}>
        <Text style={styles.title}>{title}</Text>
        {description ? <Text style={styles.description}>{description}</Text> : null}
        <Text style={styles.meta}>{meta}</Text>
      </View>
      {selectable ? <View style={[styles.check, selected && styles.checkSelected]}>
        <Animated.View style={{ opacity }}><LineIcon color={colors.white} name="check" size={16} /></Animated.View>
      </View> : <LineIcon name="chevron-right" size={20} />}
    </Pressable>
  );
});
const styles = StyleSheet.create({
  item: { alignItems: 'center', borderBottomColor: colors.line, borderBottomWidth: 1, flexDirection: 'row', gap: spacing.lg, minHeight: 88, paddingVertical: spacing.xl },
  pressed: { backgroundColor: colors.surfaceMuted }, disabled: { opacity: 0.55 },
  copy: { flex: 1, minWidth: 0 }, title: { ...type.heading, fontSize: 20, lineHeight: 26 },
  description: { ...type.caption, marginTop: spacing.xs }, meta: { ...type.caption, marginTop: spacing.sm },
  check: { alignItems: 'center', justifyContent: 'center', borderColor: colors.ink, borderRadius: radii.pill, borderWidth: 1, height: 24, width: 24 },
  checkSelected: { backgroundColor: colors.ink },
});
