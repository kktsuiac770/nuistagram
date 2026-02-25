import { useQuery } from '@tanstack/react-query'
import { Link, Navigate } from 'react-router-dom'
import { api } from '../lib/api'
import { useAuth } from '../contexts/Auth'

export default function Favorites() {
  const { user } = useAuth()

  const { data, isLoading } = useQuery({
    queryKey: ['favorites'],
    queryFn: () => api.getPhotos(1),
  })

  if (!user) {
    return <Navigate to="/login" />
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900 dark:border-white"></div>
      </div>
    )
  }

  const favorites = data?.photos.filter(p => p.is_favorite) || []

  if (!favorites.length) {
    return (
      <div className="flex flex-col items-center justify-center py-20">
        <p className="text-gray-500 mb-4">No favorites yet</p>
        <p className="text-sm text-gray-400">Photos you like will appear here</p>
      </div>
    )
  }

  return (
    <div className="py-4">
      <h1 className="text-xl font-semibold mb-4 px-4">Favorites</h1>
      <div className="grid grid-cols-3 gap-1 sm:gap-4">
        {favorites.map((photo) => (
          <Link
            key={photo.id}
            to={`/photo/${photo.id}`}
            className="aspect-square relative group overflow-hidden"
          >
            <img
              src={api.getThumbnailUrl(photo.thumbnail) || api.getImageUrl(photo.filename)}
              alt=""
              className="w-full h-full object-cover"
            />
          </Link>
        ))}
      </div>
    </div>
  )
}
