import { createRootRoute, Link, Outlet, useNavigate } from "@tanstack/react-router"
import { TanStackRouterDevtools } from "@tanstack/router-devtools"
import { useEffect, useState } from "react"
import { whoami, logout, MeResponse } from "@/lib/api"

export const Route = createRootRoute({
  component: RootLayout,
})

function RootLayout() {
  const navigate = useNavigate()
  const [admin, setAdmin] = useState<MeResponse | null>(null)
  const [checkingAuth, setCheckingAuth] = useState(true)

  useEffect(() => {
    if (window.location.pathname === "/login") {
      setCheckingAuth(false)
      return
    }

    whoami()
      .then((me) => {
        setAdmin(me)
        setCheckingAuth(false)
      })
      .catch(() => {
        navigate({ to: "/login" })
      })
  }, [navigate])

  async function handleLogout() {
    await logout()
    setAdmin(null)
    navigate({ to: "/login" })
  }

  if (checkingAuth) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-gray-500">Loading...</div>
      </div>
    )
  }

  if (window.location.pathname === "/login") {
    return <Outlet />
  }

  const canWrite = admin?.role === "admin" || admin?.role === "super_admin"

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
                {canWrite && (
                  <Link
                    to="/admin-users"
                    className="px-3 py-2 rounded-md text-sm font-medium text-gray-700 hover:text-gray-900 hover:bg-gray-50"
                    activeProps={{ className: "text-yellow-600 bg-yellow-50" }}
                  >
                    Admins
                  </Link>
                )}
              </div>
            </div>
            <div className="flex items-center space-x-4">
              <span className="text-sm text-gray-500">
                {admin?.email}
              </span>
              <RoleBadge role={admin?.role} />
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

function RoleBadge({ role }: { role?: string }) {
  if (!role) return null

  const styles: Record<string, string> = {
    super_admin: "bg-purple-100 text-purple-800",
    admin: "bg-blue-100 text-blue-800",
    viewer: "bg-gray-100 text-gray-800",
  }

  return (
    <span
      className={`px-2 py-1 text-xs font-medium rounded-full ${
        styles[role] || "bg-gray-100 text-gray-800"
      }`}
    >
      {role?.replace("_", " ")}
    </span>
  )
}
