import { createFileRoute } from "@tanstack/react-router"

export const Route = createFileRoute("/jobs/$jobId")({
  component: JobDetail,
})

function JobDetail() {
  const { jobId } = Route.useParams()

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900 mb-6">Job Detail</h2>
      <div className="bg-white shadow rounded-lg p-6">
        <p className="text-gray-500">Job ID: {jobId}</p>
      </div>
    </div>
  )
}
