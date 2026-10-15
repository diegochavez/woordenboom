import { useEffect, useState, useSyncExternalStore } from 'react';
import { AccessibilityInfo, Animated, Easing, Platform } from 'react-native';

export const motion = { press: 110, selection: 120, word: 160, sheet: 180, progress: 200, complete: 180 };
let reducedPreference = true;
const listeners = new Set<() => void>();
let preferenceVersion = 0;
function subscribe(listener: () => void) {
  listeners.add(listener);
  // One native listener is enough for all progress fills and controls.
  if (listeners.size === 1) {
    const version = ++preferenceVersion;
    let changed = false;
    const update = (value: boolean) => { reducedPreference = value; listeners.forEach((notify) => notify()); };
    subscription = AccessibilityInfo.addEventListener('reduceMotionChanged', (value) => { changed = true; update(value); });
    void AccessibilityInfo.isReduceMotionEnabled().then((value) => {
      if (version === preferenceVersion && !changed) update(value);
    }).catch(() => {});
  }
  return () => { listeners.delete(listener); if (!listeners.size) { ++preferenceVersion; subscription?.remove(); } };
}
let subscription: ReturnType<typeof AccessibilityInfo.addEventListener> | undefined;
export function useReducedMotion() {
  return useSyncExternalStore(subscribe, () => reducedPreference, () => true);
}
// Initial values are already rendered: only subsequent changes animate.
export function useMotionValue(target: number, duration: number, native = true) {
  const reduced = useReducedMotion();
  const [value] = useState(() => new Animated.Value(target));
  useEffect(() => {
    value.stopAnimation();
    if (reduced) { value.setValue(target); return; }
    const animation = Animated.timing(value, { toValue: target, duration,
      easing: Easing.out(Easing.quad), useNativeDriver: native && Platform.OS !== 'web', isInteraction: false });
    animation.start();
    return () => animation.stop();
  }, [target, duration, native, reduced, value]);
  return value;
}
export function useReveal(duration: number) {
  const reduced = useReducedMotion();
  const [opacity] = useState(() => new Animated.Value(reduced ? 1 : 0));
  useEffect(() => {
    if (reduced) { opacity.stopAnimation(); opacity.setValue(1); return; }
    const animation = Animated.timing(opacity, { toValue: 1, duration,
      easing: Easing.out(Easing.quad), useNativeDriver: Platform.OS !== 'web', isInteraction: false });
    animation.start();
    return () => animation.stop();
  }, [duration, opacity, reduced]);
  return opacity;
}
