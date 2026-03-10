import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'
import { api, type User, getAccessToken, clearTokens } from '../lib/api'

interface AuthContextType {
  user: User | null
  loading: boolean
  login: (username: string, password: string) => Promise<void>
  register: (username: string, password: string) => Promise<void>
  logout: () => Promise<void>
}

const AuthContext = createContext<AuthContextType | null>(null)

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    if (!getAccessToken()) {
      setLoading(false)
      return
    }
    api.getMe()
      .then(setUser)
      .catch(() => {
        clearTokens()
        setUser(null)
      })
      .finally(() => setLoading(false))
  }, [])

  const login = async (username: string, password: string) => {
    await api.login(username, password)
    const u = await api.getMe()
    setUser(u)
  }

  const register = async (username: string, password: string) => {
    await api.register(username, password)
    const u = await api.getMe()
    setUser(u)
  }

  const logout = async () => {
    await api.logout()
    setUser(null)
  }

  return (
    <AuthContext.Provider value={{ user, loading, login, register, logout }}>
      {children}
    </AuthContext.Provider>
  )
}

// eslint-disable-next-line react-refresh/only-export-components
export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error('useAuth must be used within AuthProvider')
  return ctx
}
