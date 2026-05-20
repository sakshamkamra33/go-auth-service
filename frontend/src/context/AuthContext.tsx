import React, { createContext, useContext, useState, useEffect, ReactNode } from 'react';
import { fetchWithAuth, clearTokens, setTokens, getAccessToken, getRefreshToken } from '../api/client';

export interface User {
  id: string;
  username: string;
  email: string;
  role: string;
  email_verified: boolean;
  created_at: string;
}

interface AuthContextType {
  user: User | null;
  loading: boolean;
  login: (access: string, refresh: string) => Promise<void>;
  logout: () => Promise<void>;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  const refreshUser = async () => {
    if (!getAccessToken() && !getRefreshToken()) {
      setUser(null);
      setLoading(false);
      return;
    }
    try {
      const res = await fetchWithAuth('/auth/me');
      if (res.ok) {
        const data = await res.json();
        setUser(data);
      } else {
        setUser(null);
      }
    } catch (err) {
      setUser(null);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    refreshUser();
  }, []);

  const login = async (access: string, refresh: string) => {
    setTokens(access, refresh);
    await refreshUser();
  };

  const logout = async () => {
    const refresh = localStorage.getItem('refresh_token');
    try {
      if (refresh) {
        await fetchWithAuth('/auth/logout', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: refresh })
        });
      }
    } catch (e) {
      // Ignore errors on logout
    }
    clearTokens();
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ user, loading, login, logout, refreshUser }}>
      {children}
    </AuthContext.Provider>
  );
};

export const useAuth = () => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
