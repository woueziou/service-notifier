import { createRootRoute, Link, Outlet, useNavigate } from "@tanstack/react-router"
import { TanStackRouterDevtools } from "@tanstack/router-devtools"
import { useEffect, useState } from "react"
import { whoami, logout } from "@/lib/api"

export const Route = createRootRoute({
  component: RootLayout,
})

function RootLayout() {
  const navigate = useNavigate()
  const [username, setUsername] = useState<string | null>(null)
  const [checkingAuth, setCheckingAuth] = useState(true)

  useEffect(() => {
    // If already on /login, skip auth check
    if (window.location.pathname === "/login") {
      setCheckingAuth(false)
      return
    }

    whoami()
      .then((me) => {
        setUsername(me.username)
        setCheckingAuth(false)
      })
      .catch(() => {
        // Not authenticated — redirect to login
        navigate({ to: "/login" })
      })
  }, [navigate])

  async function handleLogout() {
    await logout()
    setUsername(null)
    navigate({ to: "/login" })
  }

  if (checkingAuth) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-gray-500">Loading...</div>
      </div>
    )
  }

  // Login page — render without the admin nav
  if (window.location.pathname === "/login") {
    return <Outlet />
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <nav className="bg-white border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-16">
            <div className="flex">
              <div className="flex-shrink-0 flex items-center">
                <h1 className="text-xl font-bold text-gray-900">Notifier</h1>
              </div>
              <div className="ml-10 flex items-center space-x-4">
                <Link
                  to="/"
                  className="px-3 py-2 rounded-md text-sm font-medium text-gray-700 hover:text-gray-900 hover:bg-gray-50"
                  activeProps={{ className: "text-blue-600 bg-blue-50" }}
                >
                  Dashboard
                </Link>
                <Link
                  to="/consumers"
                  className="px-3 py-2 rounded-md text-sm font-medium text-gray-700 hover:text-gray-900 hover:bg-gray-50"
                  activeProps={{ className: "text-blue-600 bg-blue-50" }}
                >
                  Consumers
                </Link>
                <Link
                  to="/jobs"
                  className="px-3 py-2 rounded-md text-sm font-medium text-gray-700 hover:text-gray-900 hover:bg-gray-50"
                  activeProps={{ className: "text-blue-600 bg-blue-50" }}
                >
                  Jobs
                </Link>
                <Link
                  to="/dlq"
                  className="px-3 py-2 rounded-md text-sm font-medium text-gray-700 hover:text-gray-900 hover:bg-gray-50"
                  activeProps={{ className: "text-red-600 bg-red-50" }}
                >
                  DLQ
                </Link>
                <Link
                  to="/stats"
                  className="px-3 py-2 rounded-md text-sm font-medium text-gray-700 hover:text-gray-900 hover:bg-gray-50"
                  activeProps={{ className: "text-purple-600 bg-purple-50" }}
                >
                  Stats
                </Link>
              </div>
            </div>
            <div className="flex items-center space-x-4">
              {username && (
                <span className="text-sm text-gray-500">{username}</span>
              )}
              <button
                onClick={handleLogout}
                className="px-3 py-2 rounded-md text-sm font-medium text-gray-700 hover:text-gray-900 hover:bg-gray-50"
              >
                Logout
              </button>
            </div>
          </div>
        </div>
      </nav>
      <main className="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
        <Outlet />
      </main>
      <TanStackRouterDevtools />
    </div>
  )
}
