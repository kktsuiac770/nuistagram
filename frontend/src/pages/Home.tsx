import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../lib/api'
import { useAuth } from '../contexts/Auth'
import { Heart } from 'lucide-react'

export default function Home() {
  const [page, setPage] = useState(1)
  const { user } = useAuth()

  const { data, isLoading, error } = useQuery({
    queryKey: ['photos', page],
    queryFn: () => api.getPhotos(page),
  })

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center py-20">
        <p className="text-gray-500 mb-4">Failed to load photos</p>
        <button 
          onClick={() => window.location.reload()}
          className="px-4 py-2 bg-blue-500 text-white rounded-lg"
        >
          Retry
        </button>
      </div>
    )
  }

  if (isLoading || !data) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900 dark:border-white"></div>
      </div>
    )
  }

  if (!data.photos || !data.photos.length) {
    return (
      <div className="flex flex-col items-center justify-center py-20">
        <p className="text-gray-500 mb-4">No photos yet</p>
        {user && (
          <Link to="/upload" className="px-4 py-2 bg-blue-500 text-white rounded-lg">
            Upload your first photo
          </Link>
        )}
      </div>
    )
  }

  return (
    <div className="py-4">
      <div className="grid grid-cols-3 gap-1 sm:gap-4">
        {data.photos.map((photo) => (
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
            <div className="absolute inset-0 bg-black/50 opacity-0 group-hover:opacity-100 transition-opacity flex items-center justify-center">
              <div className="text-white flex items-center gap-4">
                {photo.is_favorite && <Heart className="w-6 h-6 fill-red-500 text-red-500" />}
              </div>
            </div>
          </Link>
        ))}
      </div>

      {data.total_pages > 1 && (
        <div className="flex justify-center gap-2 mt-8">
          <button
            onClick={() => setPage(p => Math.max(1, p - 1))}
            disabled={!data.has_prev}
            className="px-4 py-2 rounded-lg border disabled:opacity-50"
          >
            Previous
          </button>
          <span className="px-4 py-2">
            {data.current_page} / {data.total_pages}
          </span>
          <button
            onClick={() => setPage(p => p + 1)}
            disabled={!data.has_next}
            className="px-4 py-2 rounded-lg border disabled:opacity-50"
          >
            Next
          </button>
        </div>
      )}
    </div>
  )
}
