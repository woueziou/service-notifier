import { createFileRoute } from "@tanstack/react-router"

export const Route = createFileRoute("/")({
  component: Dashboard,
})

function Dashboard() {
  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900 mb-6">Dashboard</h2>
      <div className="grid grid-cols-1 gap-6 sm:grid-cols-3">
        <div className="bg-white overflow-hidden shadow rounded-lg">
          <div className="p-5">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <div className="h-10 w-10 rounded-full bg-blue-100 flex items-center justify-center">
                  <span className="text-blue-600 text-lg font-bold">C</span>
                </div>
              </div>
              <div className="ml-5">
                <p className="text-sm font-medium text-gray-500 truncate">
                  Consumers
                </p>
                <p className="mt-1 text-3xl font-semibold text-gray-900">—</p>
              </div>
            </div>
          </div>
        </div>
        <div className="bg-white overflow-hidden shadow rounded-lg">
          <div className="p-5">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <div className="h-10 w-10 rounded-full bg-green-100 flex items-center justify-center">
                  <span className="text-green-600 text-lg font-bold">J</span>
                </div>
              </div>
              <div className="ml-5">
                <p className="text-sm font-medium text-gray-500 truncate">
                  Jobs Today
                </p>
                <p className="mt-1 text-3xl font-semibold text-gray-900">—</p>
              </div>
            </div>
          </div>
        </div>
        <div className="bg-white overflow-hidden shadow rounded-lg">
          <div className="p-5">
            <div className="flex items-center">
              <div className="flex-shrink-0">
                <div className="h-10 w-10 rounded-full bg-red-100 flex items-center justify-center">
                  <span className="text-red-600 text-lg font-bold">!</span>
                </div>
              </div>
              <div className="ml-5">
                <p className="text-sm font-medium text-gray-500 truncate">
                  DLQ
                </p>
                <p className="mt-1 text-3xl font-semibold text-gray-900">—</p>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}
