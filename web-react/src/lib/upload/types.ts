export type UploadProgress = {
  uploaded: number
  total: number
  percentage: number
}

export type ChunkedSession = {
  id: string
  filename: string
  total_size: number
  chunk_size: number
  total_chunks: number
}

export type ChunkedCompleteResult = {
  filename: string
  size: number
  status: string
  media_id?: number
  temporary_upload_id?: number
}

export type ChunkedStatus = {
  upload_id: string
  filename: string
  uploaded_chunks: number
  total_chunks: number
  missing_chunks: number[]
  expires_at: string
}
