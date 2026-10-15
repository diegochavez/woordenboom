import { forwardRef, useState } from 'react';
import { Animated, Pressable, PressableProps, StyleProp, StyleSheet, Text, View, ViewStyle } from 'react-native';
import { LineIcon } from '@/components/LineIcon';
import { motion, useMotionValue } from '@/hooks/useMotion';
import { colors, radii } from '@/lib/theme';

const AnimatedPressable = Animated.createAnimatedComponent(Pressable);

type OutlineButtonProps = Omit<PressableProps, 'style'> & {
  icon?: React.ComponentProps<typeof LineIcon>['name'];
  label: string;
  variant?: 'outline' | 'primary' | 'text';
  inverse?: boolean;
  style?: StyleProp<ViewStyle>;
};
export const OutlineButton = forwardRef<View, OutlineButtonProps>(function OutlineButton({ disabled, icon, label, variant = 'outline', inverse = false, style, accessibilityState, onPressIn, onPressOut, ...props }, ref) {
  const [pressed, setPressed] = useState(false);
  const opacity = useMotionValue(disabled ? 0.45 : pressed ? 0.65 : 1, motion.press);
  return (
    <AnimatedPressable aria-disabled={Boolean(disabled)} accessibilityRole="button" accessibilityState={{ ...accessibilityState, disabled: Boolean(disabled) }} disabled={disabled} ref={ref}
      onPressIn={(event) => { setPressed(true); onPressIn?.(event); }} onPressOut={(event) => { setPressed(false); onPressOut?.(event); }}
      style={[styles.outline, variant === 'primary' && styles.primary, variant === 'text' && styles.textButton, inverse && styles.inverseBorder, inverse && variant === 'primary' && styles.inversePrimary, style, { opacity }]} {...props}>
      {icon ? <LineIcon color={inverse ? (variant === 'primary' ? colors.black : colors.white) : (variant === 'primary' ? colors.white : colors.ink)} name={icon} size={20} /> : null}
      <Text style={[styles.label, variant === 'primary' && styles.primaryLabel, inverse && styles.inverseLabel, inverse && variant === 'primary' && styles.inversePrimaryLabel]}>{label}</Text>
    </AnimatedPressable>
  );
});
type CircleButtonProps = Omit<PressableProps, 'style'> & { icon: React.ComponentProps<typeof LineIcon>['name']; style?: StyleProp<ViewStyle>; inverse?: boolean };
export const CircleButton = forwardRef<View, CircleButtonProps>(function CircleButton({ disabled, icon, style, inverse = false, onPressIn, onPressOut, ...props }, ref) {
  const [pressed, setPressed] = useState(false);
  const opacity = useMotionValue(disabled ? 0.45 : pressed ? 0.65 : 1, motion.press);
  return (
    <AnimatedPressable aria-disabled={Boolean(disabled)} accessibilityRole="button" accessibilityLabel={icon === 'arrow-left' ? 'Back' : undefined} accessibilityState={{ disabled: Boolean(disabled) }} disabled={disabled} ref={ref}
      onPressIn={(event) => { setPressed(true); onPressIn?.(event); }} onPressOut={(event) => { setPressed(false); onPressOut?.(event); }}
      style={[styles.circle, style, { opacity }]} {...props}>
      <LineIcon color={inverse ? colors.white : colors.ink} name={icon} size={24} />
    </AnimatedPressable>
  );
});
const styles = StyleSheet.create({
  outline: { alignItems: 'center', borderColor: colors.ink, borderRadius: radii.pill, borderWidth: 1, flexDirection: 'row', gap: 8, justifyContent: 'center', minHeight: 56, paddingHorizontal: 20, paddingVertical: 14 },
  label: { color: colors.ink, fontSize: 16, fontWeight: '500', textAlign: 'center', flexShrink: 1 },
  inverseBorder: { borderColor: colors.white }, inversePrimary: { backgroundColor: colors.white },
  inverseLabel: { color: colors.white }, inversePrimaryLabel: { color: colors.black },
  primary: { backgroundColor: colors.ink }, primaryLabel: { color: colors.white },
  textButton: { borderWidth: 0, minHeight: 44, paddingHorizontal: 8, paddingVertical: 10 },
  circle: { alignItems: 'center', borderRadius: radii.pill, height: 44, justifyContent: 'center', width: 44 },
  pressed: { opacity: 0.65 }, disabled: { opacity: 0.45 },
});
