import { Outlet, Link, useLocation } from 'react-router-dom'
import { useAuth } from './contexts/Auth'
import { Home, PlusSquare, Heart, LogOut, Instagram } from 'lucide-react'

export default function Layout() {
  const { user, loading, logout } = useAuth()
  const location = useLocation()

  const navItems = [
    { path: '/', icon: Home, label: 'Home' },
    { path: '/favorites', icon: Heart, label: 'Favorites' },
  ]

  if (user) {
    navItems.push({ path: '/upload', icon: PlusSquare, label: 'Upload' })
  }

  if (loading) {
    return (
      <div className="min-h-screen bg-white dark:bg-black flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900 dark:border-white"></div>
      </div>
    )
  }

  return (
    <div className="min-h-screen bg-white dark:bg-black">
      <header className="sticky top-0 z-50 border-b border-gray-200 dark:border-gray-800 bg-white dark:bg-black">
        <div className="max-w-[975px] mx-auto px-4 h-14 flex items-center justify-between">
          <Link to="/" className="flex items-center gap-2">
            <Instagram className="w-7 h-7" />
            <span className="text-xl font-semibold hidden sm:block">NUIstagram</span>
          </Link>

          <nav className="flex items-center gap-1 sm:gap-4">
            {navItems.map(({ path, icon: Icon, label }) => (
              <Link
                key={path}
                to={path}
                className={`p-2 rounded-lg transition-colors ${
                  location.pathname === path
                    ? 'bg-gray-100 dark:bg-gray-800'
                    : 'hover:bg-gray-50 dark:hover:bg-gray-900'
                }`}
                title={label}
              >
                <Icon className="w-6 h-6" />
              </Link>
            ))}

            {user ? (
              <button
                onClick={logout}
                className="p-2 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-900 transition-colors"
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

      <main className="max-w-[975px] mx-auto">
        <Outlet />
      </main>
    </div>
  )
}
