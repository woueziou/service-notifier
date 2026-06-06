import { createFileRoute } from "@tanstack/react-router"

export const Route = createFileRoute("/jobs/")({
  component: JobsList,
})

function JobsList() {
  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900 mb-6">Jobs</h2>
      <div className="bg-white shadow overflow-hidden sm:rounded-md">
        <div className="p-6 text-center text-gray-500">
          Job listing will be implemented once the backend provides the endpoint.
        </div>
      </div>
    </div>
  )
}
