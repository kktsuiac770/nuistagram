import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useAuth } from '../contexts/Auth'
import { validatePassword } from '../lib/api'

export default function Register() {
  const [username, setUsername] = useState('')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()
  const { register } = useAuth()

  const passwordError = validatePassword(password)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    if (passwordError) {
      setError(passwordError)
      return
    }

    setLoading(true)

    try {
      await register(username, password)
      navigate('/')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Registration failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="min-h-[80vh] flex items-center justify-center px-4">
      <div className="w-full max-w-sm">
        <div className="border border-gray-300 dark:border-gray-700 bg-white dark:bg-black p-10 rounded-lg">
          <h1 className="text-4xl font-serif text-center mb-6">NUIstagram</h1>
          <p className="text-gray-500 text-center mb-6 text-sm">Sign up to see photos from your friends.</p>
          
          {error && (
            <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 text-sm text-center rounded">
              {error}
            </div>
          )}

          <form onSubmit={handleSubmit} className="space-y-2">
            <input
              type="text"
              placeholder="Username"
              value={username}
              onChange={e => setUsername(e.target.value)}
              required
              minLength={3}
              maxLength={50}
              className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-900 rounded focus:outline-none focus:border-gray-400"
            />
            <input
              type="password"
              placeholder="Password"
              value={password}
              onChange={e => setPassword(e.target.value)}
              required
              className="w-full px-3 py-2 text-sm border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-900 rounded focus:outline-none focus:border-gray-400"
            />
            {password.length > 0 && passwordError && (
              <p className="text-xs text-red-500 mt-1">{passwordError}</p>
            )}
            <button
              type="submit"
              disabled={loading || !username.trim() || !password || !!passwordError}
              className="w-full py-2 text-sm font-semibold bg-blue-500 text-white rounded hover:bg-blue-600 disabled:opacity-50 transition-colors"
            >
              {loading ? 'Signing up...' : 'Sign up'}
            </button>
          </form>
        </div>

        <div className="border border-gray-300 dark:border-gray-700 bg-white dark:bg-black p-5 mt-2 text-center rounded-lg">
          <p className="text-sm">
            Have an account?{' '}
            <Link to="/login" className="text-blue-500 font-semibold">
              Log in
            </Link>
          </p>
        </div>
      </div>
    </div>
  )
}
