import { useRef, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useMutation, useQuery } from '@tanstack/react-query'
import { apiFetch } from '@/lib/api'
import { ChunkedUploader } from '@/lib/upload/chunked-upload'
import type { Candidate, PublicJobOpening } from '@/types/api'

const inputClass =
  'w-full rounded-lg border border-black/15 bg-white px-3 py-2 text-sm outline-none focus:border-[var(--empops-accent)]'

const applySchema = z.object({
  name: z.string().min(1, 'Required'),
  email: z.string().email(),
  url: z.string().optional(),
  desired_salary: z.string().optional(),
  notes: z.string().optional(),
})
type ApplyValues = z.infer<typeof applySchema>

function PublicShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen">
      <header className="border-b border-black/10 bg-white/70 backdrop-blur">
        <div className="mx-auto max-w-5xl px-6 py-4">
          <Link
            to="/jobs"
            className="text-sm font-semibold uppercase tracking-[0.2em] text-[var(--empops-accent)]"
          >
            EmpOps Careers
          </Link>
        </div>
      </header>
      <main className="mx-auto max-w-5xl px-6 py-8">{children}</main>
    </div>
  )
}

export default function JobApplyPage() {
  const { companySlug, jobSlug } = useParams<{ companySlug: string; jobSlug: string }>()
  const [step, setStep] = useState<1 | 2 | 3>(1)
  const [candidateUuid, setCandidateUuid] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)
  const [uploadProgress, setUploadProgress] = useState<number | null>(null)
  const [uploadBusy, setUploadBusy] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  const jobBase = `/jobs/${companySlug}/jobs/${jobSlug}`

  const jobQuery = useQuery({
    queryKey: ['public-job', companySlug, jobSlug],
    queryFn: async () => {
      const res = await apiFetch<PublicJobOpening>(jobBase)
      return res.data
    },
    enabled: Boolean(companySlug && jobSlug),
  })

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<ApplyValues>({
    resolver: zodResolver(applySchema),
    defaultValues: { name: '', email: '', url: '', desired_salary: '', notes: '' },
  })

  const createCandidate = useMutation({
    mutationFn: async (values: ApplyValues) => {
      const res = await apiFetch<Candidate>(jobBase, {
        method: 'POST',
        body: JSON.stringify({
          name: values.name,
          email: values.email,
          url: values.url || null,
          desired_salary: values.desired_salary || null,
          notes: values.notes || null,
        }),
      })
      return res.data
    },
    onSuccess: (data) => {
      setError(null)
      setCandidateUuid(data.uuid)
      setStep(2)
    },
    onError: (e: Error) => setError(e.message),
  })

  const attachFile = useMutation({
    mutationFn: async (payload: { temporary_upload_id: number; media_id: number }) => {
      await apiFetch(`${jobBase}/apply/${candidateUuid}/files`, {
        method: 'POST',
        body: JSON.stringify(payload),
      })
    },
  })

  const completeApplication = useMutation({
    mutationFn: async () => {
      await apiFetch(`${jobBase}/apply/${candidateUuid}/complete`, { method: 'POST' })
    },
    onSuccess: () => {
      setError(null)
      setSuccess(true)
      setStep(3)
    },
    onError: (e: Error) => setError(e.message),
  })

  const handleFileUpload = async (file: File) => {
    if (!candidateUuid) return
    setError(null)
    setUploadBusy(true)
    setUploadProgress(0)
    try {
      const uploader = new ChunkedUploader(file, {
        onProgress: (p) => setUploadProgress(p.percentage),
      })
      const result = await uploader.upload()
      if (result.media_id == null || result.temporary_upload_id == null) {
        throw new Error('Upload completed but media IDs were not returned')
      }
      await attachFile.mutateAsync({
        temporary_upload_id: result.temporary_upload_id,
        media_id: result.media_id,
      })
      setUploadProgress(null)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Upload failed')
      setUploadProgress(null)
    } finally {
      setUploadBusy(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  if (jobQuery.isLoading) {
    return (
      <PublicShell>
        <p className="text-black/60">Loading job…</p>
      </PublicShell>
    )
  }

  if (jobQuery.isError || !jobQuery.data) {
    return (
      <PublicShell>
        <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">
          {(jobQuery.error as Error)?.message ?? 'Job not found'}
        </div>
      </PublicShell>
    )
  }

  const job = jobQuery.data

  return (
    <PublicShell>
      <div className="space-y-6">
        <div>
          {companySlug && (
            <Link
              to={`/jobs/${companySlug}`}
              className="text-sm text-black/50 hover:text-black"
            >
              ← {job.company?.name ?? 'Back to company'}
            </Link>
          )}
          <h1 className="mt-2 text-2xl font-semibold">{job.title}</h1>
          {job.reference_number && (
            <p className="text-xs text-black/50">Ref #{job.reference_number}</p>
          )}
          {job.description && (
            <div className="mt-3 rounded-2xl border border-black/10 bg-white/70 p-4 text-sm text-black/70 whitespace-pre-wrap">
              {job.description}
            </div>
          )}
        </div>

        {error && (
          <div className="rounded-2xl border border-red-200 bg-red-50 p-4 text-red-700">{error}</div>
        )}

        {success ? (
          <div className="rounded-2xl border border-green-200 bg-green-50 p-5 text-green-800">
            <h2 className="font-semibold">Application submitted</h2>
            <p className="mt-1 text-sm">
              Thank you for applying. We will review your application and get back to you.
            </p>
          </div>
        ) : (
          <>
            <nav className="flex gap-2 text-sm">
              {([1, 2, 3] as const).map((s) => (
                <span
                  key={s}
                  className={`rounded-lg px-3 py-1 ${
                    step === s
                      ? 'bg-[var(--empops-accent)] text-white'
                      : 'bg-black/[0.04] text-black/60'
                  }`}
                >
                  {s === 1 ? 'Details' : s === 2 ? 'CV upload' : 'Complete'}
                </span>
              ))}
            </nav>

            {step === 1 && (
              <form
                onSubmit={handleSubmit((v) => createCandidate.mutate(v))}
                className="space-y-3 rounded-2xl border border-black/10 bg-white/70 p-4"
              >
                <label className="block space-y-1">
                  <span className="text-xs text-black/60">Full name</span>
                  <input className={inputClass} {...register('name')} />
                  {errors.name && (
                    <span className="text-xs text-red-700">{errors.name.message}</span>
                  )}
                </label>
                <label className="block space-y-1">
                  <span className="text-xs text-black/60">Email</span>
                  <input type="email" className={inputClass} {...register('email')} />
                  {errors.email && (
                    <span className="text-xs text-red-700">{errors.email.message}</span>
                  )}
                </label>
                <label className="block space-y-1">
                  <span className="text-xs text-black/60">Profile URL (optional)</span>
                  <input className={inputClass} {...register('url')} />
                </label>
                <label className="block space-y-1">
                  <span className="text-xs text-black/60">Desired salary (optional)</span>
                  <input className={inputClass} {...register('desired_salary')} />
                </label>
                <label className="block space-y-1">
                  <span className="text-xs text-black/60">Notes (optional)</span>
                  <textarea className={`${inputClass} min-h-[80px]`} {...register('notes')} />
                </label>
                <button
                  type="submit"
                  disabled={createCandidate.isPending}
                  className="rounded-lg bg-[var(--empops-accent)] px-4 py-2 text-sm font-medium text-white disabled:opacity-60"
                >
                  Continue to CV upload
                </button>
              </form>
            )}

            {step === 2 && candidateUuid && (
              <section className="space-y-4 rounded-2xl border border-black/10 bg-white/70 p-4">
                <h2 className="font-medium">Upload your CV</h2>
                <p className="text-sm text-black/60">
                  Upload one or more files, then complete your application.
                </p>
                <input
                  ref={fileInputRef}
                  type="file"
                  accept=".pdf,.doc,.docx,.txt"
                  disabled={uploadBusy}
                  onChange={(e) => {
                    const file = e.target.files?.[0]
                    if (file) void handleFileUpload(file)
                  }}
                  className="text-sm"
                />
                {uploadProgress != null && (
                  <div className="space-y-1">
                    <div className="h-2 overflow-hidden rounded-full bg-black/10">
                      <div
                        className="h-full bg-[var(--empops-accent)] transition-all"
                        style={{ width: `${uploadProgress}%` }}
                      />
                    </div>
                    <p className="text-xs text-black/50">{Math.round(uploadProgress)}%</p>
                  </div>
                )}
                <button
                  type="button"
                  disabled={completeApplication.isPending}
                  onClick={() => completeApplication.mutate()}
                  className="rounded-lg bg-[var(--empops-accent)] px-4 py-2 text-sm font-medium text-white disabled:opacity-60"
                >
                  Submit application
                </button>
              </section>
            )}
          </>
        )}
      </div>
    </PublicShell>
  )
}
