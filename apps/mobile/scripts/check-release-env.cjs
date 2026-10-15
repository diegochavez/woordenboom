// EAS runs this before dependency installation, so a release without an API
// address fails before anyone can distribute a build pointing at localhost.
if (['preview', 'production'].includes(process.env.EAS_BUILD_PROFILE)) {
  const value = process.env.EXPO_PUBLIC_API_URL;
  let url;
  try {
    url = new URL(value);
  } catch {
    throw new Error('Set EXPO_PUBLIC_API_URL in the matching EAS environment before building.');
  }
  if (url.protocol !== 'https:' || !url.hostname || ['localhost', '127.0.0.1'].includes(url.hostname) || url.username || url.password) {
    throw new Error('EXPO_PUBLIC_API_URL must be a public HTTPS API address without embedded credentials.');
  }
}
