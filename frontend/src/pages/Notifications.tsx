import { useEffect } from 'react'
import { Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api, getAvatarUrl } from '../lib/api'
import { useAuth } from '../contexts/Auth'
import { Heart, UserPlus, MessageCircle, Bell } from 'lucide-react'

export default function Notifications() {
  const { user } = useAuth()
  const queryClient = useQueryClient()

  const { data: notifications, isLoading } = useQuery({
    queryKey: ['notifications'],
    queryFn: () => api.getNotifications(50),
    enabled: !!user,
  })

  const markAllReadMutation = useMutation({
    mutationFn: () => api.markAllNotificationsRead(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
      queryClient.invalidateQueries({ queryKey: ['unreadCount'] })
    },
  })

  useEffect(() => {
    if (notifications && notifications.some(n => !n.read)) {
      markAllReadMutation.mutate()
    }
  }, [notifications, markAllReadMutation])

  if (!user) {
    return (
      <div className="flex flex-col items-center justify-center min-h-[60vh]">
        <p className="text-gray-500 dark:text-gray-400 mb-4">Log in to see notifications</p>
        <Link to="/login" className="px-4 py-2 bg-blue-500 text-white rounded-lg">
          Log in
        </Link>
      </div>
    )
  }

  const getNotificationIcon = (type: string) => {
    switch (type) {
      case 'follow':
        return <UserPlus className="w-5 h-5" />
      case 'like':
        return <Heart className="w-5 h-5 fill-red-500 text-red-500" />
      case 'comment':
        return <MessageCircle className="w-5 h-5" />
      default:
        return <Bell className="w-5 h-5" />
    }
  }

  const getNotificationText = (notif: { type: string; actor: { username: string } }) => {
    const actor = notif.actor.username
    switch (notif.type) {
      case 'follow':
        return <><strong>{actor}</strong> started following you</>
      case 'like':
        return <><strong>{actor}</strong> liked your photo</>
      case 'comment':
        return <><strong>{actor}</strong> commented on your photo</>
      default:
        return <>Notification from <strong>{actor}</strong></>
    }
  }

  return (
    <div className="py-4 px-4">
      <div className="max-w-lg mx-auto">
        <h1 className="text-xl font-semibold mb-6">Notifications</h1>

        {isLoading ? (
          <div className="flex items-center justify-center py-8">
            <div className="animate-spin rounded-full h-6 w-6 border-b-2 border-gray-900 dark:border-white"></div>
          </div>
        ) : !notifications || notifications.length === 0 ? (
          <div className="text-center py-8">
            <Bell className="w-12 h-12 mx-auto text-gray-300 dark:text-gray-700 mb-4" />
            <p className="text-gray-500 dark:text-gray-400">No notifications yet</p>
          </div>
        ) : (
          <div className="space-y-1">
            {notifications.map((notif) => (
              <div
                key={notif.id}
                className={`flex items-center gap-3 p-3 rounded-lg ${
                  notif.read ? '' : 'bg-blue-50 dark:bg-blue-900/20'
                }`}
              >
                {notif.actor.avatar ? (
                  <img
                    src={getAvatarUrl(notif.actor.avatar)}
                    alt={notif.actor.username}
                    className="w-10 h-10 rounded-full object-cover"
                  />
                ) : (
                  <Link
                    to={`/user/${notif.actor.username}`}
                    className="w-10 h-10 rounded-full bg-gray-200 dark:bg-gray-800 flex items-center justify-center font-semibold flex-shrink-0"
                  >
                    {notif.actor.username[0].toUpperCase()}
                  </Link>
                )}

                <div className="flex-1 min-w-0">
                  <p className="text-sm truncate">
                    {getNotificationText(notif)}
                  </p>
                  <p className="text-xs text-gray-500">{notif.created_at}</p>
                </div>

                <div className="text-gray-400">
                  {getNotificationIcon(notif.type)}
                </div>

                {notif.photo_id && (
                  <Link
                    to={`/photo/${notif.photo_id}`}
                    className="w-10 h-10 bg-gray-100 dark:bg-gray-800 rounded flex-shrink-0"
                  />
                )}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
