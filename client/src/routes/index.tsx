import { createFileRoute, Link } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import { listConsumers, listDLQ, listJobs } from "@/lib/api"

export const Route = createFileRoute("/")({
  component: Dashboard,
})

function StatCard({
  label,
  value,
  icon,
  color,
  href,
}: {
  label: string
  value: string | number
  icon: string
  color: string
  href: string
}) {
  const colors: Record<string, string> = {
    blue: "bg-blue-100 text-blue-600",
    green: "bg-green-100 text-green-600",
    red: "bg-red-100 text-red-600",
    purple: "bg-purple-100 text-purple-600",
  }

  return (
    <Link to={href} className="block bg-white overflow-hidden shadow rounded-lg hover:shadow-md transition-shadow">
      <div className="p-5">
        <div className="flex items-center">
          <div className="flex-shrink-0">
            <div
              className={`h-12 w-12 rounded-full ${colors[color] || colors.blue} flex items-center justify-center`}
            >
              <span className="text-lg font-bold">{icon}</span>
            </div>
          </div>
          <div className="ml-5 w-0 flex-1">
            <p className="text-sm font-medium text-gray-500 truncate">
              {label}
            </p>
            <p className="mt-1 text-3xl font-semibold text-gray-900">
              {value}
            </p>
          </div>
        </div>
      </div>
    </Link>
  )
}

function Dashboard() {
  const consumers = useQuery({
    queryKey: ["consumers"],
    queryFn: listConsumers,
  })

  const jobs = useQuery({
    queryKey: ["jobs", "all"],
    queryFn: () => listJobs({ limit: 200 }),
  })

  const dlq = useQuery({
    queryKey: ["dlq"],
    queryFn: () => listDLQ({ count: 100 }),
  })

  const consumersCount = consumers.data?.length ?? "—"
  const dlqDepth = dlq.data?.length ?? "—"

  const jobsToday =
    jobs.data?.jobs.filter((j) => {
      const created = new Date(j.created_at)
      const now = new Date()
      return (
        created.getDate() === now.getDate() &&
        created.getMonth() === now.getMonth() &&
        created.getFullYear() === now.getFullYear()
      )
    }).length ?? "—"

  const isLoading =
    consumers.isLoading || jobs.isLoading || dlq.isLoading
  const loadingValue = isLoading ? "..." : undefined

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900 mb-6">Dashboard</h2>
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard
          label="Consumers"
          value={consumersCount}
          icon="C"
          color="blue"
          href="/consumers"
        />
        <StatCard
          label="Jobs Today"
          value={loadingValue ?? jobsToday}
          icon="J"
          color="green"
          href="/jobs"
        />
        <StatCard
          label="DLQ"
          value={dlqDepth}
          icon="!"
          color="red"
          href="/dlq"
        />
        <StatCard
          label="Total Jobs"
          value={loadingValue ?? jobs.data?.total ?? "—"}
          icon="#"
          color="purple"
          href="/jobs"
        />
      </div>

      {/* Recent jobs */}
      <h3 className="text-lg font-medium text-gray-900 mt-8 mb-4">
        Recent Jobs
      </h3>
      <div className="bg-white shadow overflow-hidden sm:rounded-md">
        {jobs.isLoading ? (
          <div className="p-6 text-center text-gray-500">Loading...</div>
        ) : !jobs.data?.jobs.length ? (
          <div className="p-6 text-center text-gray-500">No jobs yet.</div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  To
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Subject
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Created
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Action
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {jobs.data.jobs.slice(0, 10).map((job) => (
                <tr key={job.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <StatusBadge status={job.status} />
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {job.to}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 max-w-xs truncate">
                    {job.subject}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {new Date(job.created_at).toLocaleString()}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    <Link
                      to="/jobs/$jobId"
                      params={{ jobId: job.id }}
                      className="text-blue-600 hover:text-blue-900"
                    >
                      View
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

export function StatusBadge({ status }: { status: string }) {
  const styles: Record<string, string> = {
    pending:
      "bg-yellow-100 text-yellow-800",
    delivered:
      "bg-green-100 text-green-800",
    failed:
      "bg-red-100 text-red-800",
    bounced:
      "bg-orange-100 text-orange-800",
  }

  return (
    <span
      className={`px-2 py-1 text-xs font-medium rounded-full ${
        styles[status] || "bg-gray-100 text-gray-800"
      }`}
    >
      {status}
    </span>
  )
}
