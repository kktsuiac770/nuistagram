import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../lib/api'
import { useAuth } from '../contexts/Auth'
import { X } from 'lucide-react'

export default function Upload() {
  const [files, setFiles] = useState<File[]>([])
  const [previews, setPreviews] = useState<string[]>([])
  const [nuiNames, setNuiNames] = useState('')
  const [description, setDescription] = useState('')
  const [error, setError] = useState('')
  const navigate = useNavigate()
  const { user } = useAuth()
  const queryClient = useQueryClient()

  const { data: nuis } = useQuery({
    queryKey: ['nuis'],
    queryFn: api.getNuis,
  })

  const uploadMutation = useMutation({
    mutationFn: api.upload,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['photos'] })
      navigate('/')
    },
    onError: (err) => setError(err instanceof Error ? err.message : 'Upload failed'),
  })

  if (!user) {
    navigate('/login')
    return null
  }

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const selected = Array.from(e.target.files || [])
    setFiles(selected)
    setPreviews(selected.map(f => URL.createObjectURL(f)))
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (files.length === 0) {
      setError('Please select at least one photo')
      return
    }
    if (!nuiNames.trim()) {
      setError('Please enter at least one NUI name')
      return
    }

    const formData = new FormData()
    files.forEach(f => formData.append('photos', f))
    formData.append('nui_names', nuiNames)
    formData.append('description', description)

    uploadMutation.mutate(formData)
  }

  return (
    <div className="max-w-xl mx-auto py-8 px-4">
      <h1 className="text-2xl font-semibold mb-6 text-center">Create new post</h1>

      {error && (
        <div className="mb-4 p-3 bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400 text-sm text-center rounded">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="border-2 border-dashed border-gray-300 dark:border-gray-700 rounded-lg p-8 text-center">
          {previews.length > 0 ? (
            <div className="grid grid-cols-3 gap-2">
              {previews.map((src, i) => (
                <div key={i} className="relative aspect-square">
                  <img src={src} alt="" className="w-full h-full object-cover rounded" />
                  <button
                    type="button"
                    onClick={() => {
                      setFiles(files.filter((_, j) => j !== i))
                      setPreviews(previews.filter((_, j) => j !== i))
                    }}
                    className="absolute top-1 right-1 p-1 bg-black/50 rounded-full text-white"
                  >
                    <X className="w-4 h-4" />
                  </button>
                </div>
              ))}
            </div>
          ) : (
            <label className="cursor-pointer block">
              <p className="text-gray-500 mb-2">Click to upload photos</p>
              <input
                type="file"
                accept="image/*"
                multiple
                onChange={handleFileChange}
                className="hidden"
              />
            </label>
          )}
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">NUI Names *</label>
          <input
            type="text"
            value={nuiNames}
            onChange={e => setNuiNames(e.target.value)}
            placeholder="e.g., Fluffy, Whiskers"
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 bg-white dark:bg-black rounded focus:outline-none focus:border-gray-400"
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
            className="w-full px-3 py-2 border border-gray-300 dark:border-gray-700 bg-white dark:bg-black rounded focus:outline-none focus:border-gray-400 resize-none"
          />
        </div>

        <button
          type="submit"
          disabled={uploadMutation.isPending}
          className="w-full py-2 font-semibold bg-blue-500 text-white rounded hover:bg-blue-600 disabled:opacity-50 transition-colors"
        >
          {uploadMutation.isPending ? 'Uploading...' : 'Share'}
        </button>
      </form>
    </div>
  )
}
