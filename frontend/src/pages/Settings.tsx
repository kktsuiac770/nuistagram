import { useState } from 'react'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from 'react-router-dom'
import { api, getAvatarUrl } from '../lib/api'
import { useAuth } from '../contexts/Auth'
import { useToast } from '../components/Toast'
import { Camera, Loader2 } from 'lucide-react'

export default function Settings() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const toast = useToast()
  const queryClient = useQueryClient()
  const [bio, setBio] = useState('')
  const [isUploading, setIsUploading] = useState(false)

  const { data: profile, isLoading } = useQuery({
    queryKey: ['me'],
    queryFn: api.getMe,
    enabled: !!user,
  })

  useState(() => {
    if (profile?.bio) {
      setBio(profile.bio)
    }
  })

  const updateProfileMutation = useMutation({
    mutationFn: () => api.updateProfile(bio),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['me'] })
      queryClient.invalidateQueries({ queryKey: ['user', user?.username] })
      toast.success('Profile updated')
    },
    onError: () => toast.error('Failed to update profile'),
  })

  const handleAvatarUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    setIsUploading(true)
    try {
      const formData = new FormData()
      formData.append('avatar', file)
      await api.uploadAvatar(formData)
      queryClient.invalidateQueries({ queryKey: ['me'] })
      queryClient.invalidateQueries({ queryKey: ['user', user?.username] })
      toast.success('Avatar updated')
    } catch {
      toast.error('Failed to upload avatar')
    } finally {
      setIsUploading(false)
    }
  }

  if (!user) {
    navigate('/login')
    return null
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center min-h-[60vh]">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900 dark:border-white"></div>
      </div>
    )
  }

  return (
    <div className="py-8 px-4">
      <div className="max-w-lg mx-auto">
        <h1 className="text-xl font-semibold mb-6">Edit Profile</h1>

        <div className="flex flex-col items-center mb-8">
          <div className="relative">
            {profile?.avatar ? (
              <img
                src={getAvatarUrl(profile.avatar)}
                alt={user.username}
                className="w-24 h-24 rounded-full object-cover"
              />
            ) : (
              <div className="w-24 h-24 rounded-full bg-gray-200 dark:bg-gray-800 flex items-center justify-center text-3xl font-semibold">
                {user.username[0].toUpperCase()}
              </div>
            )}
            <label className="absolute bottom-0 right-0 p-2 bg-blue-500 rounded-full cursor-pointer hover:bg-blue-600 transition-colors">
              <Camera className="w-4 h-4 text-white" />
              <input
                type="file"
                accept="image/*"
                onChange={handleAvatarUpload}
                className="hidden"
                disabled={isUploading}
              />
            </label>
            {isUploading && (
              <div className="absolute inset-0 bg-black/50 rounded-full flex items-center justify-center">
                <Loader2 className="w-6 h-6 text-white animate-spin" />
              </div>
            )}
          </div>
          <p className="text-sm text-gray-500 mt-2">{user.username}</p>
        </div>

        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">Bio</label>
            <textarea
              value={bio}
              onChange={(e) => setBio(e.target.value)}
              placeholder="Write a bio..."
              rows={4}
              maxLength={500}
              className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-900 rounded-lg focus:outline-none focus:border-gray-400 resize-none"
            />
            <p className="text-xs text-gray-500 mt-1">{bio.length}/500</p>
          </div>

          <button
            onClick={() => updateProfileMutation.mutate()}
            disabled={updateProfileMutation.isPending}
            className="w-full py-2 bg-blue-500 text-white rounded-lg font-semibold hover:bg-blue-600 disabled:opacity-50 transition-colors"
          >
            {updateProfileMutation.isPending ? 'Saving...' : 'Save Changes'}
          </button>
        </div>
      </div>
    </div>
  )
}
