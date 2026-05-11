import { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import { Navigate, useLocation } from "react-router-dom";
import type { ReactNode } from "react";
import { api } from "./api";
import type { Principal } from "../gen/apps/webapp/v1/api_pb";

type AuthState = {
  principal: Principal | null;
  loading: boolean;
  refresh: () => Promise<void>;
  login: (email: string, password: string) => Promise<void>;
  register: (email: string, name: string, password: string) => Promise<void>;
  logout: () => Promise<void>;
};

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [principal, setPrincipal] = useState<Principal | null>(null);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const res = await api.getSession({});
      setPrincipal(res.authenticated ? res.principal ?? null : null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  const login = useCallback(async (email: string, password: string) => {
    const res = await api.login({ email, password });
    setPrincipal(res.principal ?? null);
  }, []);

  const register = useCallback(async (email: string, name: string, password: string) => {
    const res = await api.register({ email, name, password });
    setPrincipal(res.principal ?? null);
  }, []);

  const logout = useCallback(async () => {
    await api.logout({});
    setPrincipal(null);
  }, []);

  const value = useMemo<AuthState>(
    () => ({ principal, loading, refresh, login, register, logout }),
    [principal, loading, refresh, login, register, logout],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth() {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth must be used inside <AuthProvider>");
  return ctx;
}

export function ProtectedRoute({ children }: { children: ReactNode }) {
  const { principal, loading } = useAuth();
  const location = useLocation();
  if (loading) return <div className="p-8 text-neutral-500">Loading…</div>;
  if (!principal) return <Navigate to="/login" replace state={{ next: location.pathname }} />;
  return <>{children}</>;
}
