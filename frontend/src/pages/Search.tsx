import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { api, getAvatarUrl } from '../lib/api'
import { Search, X } from 'lucide-react'

export default function SearchPage() {
  const [query, setQuery] = useState('')
  const [searchQuery, setSearchQuery] = useState('')

  const { data: users, isLoading } = useQuery({
    queryKey: ['searchUsers', searchQuery],
    queryFn: () => api.searchUsers(searchQuery),
    enabled: searchQuery.length > 0,
  })

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault()
    if (query.trim()) {
      setSearchQuery(query.trim())
    }
  }

  const handleClear = () => {
    setQuery('')
    setSearchQuery('')
  }

  return (
    <div className="min-h-screen py-4 px-4">
      <div className="max-w-lg mx-auto">
        <form onSubmit={handleSearch} className="mb-6">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-gray-400" />
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              placeholder="Search users..."
              className="w-full pl-10 pr-10 py-3 border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-900 rounded-lg focus:outline-none focus:border-gray-400"
              autoFocus
            />
            {query && (
              <button
                type="button"
                onClick={handleClear}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
              >
                <X className="w-5 h-5" />
              </button>
            )}
          </div>
        </form>

        {isLoading && (
          <div className="flex items-center justify-center py-8">
            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-gray-900 dark:border-white"></div>
          </div>
        )}

        {users && users.length === 0 && searchQuery && !isLoading && (
          <div className="text-center py-8">
            <p className="text-gray-500 dark:text-gray-400">No users found</p>
          </div>
        )}

        {users && users.length > 0 && (
          <div className="space-y-2">
            {users.map((user) => (
              <Link
                key={user.id}
                to={`/user/${user.username}`}
                className="flex items-center gap-3 p-3 rounded-lg hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
              >
                {user.avatar ? (
                  <img
                    src={getAvatarUrl(user.avatar)}
                    alt={user.username}
                    className="w-12 h-12 rounded-full object-cover"
                  />
                ) : (
                  <div className="w-12 h-12 rounded-full bg-gray-200 dark:bg-gray-800 flex items-center justify-center text-lg font-semibold">
                    {user.username[0].toUpperCase()}
                  </div>
                )}
                <div>
                  <p className="font-semibold">{user.username}</p>
                  {user.photo_count !== undefined && (
                    <p className="text-sm text-gray-500">
                      {user.photo_count} {user.photo_count === 1 ? 'post' : 'posts'}
                    </p>
                  )}
                </div>
              </Link>
            ))}
          </div>
        )}

        {!searchQuery && (
          <div className="text-center py-8">
            <p className="text-gray-500 dark:text-gray-400">
              Search for users by username
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
