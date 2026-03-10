export interface Photo {
  id: number
  filename: string
  thumbnail: string
  user_id: number
  description: string
  taken_at: string
  created_at: string
  is_favorite: boolean
  nui_names: string[]
  username?: string
  like_count: number
  is_liked: boolean
  comment_count: number
}

export interface Pagination {
  current_page: number
  total_pages: number
  total_count: number
  has_prev: boolean
  has_next: boolean
  pages: number[]
}

export interface PhotosResponse {
  photos: Photo[]
  current_page: number
  total_pages: number
  total_count: number
  has_prev: boolean
  has_next: boolean
  pages: number[]
}

export interface Nui {
  id: number
  name: string
}

export interface User {
  id: number
  username: string
  photo_count?: number
  bio?: string
  avatar?: string
  follower_count?: number
  following_count?: number
  is_following?: boolean
}

export interface UserProfile {
  id: number
  username: string
  photo_count: number
  bio: string
  avatar: string
  follower_count: number
  following_count: number
  is_following: boolean
}

export interface Comment {
  id: number
  photo_id: number
  user_id: number
  content: string
  created_at: string
  username: string
}

export interface Notification {
  id: number
  type: 'follow' | 'like' | 'comment'
  read: boolean
  created_at: string
  actor: {
    id: number
    username: string
    avatar: string
  }
  photo_id?: number
  comment_id?: number
}

export interface TokenResponse {
  access_token: string
  refresh_token: string
  expires_in: number
  token_type: string
}

class ApiError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'ApiError'
  }
}

// Token storage
const ACCESS_TOKEN_KEY = 'nuistagram_access_token'
const REFRESH_TOKEN_KEY = 'nuistagram_refresh_token'

export function getAccessToken(): string | null {
  return localStorage.getItem(ACCESS_TOKEN_KEY)
}

export function setTokens(accessToken: string, refreshToken: string): void {
  localStorage.setItem(ACCESS_TOKEN_KEY, accessToken)
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken)
}

export function clearTokens(): void {
  localStorage.removeItem(ACCESS_TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
}

// Coalesces concurrent refresh attempts into a single in-flight request
let refreshPromise: Promise<boolean> | null = null

async function tryRefresh(): Promise<boolean> {
  if (refreshPromise) return refreshPromise
  refreshPromise = (async () => {
    const rt = localStorage.getItem(REFRESH_TOKEN_KEY)
    if (!rt) return false
    try {
      const res = await fetch('/api/refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: rt }),
      })
      if (!res.ok) return false
      const data: TokenResponse = await res.json()
      setTokens(data.access_token, data.refresh_token)
      return true
    } catch {
      return false
    } finally {
      refreshPromise = null
    }
  })()
  return refreshPromise
}

async function fetchJson<T>(url: string, options?: RequestInit): Promise<T> {
  const token = getAccessToken()
  const res = await fetch(url, {
    ...options,
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  })

  if (res.status === 401) {
    const refreshed = await tryRefresh()
    if (refreshed) {
      const newToken = getAccessToken()
      const retry = await fetch(url, {
        ...options,
        headers: {
          ...(newToken ? { Authorization: `Bearer ${newToken}` } : {}),
          ...options?.headers,
        },
      })
      if (!retry.ok) {
        clearTokens()
        const text = await retry.text()
        try {
          const json = JSON.parse(text)
          throw new ApiError(json.error || 'Request failed')
        } catch (e) {
          if (e instanceof ApiError) throw e
          throw new ApiError(text || `Request failed: ${retry.status}`)
        }
      }
      return retry.json()
    }
    clearTokens()
    throw new ApiError('Unauthorized')
  }

  if (!res.ok) {
    const text = await res.text()
    try {
      const json = JSON.parse(text)
      throw new ApiError(json.error || 'Request failed')
    } catch (e) {
      if (e instanceof ApiError) throw e
      throw new ApiError(text || `Request failed: ${res.status}`)
    }
  }
  return res.json()
}

export const passwordRequirements = {
  minLength: 8,
  requiresUpper: true,
  requiresLower: true,
  requiresDigit: true,
}

export function validatePassword(password: string): string | null {
  if (password.length < passwordRequirements.minLength) {
    return `Password must be at least ${passwordRequirements.minLength} characters long`
  }
  if (passwordRequirements.requiresUpper && !/[A-Z]/.test(password)) {
    return 'Password must contain at least one uppercase letter'
  }
  if (passwordRequirements.requiresLower && !/[a-z]/.test(password)) {
    return 'Password must contain at least one lowercase letter'
  }
  if (passwordRequirements.requiresDigit && !/[0-9]/.test(password)) {
    return 'Password must contain at least one digit'
  }
  return null
}

export function getAvatarUrl(avatar?: string): string {
  if (!avatar) return ''
  if (avatar.startsWith('http')) return avatar
  return `/static/uploads/${avatar}`
}

export const api = {
  getPhotos: (page = 1, tags?: string[], feed?: 'all' | 'following') => {
    let url = `/api/photos?page=${page}`
    if (tags && tags.length > 0) {
      url += `&tags=${encodeURIComponent(tags.join(','))}`
    }
    if (feed) {
      url += `&feed=${feed}`
    }
    return fetchJson<PhotosResponse>(url)
  },

  getPhoto: (id: number) =>
    fetchJson<Photo>(`/api/photo/${id}`),

  toggleFavorite: (id: number) =>
    fetchJson<{ success: boolean; is_favorite: boolean }>(`/api/photo/${id}/favorite`, { method: 'POST' }),

  toggleLike: (id: number) =>
    fetchJson<{ success: boolean; is_liked: boolean; like_count: number }>(`/api/photo/${id}/like`, { method: 'POST' }),

  getLikers: (id: number, limit = 20) =>
    fetchJson<User[]>(`/api/photo/${id}/likers?limit=${limit}`),

  getComments: (id: number, limit = 20, offset = 0) =>
    fetchJson<Comment[]>(`/api/photo/${id}/comments?limit=${limit}&offset=${offset}`),

  createComment: (photoId: number, content: string) => {
    const form = new URLSearchParams()
    form.append('content', content)
    return fetchJson<{ success: boolean; comment_id: number }>(`/api/photo/${photoId}/comment`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: form.toString(),
    })
  },

  deleteComment: (id: number) =>
    fetchJson<{ success: boolean }>(`/api/comment/${id}`, { method: 'POST' }),

  getNuis: () =>
    fetchJson<Nui[]>('/api/nuis'),

  getMe: () =>
    fetchJson<User>('/api/me'),

  getUser: (username: string) =>
    fetchJson<UserProfile>(`/api/user/${encodeURIComponent(username)}`),

  getUserPhotos: (username: string, page = 1) =>
    fetchJson<PhotosResponse>(`/api/user/${encodeURIComponent(username)}/photos?page=${page}`),

  followUser: (username: string) =>
    fetchJson<{ success: boolean }>(`/api/user/${encodeURIComponent(username)}/follow`, { method: 'POST' }),

  unfollowUser: (username: string) =>
    fetchJson<{ success: boolean }>(`/api/user/${encodeURIComponent(username)}/unfollow`, { method: 'POST' }),

  getFollowStatus: (username: string) =>
    fetchJson<{ is_following: boolean }>(`/api/user/${encodeURIComponent(username)}/follow-status`),

  searchUsers: (query: string) =>
    fetchJson<User[]>(`/api/search/users?q=${encodeURIComponent(query)}`),

  updateProfile: (bio: string) => {
    const form = new URLSearchParams()
    form.append('bio', bio)
    return fetchJson<{ success: boolean }>('/api/profile', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: form.toString(),
    })
  },

  uploadAvatar: (formData: FormData) =>
    fetchJson<{ success: boolean; avatar: string }>('/api/avatar', {
      method: 'POST',
      body: formData,
    }),

  getNotifications: (limit = 20, offset = 0) =>
    fetchJson<Notification[]>(`/api/notifications?limit=${limit}&offset=${offset}`),

  getUnreadCount: () =>
    fetchJson<{ count: number }>('/api/notifications/unread'),

  markNotificationRead: (id: number) =>
    fetchJson<{ success: boolean }>(`/api/notifications/${id}/read`, { method: 'POST' }),

  markAllNotificationsRead: () =>
    fetchJson<{ success: boolean }>('/api/notifications/read-all', { method: 'POST' }),

  login: async (username: string, password: string): Promise<TokenResponse> => {
    const form = new URLSearchParams()
    form.append('username', username)
    form.append('password', password)
    const data = await fetchJson<TokenResponse>('/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: form.toString(),
    })
    setTokens(data.access_token, data.refresh_token)
    return data
  },

  register: async (username: string, password: string): Promise<TokenResponse> => {
    const form = new URLSearchParams()
    form.append('username', username)
    form.append('password', password)
    const data = await fetchJson<TokenResponse>('/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: form.toString(),
    })
    setTokens(data.access_token, data.refresh_token)
    return data
  },

  logout: async (): Promise<{ success: boolean }> => {
    const rt = localStorage.getItem(REFRESH_TOKEN_KEY)
    const result = await fetchJson<{ success: boolean }>('/logout', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: rt ?? '' }),
    })
    clearTokens()
    return result
  },

  upload: (formData: FormData) =>
    fetchJson<{ success: boolean }>('/upload', {
      method: 'POST',
      body: formData,
    }),

  deletePhoto: (id: number) =>
    fetchJson<{ success: boolean }>(`/photo/${id}/delete`, { method: 'POST' }),

  updatePhoto: (id: number, formData: FormData) =>
    fetchJson<{ success: boolean }>(`/photo/${id}/edit`, {
      method: 'POST',
      body: formData,
    }),

  getImageUrl: (filename: string) =>
    filename.startsWith('http') ? filename : `/static/uploads/${filename}`,

  getThumbnailUrl: (thumbnail: string) =>
    thumbnail
      ? thumbnail.startsWith('http') ? thumbnail : `/static/uploads/${thumbnail}`
      : '',
}
