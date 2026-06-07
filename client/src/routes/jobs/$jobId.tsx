import { createFileRoute, Link } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import { getJob } from "@/lib/api"
import { StatusBadge } from "@/routes/index"

export const Route = createFileRoute("/jobs/$jobId")({
  component: JobDetail,
})

function JobDetail() {
  const { jobId } = Route.useParams()
  const { data: job, isLoading, isError, error } = useQuery({
    queryKey: ["job", jobId],
    queryFn: () => getJob(jobId),
  })

  if (isLoading) {
    return <div className="text-gray-500">Loading...</div>
  }

  if (isError) {
    return (
      <div className="text-red-600">
        Error: {(error as Error).message}
      </div>
    )
  }

  if (!job) {
    return <div className="text-red-600">Job not found.</div>
  }

  return (
    <div>
      <div className="flex items-center space-x-4 mb-6">
        <Link
          to="/jobs"
          className="text-sm text-blue-600 hover:text-blue-900"
        >
          &larr; Back to Jobs
        </Link>
        <h2 className="text-2xl font-bold text-gray-900">Job Detail</h2>
      </div>

      <div className="bg-white shadow rounded-lg overflow-hidden">
        {/* Header */}
        <div className="px-6 py-4 border-b border-gray-200 flex items-center justify-between">
          <div className="flex items-center space-x-3">
            <StatusBadge status={job.status} />
            <span className="text-sm text-gray-500">
              {job.status === "delivered" && job.delivered_at
                ? `Delivered ${new Date(job.delivered_at).toLocaleString()}`
                : `Created ${new Date(job.created_at).toLocaleString()}`}
            </span>
          </div>
        </div>

        {/* Details */}
        <div className="px-6 py-4">
          <dl className="grid grid-cols-1 gap-x-4 gap-y-6 sm:grid-cols-2">
            <DetailField label="Job ID" value={job.id} mono />
            <DetailField label="Consumer ID" value={job.consumer_id} mono />
            <DetailField label="To" value={job.to} />
            <DetailField label="Subject" value={job.subject || "—"} />
            <div className="sm:col-span-2">
              <DetailField label="Error" value={job.error || "—"} />
            </div>
            <DetailField
              label="Created"
              value={new Date(job.created_at).toLocaleString()}
            />
            <DetailField
              label="Updated"
              value={new Date(job.updated_at).toLocaleString()}
            />
            <DetailField
              label="Delivered At"
              value={
                job.delivered_at
                  ? new Date(job.delivered_at).toLocaleString()
                  : "—"
              }
            />
            <DetailField label="Status" value={job.status} />
          </dl>
        </div>

        {/* Body */}
        {job.body && (
          <div className="px-6 py-4 border-t border-gray-200">
            <h3 className="text-sm font-medium text-gray-500 mb-2">
              Body
            </h3>
            <pre className="text-sm text-gray-900 bg-gray-50 rounded-md p-4 overflow-x-auto whitespace-pre-wrap max-h-96 overflow-y-auto">
              {job.body}
            </pre>
          </div>
        )}
      </div>
    </div>
  )
}

function DetailField({
  label,
  value,
  mono,
}: {
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div>
      <dt className="text-sm font-medium text-gray-500">{label}</dt>
      <dd
        className={`mt-1 text-sm text-gray-900 ${
          mono ? "font-mono break-all" : ""
        }`}
      >
        {value}
      </dd>
    </div>
  )
}
