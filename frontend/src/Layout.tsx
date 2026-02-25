import { Outlet, Link, useLocation } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { useAuth } from './contexts/Auth'
import { useTheme } from './contexts/Theme'
import { Home, PlusSquare, Heart, LogOut, Instagram, Sun, Moon, Search, Bell, User } from 'lucide-react'
import { api } from './lib/api'

export default function Layout() {
  const { user, loading, logout } = useAuth()
  const { theme, toggleTheme } = useTheme()
  const location = useLocation()

  const { data: unreadData } = useQuery({
    queryKey: ['unreadCount'],
    queryFn: api.getUnreadCount,
    enabled: !!user,
    refetchInterval: 60000,
  })

  const navItems: Array<{ path: string; icon: typeof Home; label: string; badge?: number }> = [
    { path: '/', icon: Home, label: 'Home' },
    { path: '/search', icon: Search, label: 'Search' },
    { path: '/favorites', icon: Heart, label: 'Favorites' },
  ]

  if (user) {
    navItems.push({ path: '/notifications', icon: Bell, label: 'Notifications', badge: unreadData?.count })
    navItems.push({ path: '/upload', icon: PlusSquare, label: 'Upload' })
    navItems.push({ path: `/user/${user.username}`, icon: User, label: 'Profile' })
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-white dark:bg-black flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900 dark:border-white"></div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-white dark:bg-black text-gray-900 dark:text-white">
      <header className="sticky top-0 z-50 border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-black">
        <div className="max-w-[975px] mx-auto px-4 h-14 flex items-center justify-between">
          <Link to="/" className="flex items-center gap-2">
            <Instagram className="w-7 h-7" />
            <span className="text-xl font-semibold hidden sm:block">NUIstagram</span>
          </Link>

          <nav className="flex items-center gap-1 sm:gap-4">
            {navItems.map(({ path, icon: Icon, label, badge }) => (
              <Link
                key={path}
                to={path}
                className={`p-2 rounded-lg transition-colors hidden sm:block relative ${
                  location.pathname === path
                    ? 'bg-gray-100 dark:bg-gray-800'
                    : 'hover:bg-gray-50 dark:hover:bg-gray-900'
                }`}
                title={label}
              >
                <Icon className="w-6 h-6" />
                {badge && badge > 0 && (
                  <span className="absolute -top-1 -right-1 w-5 h-5 bg-red-500 text-white text-xs rounded-full flex items-center justify-center">
                    {badge > 9 ? '9+' : badge}
                  </span>
                )}
              </Link>
            ))}

            <button
              onClick={toggleTheme}
              className="p-2 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-900 transition-colors"
              title={theme === 'dark' ? 'Light mode' : 'Dark mode'}
            >
              {theme === 'dark' ? <Sun className="w-6 h-6" /> : <Moon className="w-6 h-6" />}
            </button>

            {user ? (
              <button
                onClick={logout}
                className="p-2 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-900 transition-colors hidden sm:block"
                title="Logout"
              >
                <LogOut className="w-6 h-6" />
              </button>
            ) : (
              <Link
                to="/login"
                className="px-4 py-1.5 text-sm font-semibold bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
              >
                Log in
              </Link>
            )}
          </nav>
        </div>
      </header>

      <main className="max-w-[975px] mx-auto pb-16 sm:pb-0">
        <Outlet />
      </main>

      <nav className="sm:hidden fixed bottom-0 left-0 right-0 border-t border-gray-200 dark:border-gray-800 bg-white dark:bg-black z-50">
        <div className="flex items-center justify-around h-14">
          {navItems.slice(0, 5).map(({ path, icon: Icon, label, badge }) => (
            <Link
              key={path}
              to={path}
              className={`flex flex-col items-center justify-center flex-1 h-full relative ${
                location.pathname === path ? 'text-blue-500' : ''
              }`}
            >
              <Icon className="w-6 h-6" />
              <span className="text-xs mt-1">{label}</span>
              {badge && badge > 0 && (
                <span className="absolute top-1 right-1/4 w-4 h-4 bg-red-500 text-white text-xs rounded-full flex items-center justify-center">
                  {badge > 9 ? '9+' : badge}
                </span>
              )}
            </Link>
          ))}
          {user && (
            <button
              onClick={logout}
              className="flex flex-col items-center justify-center flex-1 h-full"
            >
              <LogOut className="w-6 h-6" />
              <span className="text-xs mt-1">Logout</span>
            </button>
          )}
        </div>
      </nav>
    </div>
  )
}
