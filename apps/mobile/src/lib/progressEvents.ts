// Invalidate visible progress after a confirmed write, even if the study screen
// was closed while that request was in flight. Never increment counts locally.
const listeners = new Set<() => void>();
export function notifyProgressChanged() {
  listeners.forEach((listener) => listener());
}
export function subscribeToProgressChanges(listener: () => void) {
  listeners.add(listener);
  return () => { listeners.delete(listener); };
}
