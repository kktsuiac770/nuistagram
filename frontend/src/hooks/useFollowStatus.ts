import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../lib/api'

export function useFollowStatusAndMutations(username: string | undefined) {
  const queryClient = useQueryClient()

  const followStatusQuery = useQuery({
    queryKey: ['followStatus', username],
    queryFn: () => api.getFollowStatus(username!),
    enabled: !!username,
  })

  const followMutation = useMutation({
    mutationFn: async (isFollow: boolean) => {
      if (isFollow) {
        return api.followUser(username!)
      } else {
        return api.unfollowUser(username!)
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['followStatus', username] })
      queryClient.invalidateQueries({ queryKey: ['notifications'] })
    },
  })

  const isFollowing = followStatusQuery.data?.is_following ?? false
  const isPending = followMutation.isPending

  const handleFollowClick = () => {
    followMutation.mutate(!isFollowing)
  }

  return {
    isFollowing,
    isPending,
    handleFollowClick,
    followStatusQuery,
    followMutation,
  }
}
