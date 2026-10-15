import Constants from 'expo-constants';
import * as SecureStore from 'expo-secure-store';

const hostUri = Constants.expoConfig?.hostUri ?? '';
const lanHost = hostUri.split(':')[0];

const configuredApiUrl = process.env.EXPO_PUBLIC_API_URL?.replace(/\/+$/, '');
if (!__DEV__) {
  let validProductionUrl = false;
  try {
    validProductionUrl = Boolean(configuredApiUrl && new URL(configuredApiUrl).protocol === 'https:');
  } catch {
    // A release must fail immediately if its API address was omitted or malformed.
  }
  if (!validProductionUrl) throw new Error('EXPO_PUBLIC_API_URL must be a valid HTTPS URL for release builds');
}
export const apiBaseUrl = configuredApiUrl ?? (lanHost ? `http://${lanHost}:8080` : 'http://localhost:8080');

const accessTokenKey = 'woordenboom.accessToken';
const refreshTokenKey = 'woordenboom.refreshToken';
let refreshPromise: Promise<boolean> | null = null;
let authenticationLostHandler: (() => void) | null = null;

export async function postJSON<T>(path: string, body: unknown): Promise<T> {
  return requestJSON<T>(path, { method: 'POST', body: JSON.stringify(body) });
}

export async function postPublicJSON<T>(path: string, body: unknown): Promise<T> {
  return requestJSON<T>(path, { method: 'POST', body: JSON.stringify(body) }, false);
}

export async function getJSON<T>(path: string): Promise<T> {
  return requestJSON<T>(path, { cache: 'no-store' });
}

export async function saveTokens(accessToken: string, refreshToken: string) {
  await Promise.all([
    setStorageItem(accessTokenKey, accessToken),
    setStorageItem(refreshTokenKey, refreshToken),
  ]);
}

export async function clearTokens() {
  await Promise.all([
    setStorageItem(accessTokenKey, null),
    setStorageItem(refreshTokenKey, null),
  ]);
}

export async function getStoredRefreshToken() {
  return getStorageItem(refreshTokenKey);
}

export function setAuthenticationLostHandler(handler: (() => void) | null) {
  authenticationLostHandler = handler;
}

async function requestJSON<T>(path: string, init: RequestInit = {}, authenticated = true, retry = true): Promise<T> {
  const accessToken = authenticated ? await getStorageItem(accessTokenKey) : null;
  const response = await fetch(`${apiBaseUrl}${path}`, {
    ...init,
    headers: {
      'Content-Type': 'application/json',
      ...init.headers,
      ...(accessToken ? { Authorization: `Bearer ${accessToken}` } : {}),
    },
  });

  if (response.status === 401 && authenticated && retry && await refreshSession()) {
    return requestJSON<T>(path, init, authenticated, false);
  }

  return parseJSONResponse<T>(response);
}

async function refreshSession() {
  if (!refreshPromise) {
    refreshPromise = refreshAccessToken().finally(() => {
      refreshPromise = null;
    });
  }
  return refreshPromise;
}

async function refreshAccessToken() {
  const refreshToken = await getStorageItem(refreshTokenKey);
  if (!refreshToken) {
    return false;
  }
  const response = await fetch(`${apiBaseUrl}/auth/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refreshToken }),
  });
  if (!response.ok) {
    await clearTokens();
    authenticationLostHandler?.();
    return false;
  }
  const session = await response.json() as { accessToken: string; refreshToken: string };
  await saveTokens(session.accessToken, session.refreshToken);
  return true;
}

async function getStorageItem(key: string) {
  if (process.env.EXPO_OS === 'web') {
    return typeof localStorage === 'undefined' ? null : localStorage.getItem(key);
  }
  return SecureStore.getItemAsync(key);
}

async function setStorageItem(key: string, value: string | null) {
  if (process.env.EXPO_OS === 'web') {
    if (typeof localStorage !== 'undefined') {
      if (value === null) {
        localStorage.removeItem(key);
      } else {
        localStorage.setItem(key, value);
      }
    }
    return;
  }
  if (value === null) {
    await SecureStore.deleteItemAsync(key);
  } else {
    await SecureStore.setItemAsync(key, value);
  }
}

async function parseJSONResponse<T>(response: Response): Promise<T> {
  if (response.status === 204) {
    return undefined as T;
  }
  const payload = await response.json();

  if (!response.ok) {
    const message = typeof payload.error === 'string' ? payload.error : 'request_failed';
    throw new Error(message);
  }

  return payload as T;
}
