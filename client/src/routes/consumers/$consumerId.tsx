import { createFileRoute } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import { getConsumer } from "@/lib/api"

export const Route = createFileRoute("/consumers/$consumerId")({
  component: ConsumerDetail,
})

function ConsumerDetail() {
  const { consumerId } = Route.useParams()
  const { data: consumer, isLoading } = useQuery({
    queryKey: ["consumer", consumerId],
    queryFn: () => getConsumer(consumerId),
  })

  if (isLoading) {
    return <div className="text-gray-500">Loading...</div>
  }

  if (!consumer) {
    return <div className="text-red-600">Consumer not found.</div>
  }

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900 mb-6">
        {consumer.name}
      </h2>
      <div className="bg-white shadow rounded-lg p-6">
        <dl className="grid grid-cols-1 gap-x-4 gap-y-6 sm:grid-cols-2">
          <div>
            <dt className="text-sm font-medium text-gray-500">ID</dt>
            <dd className="mt-1 text-sm text-gray-900 font-mono">
              {consumer.id}
            </dd>
          </div>
          <div>
            <dt className="text-sm font-medium text-gray-500">Sender Email</dt>
            <dd className="mt-1 text-sm text-gray-900">
              {consumer.sender_email}
            </dd>
          </div>
          <div>
            <dt className="text-sm font-medium text-gray-500">Status</dt>
            <dd className="mt-1">
              {consumer.suspended ? (
                <span className="px-2 py-1 text-xs font-medium rounded-full bg-red-100 text-red-800">
                  Suspended
                </span>
              ) : (
                <span className="px-2 py-1 text-xs font-medium rounded-full bg-green-100 text-green-800">
                  Active
                </span>
              )}
            </dd>
          </div>
          <div>
            <dt className="text-sm font-medium text-gray-500">Created</dt>
            <dd className="mt-1 text-sm text-gray-900">
              {new Date(consumer.created_at).toLocaleString()}
            </dd>
          </div>
        </dl>
      </div>
    </div>
  )
}
