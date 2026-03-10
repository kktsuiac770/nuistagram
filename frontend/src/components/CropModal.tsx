import { useState, useCallback } from 'react'
import Cropper, { type Area } from 'react-easy-crop'
import { X } from 'lucide-react'

interface Props {
  imageSrc: string
  filename: string
  onCrop: (blob: Blob) => void
  onCancel: () => void
}

async function cropToBlob(imageSrc: string, pixelCrop: Area): Promise<Blob> {
  const image = new Image()
  image.src = imageSrc
  await new Promise<void>((resolve, reject) => {
    image.onload = () => resolve()
    image.onerror = () => reject(new Error('Failed to load image'))
  })
  const canvas = document.createElement('canvas')
  canvas.width = pixelCrop.width
  canvas.height = pixelCrop.height
  const ctx = canvas.getContext('2d')!
  ctx.drawImage(
    image,
    pixelCrop.x,
    pixelCrop.y,
    pixelCrop.width,
    pixelCrop.height,
    0,
    0,
    pixelCrop.width,
    pixelCrop.height,
  )
  return new Promise<Blob>((resolve, reject) => {
    canvas.toBlob(
      blob => (blob ? resolve(blob) : reject(new Error('Canvas is empty'))),
      'image/jpeg',
      0.95,
    )
  })
}

export default function CropModal({ imageSrc, filename, onCrop, onCancel }: Props) {
  const [crop, setCrop] = useState({ x: 0, y: 0 })
  const [zoom, setZoom] = useState(1)
  const [croppedAreaPixels, setCroppedAreaPixels] = useState<Area | null>(null)

  const onCropComplete = useCallback((_: Area, cap: Area) => {
    setCroppedAreaPixels(cap)
  }, [])

  const handleApply = async () => {
    if (!croppedAreaPixels) return
    const blob = await cropToBlob(imageSrc, croppedAreaPixels)
    onCrop(blob)
  }

  return (
    <div className="fixed inset-0 z-50 flex flex-col bg-black">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 shrink-0">
        <button
          type="button"
          onClick={onCancel}
          className="p-1 text-white/70 hover:text-white"
        >
          <X className="w-6 h-6" />
        </button>
        <p className="text-white text-sm font-medium truncate max-w-[60%]">{filename}</p>
        <button
          type="button"
          onClick={handleApply}
          className="text-blue-400 font-semibold text-sm hover:text-blue-300"
        >
          Apply
        </button>
      </div>

      {/* Crop area */}
      <div className="relative flex-1">
        <Cropper
          image={imageSrc}
          crop={crop}
          zoom={zoom}
          aspect={1}
          onCropChange={setCrop}
          onZoomChange={setZoom}
          onCropComplete={onCropComplete}
        />
      </div>

      {/* Zoom slider */}
      <div className="px-6 py-4 shrink-0 flex items-center gap-3">
        <span className="text-white/50 text-xs">−</span>
        <input
          type="range"
          min={1}
          max={3}
          step={0.01}
          value={zoom}
          onChange={e => setZoom(Number(e.target.value))}
          className="flex-1 accent-white"
        />
        <span className="text-white/50 text-xs">+</span>
      </div>
    </div>
  )
}
