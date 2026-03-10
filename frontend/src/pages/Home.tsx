import { useQuery } from '@tanstack/react-query'
import { useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../lib/api'
import { useAuth } from '../contexts/Auth'
import { Heart, X, Search, MessageCircle } from 'lucide-react'

export default function Home() {
  const [page, setPage] = useState(1)
  const [selectedTags, setSelectedTags] = useState<string[]>([])
  const [searchInput, setSearchInput] = useState('')
  const [feedType, setFeedType] = useState<'all' | 'following'>('all')
  const { user } = useAuth()

  const { data, isLoading, error } = useQuery({
    queryKey: ['photos', page, selectedTags, feedType],
    queryFn: () => api.getPhotos(page, selectedTags.length > 0 ? selectedTags : undefined, user ? feedType : 'all'),
    staleTime: 60_000,
  })

  const { data: nuis } = useQuery({
    queryKey: ['nuis'],
    queryFn: api.getNuis,
    staleTime: 5 * 60_000,
  })

  const addTag = (tag: string) => {
    const trimmed = tag.trim()
    if (trimmed && !selectedTags.includes(trimmed)) {
      setSelectedTags([...selectedTags, trimmed])
      setPage(1)
    }
    setSearchInput('')
  }

  const removeTag = (tag: string) => {
    setSelectedTags(selectedTags.filter(t => t !== tag))
    setPage(1)
  }

  const handleSearchKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      addTag(searchInput)
    }
  }

  if (error) {
    return (
      <div className="flex flex-col items-center justify-center py-20">
        <p className="text-gray-500 dark:text-gray-400 mb-4">Failed to load photos</p>
        <button
          onClick={() => window.location.reload()}
          className="px-4 py-2 bg-blue-500 text-white rounded-lg"
        >
          Retry
        </button>
      </div>
    )
  }

  return (
    <div className="py-4">
      <div className="px-4 mb-4">
        {user && (
          <div className="flex gap-2 mb-4">
            <button
              onClick={() => { setFeedType('all'); setPage(1) }}
              className={`px-4 py-2 rounded-full text-sm font-medium transition-colors ${feedType === 'all'
                ? 'bg-gray-900 dark:bg-white text-white dark:text-black'
                : 'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300'
                }`}
            >
              All
            </button>
            <button
              onClick={() => { setFeedType('following'); setPage(1) }}
              className={`px-4 py-2 rounded-full text-sm font-medium transition-colors ${feedType === 'following'
                ? 'bg-gray-900 dark:bg-white text-white dark:text-black'
                : 'bg-gray-100 dark:bg-gray-800 text-gray-700 dark:text-gray-300'
                }`}
            >
              Following
            </button>
          </div>
        )}

        <div className="flex flex-wrap gap-2 items-center">
          <div className="relative flex-1 min-w-[200px]">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
            <input
              type="text"
              value={searchInput}
              onChange={e => setSearchInput(e.target.value)}
              onKeyDown={handleSearchKeyDown}
              placeholder="Search by NUI name..."
              className="w-full pl-10 pr-4 py-2 text-sm border border-gray-300 dark:border-gray-700 bg-gray-50 dark:bg-gray-900 rounded-full focus:outline-none focus:border-gray-400"
            />
          </div>
        </div>

        {selectedTags.length > 0 && (
          <div className="flex flex-wrap gap-2 mt-2">
            {selectedTags.map(tag => (
              <span
                key={tag}
                className="inline-flex items-center gap-1 px-3 py-1 bg-blue-100 dark:bg-blue-900 text-blue-800 dark:text-blue-200 rounded-full text-sm"
              >
                {tag}
                <button onClick={() => removeTag(tag)} className="hover:text-blue-600">
                  <X className="w-3 h-3" />
                </button>
              </span>
            ))}
            <button
              onClick={() => setSelectedTags([])}
              className="text-sm text-gray-500 hover:text-gray-700 dark:hover:text-gray-300"
            >
              Clear all
            </button>
          </div>
        )}

        {nuis && nuis.length > 0 && selectedTags.length === 0 && (
          <div className="flex flex-wrap gap-1 mt-2">
            {nuis.slice(0, 10).map(nui => (
              <button
                key={nui.id}
                onClick={() => addTag(nui.name)}
                className="px-2 py-1 text-xs bg-gray-100 dark:bg-gray-800 rounded hover:bg-gray-200 dark:hover:bg-gray-700"
              >
                {nui.name}
              </button>
            ))}
          </div>
        )}
      </div>

      {isLoading || !data ? (
        <div className="flex items-center justify-center py-20">
          <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900 dark:border-white"></div>
        </div>
      ) : !data.photos || !data.photos.length ? (
        <div className="flex flex-col items-center justify-center py-20">
          <p className="text-gray-500 dark:text-gray-400 mb-4">
            {feedType === 'following' && user ? 'Start following people to see their photos here' :
              selectedTags.length > 0 ? 'No photos found' : 'No photos yet'}
          </p>
          {user && selectedTags.length === 0 && feedType === 'all' && (
            <Link to="/upload" className="px-4 py-2 bg-blue-500 text-white rounded-lg">
              Upload your first photo
            </Link>
          )}
        </div>
      ) : (
        <>
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

          {data.total_pages > 1 && (
            <div className="flex justify-center gap-2 mt-8">
              <button
                onClick={() => setPage(p => Math.max(1, p - 1))}
                disabled={!data.has_prev}
                className="px-4 py-2 rounded-lg border border-gray-300 dark:border-gray-700 disabled:opacity-50"
              >
                Previous
              </button>
              <span className="px-4 py-2">
                {data.current_page} / {data.total_pages}
              </span>
              <button
                onClick={() => setPage(p => p + 1)}
                disabled={!data.has_next}
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
