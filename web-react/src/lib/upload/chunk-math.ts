export function totalChunks(totalSize: number, chunkSize: number): number {
  if (totalSize <= 0 || chunkSize <= 0) {
    throw new Error(`invalid size parameters: total_size=${totalSize} chunk_size=${chunkSize}`)
  }
  return Math.ceil(totalSize / chunkSize)
}
