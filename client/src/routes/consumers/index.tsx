import { createFileRoute, Link } from "@tanstack/react-router"
import { useQuery } from "@tanstack/react-query"
import { listConsumers } from "@/lib/api"

export const Route = createFileRoute("/consumers/")({
  component: ConsumersList,
})

function ConsumersList() {
  const { data: consumers, isLoading } = useQuery({
    queryKey: ["consumers"],
    queryFn: listConsumers,
  })

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold text-gray-900">Consumers</h2>
        <Link
          to="/consumers/create"
          className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700"
        >
          Create Consumer
        </Link>
      </div>
      <div className="bg-white shadow overflow-hidden sm:rounded-md">
        {isLoading ? (
          <div className="p-6 text-center text-gray-500">Loading...</div>
        ) : (
          <ul className="divide-y divide-gray-200">
            {consumers?.map((c) => (
              <li key={c.id}>
                <Link
                  to="/consumers/$consumerId"
                  params={{ consumerId: c.id }}
                  className="block hover:bg-gray-50"
                >
                  <div className="px-6 py-4">
                    <div className="flex items-center justify-between">
                      <div>
                        <p className="text-sm font-medium text-blue-600 truncate">
                          {c.name}
                        </p>
                        <p className="text-sm text-gray-500">
                          {c.sender_email}
                        </p>
                      </div>
                      <div className="flex items-center space-x-2">
                        {c.suspended && (
                          <span className="px-2 py-1 text-xs font-medium rounded-full bg-red-100 text-red-800">
                            Suspended
                          </span>
                        )}
                        {!c.active && (
                          <span className="px-2 py-1 text-xs font-medium rounded-full bg-gray-100 text-gray-800">
                            Inactive
                          </span>
                        )}
                      </div>
                    </div>
                  </div>
                </Link>
              </li>
            ))}
          </ul>
        )}
      </div>
    </div>
  )
}
