import { useEffect, useState, useCallback } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../lib/api'
import { useAuth } from '../contexts/Auth'
import { useToast } from '../components/Toast'
import { Heart, Trash2, X, Edit, MessageCircle } from 'lucide-react'

export default function PhotoDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()
  const toast = useToast()
  const queryClient = useQueryClient()
  const [commentText, setCommentText] = useState('')

  const photoId = Number(id)

  const { data: photo, isLoading } = useQuery({
    queryKey: ['photo', id],
    queryFn: () => api.getPhoto(photoId),
    enabled: !!id,
  })

  const { data: comments } = useQuery({
    queryKey: ['comments', photoId],
    queryFn: () => api.getComments(photoId, 50),
    enabled: !!photoId,
  })

  const favoriteMutation = useMutation({
    mutationFn: () => api.toggleFavorite(photoId),
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['photo', id] })
      toast.success(data.is_favorite ? 'Added to favorites' : 'Removed from favorites')
    },
    onError: () => toast.error('Failed to update favorite'),
  })

  const likeMutation = useMutation({
    mutationFn: () => api.toggleLike(photoId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['photo', id] })
    },
    onError: () => toast.error('Failed to update like'),
  })

  const commentMutation = useMutation({
    mutationFn: () => api.createComment(photoId, commentText),
    onSuccess: () => {
      setCommentText('')
      queryClient.invalidateQueries({ queryKey: ['comments', photoId] })
      queryClient.invalidateQueries({ queryKey: ['photo', id] })
    },
    onError: () => toast.error('Failed to post comment'),
  })

  const deleteCommentMutation = useMutation({
    mutationFn: (commentId: number) => api.deleteComment(commentId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['comments', photoId] })
      queryClient.invalidateQueries({ queryKey: ['photo', id] })
    },
    onError: () => toast.error('Failed to delete comment'),
  })

  const deleteMutation = useMutation({
    mutationFn: () => api.deletePhoto(photoId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['photos'] })
      toast.success('Photo deleted')
      navigate('/', { replace: true })
    },
    onError: () => toast.error('Failed to delete photo'),
  })

  const handleClose = useCallback(() => {
    navigate('/', { replace: true })
  }, [navigate])

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        handleClose()
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [handleClose])

  if (isLoading || !photo) {
    return (
      <div className="fixed inset-0 z-50 bg-black flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-white"></div>
      </div>
    )
  }

  return (
    <div className="fixed inset-0 z-50 bg-black/95 flex items-center justify-center">
      <button
        onClick={handleClose}
        className="absolute top-4 right-4 p-2 text-white hover:bg-white/10 rounded-full z-10"
      >
        <X className="w-6 h-6" />
      </button>

      <div className="flex flex-col md:flex-row max-w-5xl w-full h-full md:h-auto">
        <div className="flex-1 bg-black flex items-center justify-center">
          <img
            src={api.getImageUrl(photo.filename)}
            alt=""
            className="max-w-full max-h-[50vh] md:max-h-[90vh] object-contain"
          />
        </div>

        <div className="md:w-96 flex flex-col bg-white dark:bg-black text-gray-900 dark:text-white max-h-[50vh] md:max-h-[90vh]">
          <div className="p-4 border-b border-gray-200 dark:border-gray-800 flex items-center gap-3">
            <Link to={`/user/${photo.username}`} className="flex items-center gap-3 hover:opacity-80">
              <div className="w-8 h-8 rounded-full bg-gray-200 dark:bg-gray-800 flex items-center justify-center text-sm font-semibold overflow-hidden">
                {photo.username?.[0]?.toUpperCase() || '?'}
              </div>
              <span className="font-semibold">{photo.username}</span>
            </Link>
          </div>

          <div className="flex-1 overflow-auto p-4 space-y-4">
            {photo.description && (
              <p className="mb-2">
                <Link to={`/user/${photo.username}`} className="font-semibold hover:underline">
                  {photo.username}
                </Link>{' '}
                {photo.description}
              </p>
            )}

            {comments && comments.length > 0 && (
              <div className="space-y-3">
                {comments.map((comment) => (
                  <div key={comment.id} className="flex gap-2">
                    <Link to={`/user/${comment.username}`}>
                      <div className="w-8 h-8 rounded-full bg-gray-200 dark:bg-gray-800 flex items-center justify-center text-sm font-semibold flex-shrink-0">
                        {comment.username?.[0]?.toUpperCase() || '?'}
                      </div>
                    </Link>
                    <div className="flex-1 min-w-0">
                      <p className="text-sm">
                        <Link to={`/user/${comment.username}`} className="font-semibold hover:underline">
                          {comment.username}
                        </Link>{' '}
                        {comment.content}
                      </p>
                      <p className="text-xs text-gray-500 mt-0.5">{comment.created_at}</p>
                    </div>
                    {user && user.id === comment.user_id && (
                      <button
                        onClick={() => deleteCommentMutation.mutate(comment.id)}
                        className="text-gray-400 hover:text-red-500"
                      >
                        <Trash2 className="w-4 h-4" />
                      </button>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="border-t border-gray-200 dark:border-gray-800 p-4">
            <div className="flex items-center gap-4 mb-2">
              <button
                onClick={() => likeMutation.mutate()}
                disabled={likeMutation.isPending}
                className="p-2 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-full disabled:opacity-50 transition-colors"
              >
                <Heart
                  className={`w-6 h-6 ${photo.is_liked ? 'fill-red-500 text-red-500' : ''}`}
                />
              </button>
              <button
                onClick={() => favoriteMutation.mutate()}
                disabled={favoriteMutation.isPending}
                className="p-2 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-full disabled:opacity-50 transition-colors"
              >
                <MessageCircle className="w-6 h-6" />
              </button>

              {user && user.id === photo.user_id && (
                <>
                  <Link
                    to={`/photo/${id}/edit`}
                    className="p-2 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-full transition-colors ml-auto"
                  >
                    <Edit className="w-6 h-6" />
                  </Link>
                  <button
                    onClick={() => {
                      if (confirm('Delete this photo?')) {
                        deleteMutation.mutate()
                      }
                    }}
                    disabled={deleteMutation.isPending}
                    className="p-2 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-full text-red-500 disabled:opacity-50 transition-colors"
                  >
                    <Trash2 className="w-6 h-6" />
                  </button>
                </>
              )}
            </div>

            <p className="font-semibold text-sm mb-2">
              {photo.like_count} {photo.like_count === 1 ? 'like' : 'likes'}
            </p>

            {photo.nui_names.length > 0 && (
              <div className="flex flex-wrap gap-1 mb-2">
                {photo.nui_names.map(name => (
                  <span
                    key={name}
                    className="px-2 py-1 text-xs bg-gray-100 dark:bg-gray-800 rounded"
                  >
                    {name}
                  </span>
                ))}
              </div>
            )}

            {photo.taken_at && (
              <p className="text-xs text-gray-500 mb-2">
                Taken: {photo.taken_at}
              </p>
            )}

            <p className="text-xs text-gray-500 mb-3">
              {photo.created_at}
            </p>
          </div>

          {user && (
            <div className="border-t border-gray-200 dark:border-gray-800 p-4">
              <form
                onSubmit={(e) => {
                  e.preventDefault()
                  if (commentText.trim()) {
                    commentMutation.mutate()
                  }
                }}
                className="flex gap-2"
              >
                <input
                  type="text"
                  value={commentText}
                  onChange={(e) => setCommentText(e.target.value)}
                  placeholder="Add a comment..."
                  className="flex-1 bg-transparent border-none outline-none text-sm"
                  maxLength={1000}
                />
                <button
                  type="submit"
                  disabled={!commentText.trim() || commentMutation.isPending}
                  className="text-blue-500 font-semibold text-sm disabled:opacity-50"
                >
                  Post
                </button>
              </form>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
