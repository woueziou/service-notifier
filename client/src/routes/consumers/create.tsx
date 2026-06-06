import { createFileRoute, useNavigate } from "@tanstack/react-router"
import { useState } from "react"
import { createConsumer, CreateConsumerResponse } from "@/lib/api"

export const Route = createFileRoute("/consumers/create")({
  component: CreateConsumer,
})

function CreateConsumer() {
  const navigate = useNavigate()
  const [name, setName] = useState("")
  const [emailPrefix, setEmailPrefix] = useState("")
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState<CreateConsumerResponse | null>(null)
  const [error, setError] = useState("")

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    setError("")
    try {
      const resp = await createConsumer({ name, email_prefix: emailPrefix })
      setResult(resp)
    } catch (err: any) {
      setError(err.message || "Failed to create consumer")
    } finally {
      setLoading(false)
    }
  }

  if (result) {
    return (
      <div>
        <h2 className="text-2xl font-bold text-gray-900 mb-6">
          Consumer Created
        </h2>
        <div className="bg-white shadow rounded-lg p-6">
          <div className="space-y-4">
            <div>
              <label className="text-sm font-medium text-gray-500">Name</label>
              <p className="text-gray-900">{result.name}</p>
            </div>
            <div>
              <label className="text-sm font-medium text-gray-500">
                Sender Email
              </label>
              <p className="text-gray-900">{result.sender_email}</p>
            </div>
            <div>
              <label className="text-sm font-medium text-gray-500">
                API Key
              </label>
              <div className="mt-1 flex rounded-md shadow-sm">
                <input
                  type="text"
                  readOnly
                  value={result.api_key}
                  className="flex-1 min-w-0 block w-full px-3 py-2 rounded-md border border-gray-300 bg-gray-50 text-gray-900 text-sm font-mono"
                  onClick={(e) => (e.target as HTMLInputElement).select()}
                />
              </div>
              <p className="mt-2 text-sm text-red-600 font-medium">
                ⚠️ Copy this key now. It will not be shown again.
              </p>
            </div>
          </div>
          <button
            onClick={() => navigate({ to: "/consumers" })}
            className="mt-6 inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50"
          >
            Back to Consumers
          </button>
        </div>
      </div>
    )
  }

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900 mb-6">
        Create Consumer
      </h2>
      <form onSubmit={handleSubmit} className="bg-white shadow rounded-lg p-6">
        <div className="space-y-4">
          <div>
            <label
              htmlFor="name"
              className="block text-sm font-medium text-gray-700"
            >
              Name
            </label>
            <input
              type="text"
              id="name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm text-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
              placeholder="e.g., automater"
            />
          </div>
          <div>
            <label
              htmlFor="emailPrefix"
              className="block text-sm font-medium text-gray-700"
            >
              Email Prefix
            </label>
            <input
              type="text"
              id="emailPrefix"
              value={emailPrefix}
              onChange={(e) => setEmailPrefix(e.target.value)}
              required
              className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm text-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
              placeholder="e.g., automater-noreply"
            />
            <p className="mt-1 text-xs text-gray-500">
              Will be used as: {emailPrefix || "prefix"}&#64;yourdomain.com
            </p>
          </div>
          {error && (
            <div className="text-sm text-red-600 bg-red-50 p-3 rounded-md">
              {error}
            </div>
          )}
        </div>
        <div className="mt-6 flex space-x-3">
          <button
            type="submit"
            disabled={loading}
            className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md shadow-sm text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-50"
          >
            {loading ? "Creating..." : "Create Consumer"}
          </button>
          <button
            type="button"
            onClick={() => navigate({ to: "/consumers" })}
            className="inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50"
          >
            Cancel
          </button>
        </div>
      </form>
    </div>
  )
}
