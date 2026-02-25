import { useParams, useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../lib/api'
import { useAuth } from '../contexts/Auth'
import { Heart, Trash2, X } from 'lucide-react'

export default function PhotoDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()
  const queryClient = useQueryClient()

  const { data: photo, isLoading } = useQuery({
    queryKey: ['photo', id],
    queryFn: () => api.getPhoto(Number(id)),
    enabled: !!id,
  })

  const favoriteMutation = useMutation({
    mutationFn: () => api.toggleFavorite(Number(id)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['photo', id] })
      queryClient.invalidateQueries({ queryKey: ['photos'] })
    },
  })

  const deleteMutation = useMutation({
    mutationFn: () => api.deletePhoto(Number(id)),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['photos'] })
      navigate('/')
    },
  })

  if (isLoading || !photo) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900 dark:border-white"></div>
      </div>
    )
  }

  return (
    <div className="fixed inset-0 z-50 bg-black/90 flex items-center justify-center">
      <button
        onClick={() => navigate(-1)}
        className="absolute top-4 right-4 p-2 text-white hover:bg-white/10 rounded-full"
      >
        <X className="w-6 h-6" />
      </button>

      <div className="flex flex-col md:flex-row max-w-5xl w-full h-full md:h-auto bg-white dark:bg-black">
        <div className="flex-1 bg-black flex items-center justify-center">
          <img
            src={api.getImageUrl(photo.filename)}
            alt=""
            className="max-w-full max-h-[70vh] md:max-h-[90vh] object-contain"
          />
        </div>

        <div className="md:w-80 flex flex-col border-t md:border-t-0 md:border-l border-gray-200 dark:border-gray-800">
          <div className="p-4 border-b border-gray-200 dark:border-gray-800">
            <p className="font-semibold">{photo.username}</p>
          </div>

          <div className="flex-1 p-4 overflow-auto">
            {photo.description && (
              <p className="mb-2">
                <span className="font-semibold">{photo.username}</span> {photo.description}
              </p>
            )}
            {photo.nui_names.length > 0 && (
              <p className="text-sm text-gray-500">
                NUIs: {photo.nui_names.join(', ')}
              </p>
            )}
            {photo.taken_at && (
              <p className="text-sm text-gray-500">
                Taken: {photo.taken_at}
              </p>
            )}
          </div>

          <div className="p-4 border-t border-gray-200 dark:border-gray-800 flex items-center gap-4">
            <button
              onClick={() => favoriteMutation.mutate()}
              className="p-2 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-full"
            >
              <Heart
                className={`w-6 h-6 ${photo.is_favorite ? 'fill-red-500 text-red-500' : ''}`}
              />
            </button>

            {user && user.id === photo.user_id && (
              <button
                onClick={() => {
                  if (confirm('Delete this photo?')) {
                    deleteMutation.mutate()
                  }
                }}
                className="p-2 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-full text-red-500"
              >
                <Trash2 className="w-6 h-6" />
              </button>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
