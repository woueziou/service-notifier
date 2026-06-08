import { createFileRoute } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import { getStats } from "@/lib/api"
export const Route = createFileRoute("/stats")({
  component: StatsPage,
})

function StatsPage() {
  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["stats"],
    queryFn: getStats,
    refetchInterval: 15_000, // refresh every 15s
  })

  if (isLoading) {
    return <div className="text-gray-500">Loading stats...</div>
  }

  if (isError) {
    return (
      <div className="text-red-600">Error: {(error as Error).message}</div>
    )
  }

  if (!data) {
    return <div className="text-gray-500">No stats available.</div>
  }

  const { summary, consumers } = data

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900 mb-6">
        Statistics
      </h2>

      {/* Summary cards */}
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-4 mb-8">
        <SummaryCard
          label="Total Consumers"
          value={summary.total_consumers}
          color="blue"
        />
        <SummaryCard
          label="Active"
          value={summary.active_consumers}
          color="green"
        />
        <SummaryCard
          label="Suspended"
          value={summary.suspended_consumers}
          color="red"
        />
        <SummaryCard
          label="Total Jobs"
          value={summary.total_jobs}
          color="purple"
        />
      </div>

      {/* Per-consumer table */}
      <h3 className="text-lg font-medium text-gray-900 mb-4">
        Per-Consumer Details
      </h3>
      <div className="bg-white shadow overflow-hidden sm:rounded-md">
        {consumers.length === 0 ? (
          <div className="p-6 text-center text-gray-500">
            No consumers found.
          </div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Consumer
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Status
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Total Jobs
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Bounce Rate
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Rate Limit
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {consumers.map((c) => (
                <tr key={c.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="text-sm font-medium text-gray-900">
                      {c.name}
                    </div>
                    <div className="text-sm text-gray-500">
                      {c.sender_email}
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    {c.suspended ? (
                      <span className="px-2 py-1 text-xs font-medium rounded-full bg-red-100 text-red-800">
                        Suspended
                      </span>
                    ) : (
                      <span className="px-2 py-1 text-xs font-medium rounded-full bg-green-100 text-green-800">
                        Active
                      </span>
                    )}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                    {c.total_jobs}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <BounceRateBadge rate={c.bounce_rate} />
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap">
                    <div className="flex items-center space-x-2">
                      <div className="flex-1 bg-gray-200 rounded-full h-2.5 w-24">
                        <div
                          className={`h-2.5 rounded-full ${
                            c.rate_limit_current > c.rate_limit_max * 0.8
                              ? "bg-red-500"
                              : c.rate_limit_current > c.rate_limit_max * 0.5
                              ? "bg-yellow-500"
                              : "bg-green-500"
                          }`}
                          style={{
                            width: `${Math.min(
                              100,
                              (c.rate_limit_current / c.rate_limit_max) * 100
                            )}%`,
                          }}
                        />
                      </div>
                      <span className="text-sm text-gray-500 min-w-[60px]">
                        {c.rate_limit_current}/{c.rate_limit_max}
                      </span>
                    </div>
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

function SummaryCard({
  label,
  value,
  color,
}: {
  label: string
  value: number
  color: string
}) {
  const colors: Record<string, string> = {
    blue: "bg-blue-100 text-blue-600",
    green: "bg-green-100 text-green-600",
    red: "bg-red-100 text-red-600",
    purple: "bg-purple-100 text-purple-600",
  }

  return (
    <div className="bg-white overflow-hidden shadow rounded-lg">
      <div className="p-5">
        <div className="flex items-center">
          <div className="flex-shrink-0">
            <div
              className={`h-12 w-12 rounded-full ${colors[color] || colors.blue} flex items-center justify-center`}
            >
              <span className="text-lg font-bold">{value}</span>
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
    </div>
  )
}

function BounceRateBadge({ rate }: { rate: number }) {
  const pct = (rate * 100).toFixed(1)
  const isHigh = rate > 0.2
  const isMedium = rate > 0.1

  return (
    <span
      className={`px-2 py-1 text-xs font-medium rounded-full ${
        isHigh
          ? "bg-red-100 text-red-800"
          : isMedium
          ? "bg-yellow-100 text-yellow-800"
          : "bg-green-100 text-green-800"
      }`}
    >
      {pct}%
    </span>
  )
}
