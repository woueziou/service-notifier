import { createFileRoute } from "@tanstack/react-router"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { useState } from "react"
import { listDLQ, replayDLQ } from "@/lib/api"

export const Route = createFileRoute("/dlq")({
  component: DLQPage,
})

function DLQPage() {
  const queryClient = useQueryClient()
  const [replaying, setReplaying] = useState<string | null>(null)

  const { data, isLoading, isError, error } = useQuery({
    queryKey: ["dlq"],
    queryFn: () => listDLQ({ count: 100 }),
    refetchInterval: 10_000, // auto-refresh every 10s
  })

  const replayMutation = useMutation({
    mutationFn: (id: string) => replayDLQ(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["dlq"] })
      queryClient.invalidateQueries({ queryKey: ["jobs"] })
    },
  })

  const handleReplay = async (id: string) => {
    setReplaying(id)
    try {
      await replayMutation.mutateAsync(id)
    } finally {
      setReplaying(null)
    }
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold text-gray-900">
          Dead Letter Queue
        </h2>
        <span className="text-sm text-gray-500">
          {data ? `${data.length} message(s)` : ""}
        </span>
      </div>

      <div className="bg-white shadow overflow-hidden sm:rounded-md">
        {isLoading ? (
          <div className="p-6 text-center text-gray-500">Loading...</div>
        ) : isError ? (
          <div className="p-6 text-center text-red-500">
            Error: {(error as Error).message}
          </div>
        ) : !data?.length ? (
          <div className="p-6 text-center text-gray-500">
            No messages in the dead letter queue.
          </div>
        ) : (
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Message ID
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Fields
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Action
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {data.map((entry) => (
                <tr key={entry.id} className="hover:bg-gray-50">
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-900">
                    {entry.id}
                  </td>
                  <td className="px-6 py-4 text-sm text-gray-500 max-w-lg">
                    <div className="max-h-32 overflow-y-auto">
                      {Object.entries(entry.fields).map(([key, val]) => (
                        <div key={key} className="flex">
                          <span className="font-medium text-gray-700 mr-2 min-w-[100px]">
                            {key}:
                          </span>
                          <span className="truncate">{String(val)}</span>
                        </div>
                      ))}
                    </div>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm">
                    <button
                      onClick={() => handleReplay(entry.id)}
                      disabled={replaying === entry.id}
                      className="inline-flex items-center px-3 py-1.5 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-50"
                    >
                      {replaying === entry.id ? "Replaying..." : "Replay"}
                    </button>
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
