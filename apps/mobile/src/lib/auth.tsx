import { createContext, PropsWithChildren, useContext, useEffect, useState } from 'react';

import { clearTokens, getJSON, getStoredRefreshToken, postPublicJSON, saveTokens, setAuthenticationLostHandler } from '@/lib/api';

export type AuthUser = {
  id: string;
  email: string;
  displayName?: string;
  ageRange?: string;
  profileComplete: boolean;
};

type AuthContextValue = {
  isLoading: boolean;
  user: AuthUser | null;
  completeLogin: (accessToken: string, refreshToken: string, user: AuthUser) => Promise<void>;
  refreshUser: () => Promise<void>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthContextValue | null>(null);

export function AuthProvider({ children }: PropsWithChildren) {
  const [isLoading, setIsLoading] = useState(true);
  const [user, setUser] = useState<AuthUser | null>(null);

  async function refreshUser() {
    const result = await getJSON<{ user: AuthUser }>('/me');
    setUser(result.user);
  }

  useEffect(() => {
    setAuthenticationLostHandler(() => setUser(null));

    async function restoreSession() {
      try {
        if (await getStoredRefreshToken()) {
          await refreshUser();
        }
      } catch {
        await clearTokens();
        setUser(null);
      } finally {
        setIsLoading(false);
      }
    }

    restoreSession();
    return () => setAuthenticationLostHandler(null);
  }, []);

  async function completeLogin(accessToken: string, refreshToken: string, authenticatedUser: AuthUser) {
    await saveTokens(accessToken, refreshToken);
    setUser(authenticatedUser);
  }

  async function logout() {
    const refreshToken = await getStoredRefreshToken();
    if (refreshToken) {
      try {
        await postPublicJSON('/auth/logout', { refreshToken });
      } catch {
        // Local credentials are still cleared if the server is unavailable.
      }
    }
    await clearTokens();
    setUser(null);
  }

  return (
    <AuthContext.Provider value={{ completeLogin, isLoading, logout, refreshUser, user }}>
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth() {
  const value = useContext(AuthContext);
  if (!value) {
    throw new Error('useAuth must be used inside AuthProvider');
  }
  return value;
}
