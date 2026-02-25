import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from '../lib/api'
import { useAuth } from '../contexts/Auth'
import { useToast } from '../components/Toast'
import { X, Upload as UploadIcon, ImagePlus } from 'lucide-react'

export default function Upload() {
  const [files, setFiles] = useState<File[]>([])
  const [previews, setPreviews] = useState<string[]>([])
  const [nuiNames, setNuiNames] = useState('')
  const [description, setDescription] = useState('')
  const navigate = useNavigate()
  const { user } = useAuth()
  const toast = useToast()
  const queryClient = useQueryClient()

  const { data: nuis } = useQuery({
    queryKey: ['nuis'],
    queryFn: api.getNuis,
  })

  const uploadMutation = useMutation({
    mutationFn: api.upload,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['photos'] })
      toast.success('Photo uploaded successfully!')
      navigate('/')
    },
    onError: (err) => {
      toast.error(err instanceof Error ? err.message : 'Upload failed')
    },
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

  const removeFile = (index: number) => {
    URL.revokeObjectURL(previews[index])
    setFiles(files.filter((_, i) => i !== index))
    setPreviews(previews.filter((_, i) => i !== index))
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (files.length === 0) {
      toast.error('Please select at least one photo')
      return
    }
    if (!nuiNames.trim()) {
      toast.error('Please enter at least one NUI name')
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

      <form onSubmit={handleSubmit} className="space-y-4">
        <div className="border-2 border-dashed border-gray-300 dark:border-gray-700 rounded-lg p-8 text-center min-h-[200px] flex flex-col items-center justify-center">
          {previews.length > 0 ? (
            <div className="w-full">
              <div className="grid grid-cols-3 gap-2 mb-4">
                {previews.map((src, i) => (
                  <div key={i} className="relative aspect-square">
                    <img src={src} alt="" className="w-full h-full object-cover rounded" />
                    <button
                      type="button"
                      onClick={() => removeFile(i)}
                      className="absolute top-1 right-1 p-1 bg-black/50 rounded-full text-white hover:bg-black/70"
                    >
                      <X className="w-4 h-4" />
                    </button>
                  </div>
                ))}
              </div>
              <label className="cursor-pointer inline-flex items-center gap-2 px-4 py-2 text-sm border border-gray-300 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-900">
                <ImagePlus className="w-4 h-4" />
                Add more
                <input
                  type="file"
                  accept="image/*"
                  multiple
                  onChange={handleFileChange}
                  className="hidden"
                />
              </label>
            </div>
          ) : (
            <label className="cursor-pointer flex flex-col items-center">
              <ImagePlus className="w-12 h-12 text-gray-400 mb-4" />
              <p className="text-gray-500 dark:text-gray-400 mb-2">Drag photos here or click to upload</p>
              <p className="text-xs text-gray-400">Support: JPG, PNG, GIF</p>
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

        <button
          type="submit"
          disabled={uploadMutation.isPending || files.length === 0}
          className="w-full py-3 font-semibold bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed transition-colors flex items-center justify-center gap-2"
        >
          {uploadMutation.isPending ? (
            <>
              <div className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
              Uploading...
            </>
          ) : (
            <>
              <UploadIcon className="w-5 h-5" />
              Share
            </>
          )}
        </button>
      </form>
    </div>
  )
}
