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
}

class ApiError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'ApiError'
  }
}

async function fetchJson<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    ...options,
    credentials: 'include',
    headers: {
      ...options?.headers,
    },
  })
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

export const api = {
  getPhotos: (page = 1) => 
    fetchJson<PhotosResponse>(`/api/photos?page=${page}`),
  
  getPhoto: (id: number) => 
    fetchJson<Photo>(`/api/photo/${id}`),
  
  toggleFavorite: (id: number) =>
    fetchJson<{ success: boolean; is_favorite: boolean }>(`/api/photo/${id}/favorite`, { method: 'POST' }),
  
  getNuis: () => 
    fetchJson<Nui[]>('/api/nuis'),
  
  getMe: () => 
    fetchJson<User>('/api/me'),
  
  login: (username: string, password: string) => {
    const form = new URLSearchParams()
    form.append('username', username)
    form.append('password', password)
    return fetchJson<{ success: boolean }>('/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: form.toString(),
    })
  },
  
  register: (username: string, password: string) => {
    const form = new URLSearchParams()
    form.append('username', username)
    form.append('password', password)
    return fetchJson<{ success: boolean }>('/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: form.toString(),
    })
  },
  
  logout: () =>
    fetchJson<{ success: boolean }>('/logout'),
  
  upload: (formData: FormData) =>
    fetchJson<{ success: boolean }>('/upload', {
      method: 'POST',
      body: formData,
    }),
  
  deletePhoto: (id: number) =>
    fetchJson<{ success: boolean }>(`/photo/${id}/delete`, { method: 'POST' }),
  
  getImageUrl: (filename: string) =>
    `/static/uploads/${filename}`,
  
  getThumbnailUrl: (thumbnail: string) =>
    thumbnail ? `/static/uploads/${thumbnail}` : '',
}
