import { useRef, useState } from 'react'
import { ChunkedUploader } from '@/lib/upload/chunked-upload'
import { resolveMediaUrl } from '@/lib/mediaUrl'

type ImageUploadFieldProps = {
  label: string
  imageUrl?: string | null
  accept?: string
  disabled?: boolean
  onUpload: (result: { temporary_upload_id: number; media_id: number }) => Promise<void>
}

export function ImageUploadField({
  label,
  imageUrl,
  accept = 'image/*',
  disabled,
  onUpload,
}: ImageUploadFieldProps) {
  const inputRef = useRef<HTMLInputElement>(null)
  const [progress, setProgress] = useState<number | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [busy, setBusy] = useState(false)

  const resolvedUrl = resolveMediaUrl(imageUrl)

  const handleFile = async (file: File) => {
    setError(null)
    setBusy(true)
    setProgress(0)
    try {
      const uploader = new ChunkedUploader(file, {
        onProgress: (p) => setProgress(p.percentage),
      })
      const result = await uploader.upload()
      if (result.media_id == null || result.temporary_upload_id == null) {
        throw new Error('Upload completed but media IDs were not returned')
      }
      await onUpload({
        temporary_upload_id: result.temporary_upload_id,
        media_id: result.media_id,
      })
      setProgress(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Upload failed')
      setProgress(null)
    } finally {
      setBusy(false)
      if (inputRef.current) inputRef.current.value = ''
    }
  }

  return (
    <div className="space-y-2">
      <p className="text-sm font-medium">{label}</p>
      <div className="flex flex-wrap items-center gap-4">
        <div className="flex h-20 w-20 items-center justify-center overflow-hidden rounded-xl border border-black/10 bg-black/[0.03]">
          {resolvedUrl ? (
            <img src={resolvedUrl} alt="" className="h-full w-full object-cover" />
          ) : (
            <span className="text-xs text-black/40">No image</span>
          )}
        </div>
        <div className="space-y-1">
          <input
            ref={inputRef}
            type="file"
            accept={accept}
            className="text-sm"
            disabled={disabled || busy}
            onChange={(e) => {
              const file = e.target.files?.[0]
              if (file) void handleFile(file)
            }}
          />
          {progress != null && (
            <p className="text-xs text-black/55">Uploading… {Math.round(progress)}%</p>
          )}
          {error && <p className="text-xs text-red-700">{error}</p>}
        </div>
      </div>
    </div>
  )
}
