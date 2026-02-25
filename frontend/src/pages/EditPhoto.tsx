import { useState, useEffect } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../lib/api'
import { useToast } from '../components/Toast'
import { X } from 'lucide-react'

export default function EditPhoto() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const toast = useToast()
  const queryClient = useQueryClient()

  const { data: photo, isLoading } = useQuery({
    queryKey: ['photo', id],
    queryFn: () => api.getPhoto(Number(id)),
    enabled: !!id,
  })

  const [description, setDescription] = useState('')
  const [nuiNames, setNuiNames] = useState('')
  const [initialized, setInitialized] = useState(false)

  useEffect(() => {
    if (photo && !initialized) {
      setDescription(photo.description || '')
      setNuiNames(photo.nui_names.join(', '))
      setInitialized(true)
    }
  }, [photo, initialized])

  const { data: nuis } = useQuery({
    queryKey: ['nuis'],
    queryFn: api.getNuis,
  })

  const updateMutation = useMutation({
    mutationFn: () => {
      const formData = new FormData()
      formData.append('description', description)
      formData.append('nui_names', nuiNames)
      return api.updatePhoto(Number(id), formData)
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['photo', id] })
      queryClient.invalidateQueries({ queryKey: ['photos'] })
      toast.success('Photo updated')
      navigate(`/photo/${id}`, { replace: true })
    },
    onError: () => toast.error('Failed to update photo'),
  })

  if (isLoading || !photo) {
    return (
      <div className="fixed inset-0 z-50 bg-black flex items-center justify-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-white"></div>
      </div>
    )
  }

  return (
    <div className="fixed inset-0 z-50 bg-black/90 flex items-center justify-center">
      <div className="max-w-lg w-full mx-4 bg-white dark:bg-black rounded-lg overflow-hidden">
        <div className="flex items-center justify-between p-4 border-b border-gray-200 dark:border-gray-800">
          <h2 className="font-semibold">Edit photo</h2>
          <Link to={`/photo/${id}`} className="p-2 hover:bg-gray-100 dark:hover:bg-gray-800 rounded-full">
            <X className="w-5 h-5" />
          </Link>
        </div>

        <div className="p-4">
          <div className="flex gap-4 mb-6">
            <div className="w-24 h-24 flex-shrink-0">
              <img
                src={api.getThumbnailUrl(photo.thumbnail) || api.getImageUrl(photo.filename)}
                alt=""
                className="w-full h-full object-cover rounded"
              />
            </div>
            <div className="flex-1">
              <div className="flex flex-wrap gap-1">
                {photo.nui_names.map(name => (
                  <span
                    key={name}
                    className="px-2 py-1 text-xs bg-gray-100 dark:bg-gray-800 rounded"
                  >
                    {name}
                  </span>
                ))}
              </div>
            </div>
          </div>

          <div className="space-y-4">
            <div>
              <label className="block text-sm font-medium mb-1">NUI Names *</label>
              <input
                type="text"
                value={nuiNames}
                onChange={e => setNuiNames(e.target.value)}
                placeholder="e.g., Fluffy, Whiskers"
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 bg-white dark:bg-black rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
              {nuis && nuis.length > 0 && (
                <div className="mt-2 flex flex-wrap gap-1">
                  {nuis.slice(0, 10).map(nui => (
                    <button
                      key={nui.id}
                      type="button"
                      onClick={() => setNuiNames(prev => prev ? `${prev}, ${nui.name}` : nui.name)}
                      className="px-2 py-1 text-xs bg-gray-100 dark:bg-gray-800 rounded hover:bg-gray-200 dark:hover:bg-gray-700"
                    >
                      {nui.name}
                    </button>
                  ))}
                </div>
              )}
            </div>

            <div>
              <label className="block text-sm font-medium mb-1">Description</label>
              <textarea
                value={description}
                onChange={e => setDescription(e.target.value)}
                rows={3}
                placeholder="Write a caption..."
                className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 bg-white dark:bg-black rounded-lg focus:outline-none focus:ring-2 focus:ring-blue-500 resize-none"
              />
            </div>
          </div>
        </div>

        <div className="p-4 border-t border-gray-200 dark:border-gray-800 flex justify-end gap-2">
          <Link
            to={`/photo/${id}`}
            className="px-4 py-2 border border-gray-300 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-900"
          >
            Cancel
          </Link>
          <button
            onClick={() => updateMutation.mutate()}
            disabled={updateMutation.isPending || !nuiNames.trim()}
            className="px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:opacity-50"
          >
            {updateMutation.isPending ? 'Saving...' : 'Save'}
          </button>
        </div>
      </div>
    </div>
  )
}
