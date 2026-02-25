import { useParams, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { api, getAvatarUrl } from '../lib/api'
import { useAuth } from '../contexts/Auth'
import { Heart, MessageCircle, Settings, UserPlus, UserCheck } from 'lucide-react'

export default function Profile() {
  const { username } = useParams<{ username: string }>()
  const { user: currentUser } = useAuth()
  const queryClient = useQueryClient()
  const [page, setPage] = useState(1)

  const { data: profile, isLoading: profileLoading, error } = useQuery({
    queryKey: ['user', username],
    queryFn: () => api.getUser(username!),
    enabled: !!username,
  })

  const { data: photos, isLoading: photosLoading } = useQuery({
    queryKey: ['userPhotos', username, page],
    queryFn: () => api.getUserPhotos(username!, page),
    enabled: !!username,
  })

  const followMutation = useMutation({
    mutationFn: () => api.followUser(username!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user', username] })
      queryClient.invalidateQueries({ queryKey: ['user', currentUser?.username] })
    },
  })

  const unfollowMutation = useMutation({
    mutationFn: () => api.unfollowUser(username!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['user', username] })
      queryClient.invalidateQueries({ queryKey: ['user', currentUser?.username] })
    },
  })

  if (profileLoading || photosLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900 dark:border-white"></div>
      </div>
    )
  }

  if (error || !profile) {
    return (
      <div className="flex flex-col items-center justify-center py-20">
        <p className="text-gray-500 dark:text-gray-400 mb-4">User not found</p>
        <Link to="/" className="text-blue-500">Go home</Link>
      </div>
    )
  }

  const isOwnProfile = currentUser?.id === profile.id

  const handleFollowClick = () => {
    if (profile.is_following) {
      unfollowMutation.mutate()
    } else {
      followMutation.mutate()
    }
  }

  return (
    <div className="py-4">
      <div className="px-4 py-8 border-b border-gray-200 dark:border-gray-800">
        <div className="flex items-start gap-8 max-w-4xl mx-auto">
          <div className="flex-shrink-0">
            {profile.avatar ? (
              <img
                src={getAvatarUrl(profile.avatar)}
                alt={profile.username}
                className="w-20 h-20 sm:w-36 sm:h-36 rounded-full object-cover"
              />
            ) : (
              <div className="w-20 h-20 sm:w-36 sm:h-36 rounded-full bg-gray-200 dark:bg-gray-800 flex items-center justify-center text-4xl sm:text-5xl font-semibold">
                {profile.username[0].toUpperCase()}
              </div>
            )}
          </div>

          <div className="flex-1 min-w-0">
            <div className="flex flex-wrap items-center gap-2 mb-4">
              <h1 className="text-xl font-semibold">{profile.username}</h1>
              
              {isOwnProfile ? (
                <>
                  <Link
                    to="/settings"
                    className="px-4 py-1.5 bg-gray-100 dark:bg-gray-800 rounded-lg text-sm font-semibold hover:bg-gray-200 dark:hover:bg-gray-700"
                  >
                    <Settings className="w-4 h-4 inline mr-1" />
                    Edit profile
                  </Link>
                </>
              ) : currentUser && (
                <button
                  onClick={handleFollowClick}
                  disabled={followMutation.isPending || unfollowMutation.isPending}
                  className={`px-4 py-1.5 rounded-lg text-sm font-semibold ${
                    profile.is_following
                      ? 'bg-gray-100 dark:bg-gray-800 hover:bg-gray-200 dark:hover:bg-gray-700'
                      : 'bg-blue-500 text-white hover:bg-blue-600'
                  } disabled:opacity-50`}
                >
                  {profile.is_following ? (
                    <>
                      <UserCheck className="w-4 h-4 inline mr-1" />
                      Following
                    </>
                  ) : (
                    <>
                      <UserPlus className="w-4 h-4 inline mr-1" />
                      Follow
                    </>
                  )}
                </button>
              )}
            </div>

            <div className="flex gap-6 mb-4">
              <span>
                <strong>{profile.photo_count}</strong>{' '}
                {profile.photo_count === 1 ? 'post' : 'posts'}
              </span>
              <span>
                <strong>{profile.follower_count}</strong>{' '}
                {profile.follower_count === 1 ? 'follower' : 'followers'}
              </span>
              <span>
                <strong>{profile.following_count}</strong> following
              </span>
            </div>

            {profile.bio && (
              <p className="text-sm whitespace-pre-wrap">{profile.bio}</p>
            )}
          </div>
        </div>
      </div>

      {!photos?.photos || photos.photos.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20">
          <p className="text-gray-500 dark:text-gray-400">
            {isOwnProfile ? 'No photos yet' : 'No photos'}
          </p>
          {isOwnProfile && (
            <Link
              to="/upload"
              className="mt-4 px-4 py-2 bg-blue-500 text-white rounded-lg text-sm font-semibold"
            >
              Upload your first photo
            </Link>
          )}
        </div>
      ) : (
        <>
          <div className="grid grid-cols-3 gap-1 sm:gap-4 py-4">
            {photos.photos.map((photo) => (
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
                    <span className="flex items-center gap-1">
                      <Heart className={`w-6 h-6 ${photo.is_liked ? 'fill-red-500 text-red-500' : ''}`} />
                      {photo.like_count > 0 && <span>{photo.like_count}</span>}
                    </span>
                    <span className="flex items-center gap-1">
                      <MessageCircle className="w-6 h-6" />
                      {photo.comment_count > 0 && <span>{photo.comment_count}</span>}
                    </span>
                  </div>
                </div>
              </Link>
            ))}
          </div>

          {photos.total_pages > 1 && (
            <div className="flex justify-center gap-2 py-4">
              <button
                onClick={() => setPage(p => Math.max(1, p - 1))}
                disabled={!photos.has_prev}
                className="px-4 py-2 rounded-lg border border-gray-300 dark:border-gray-700 disabled:opacity-50"
              >
                Previous
              </button>
              <span className="px-4 py-2">
                {photos.current_page} / {photos.total_pages}
              </span>
              <button
                onClick={() => setPage(p => p + 1)}
                disabled={!photos.has_next}
                className="px-4 py-2 rounded-lg border border-gray-300 dark:border-gray-700 disabled:opacity-50"
              >
                Next
              </button>
            </div>
          )}
        </>
      )}
    </div>
  )
}
