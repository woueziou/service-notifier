import { createFileRoute } from "@tanstack/react-router"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { useState } from "react"
import { listAdminUsers, addAdminUser, deleteAdminUser, whoami } from "@/lib/api"

export const Route = createFileRoute("/admin-users")({
  component: AdminUsersPage,
})

function AdminUsersPage() {
  const queryClient = useQueryClient()
  const [showAdd, setShowAdd] = useState(false)
  const [newEmail, setNewEmail] = useState("")
  const [newRole, setNewRole] = useState<"viewer" | "admin">("admin")
  const [error, setError] = useState("")

  // Check if current user is super_admin
  const meQuery = useQuery({
    queryKey: ["whoami"],
    queryFn: whoami,
  })
  const isSuperAdmin = meQuery.data?.role === "super_admin"
  const canManage = meQuery.data?.role === "admin" || isSuperAdmin

  const usersQuery = useQuery({
    queryKey: ["admin-users"],
    queryFn: listAdminUsers,
  })

  const addMutation = useMutation({
    mutationFn: () => addAdminUser(newEmail, newRole),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-users"] })
      setShowAdd(false)
      setNewEmail("")
      setNewRole("admin")
      setError("")
    },
    onError: (err) => {
      setError((err as Error).message)
    },
  })

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteAdminUser(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-users"] })
    },
  })

  if (!canManage) {
    return (
      <div className="text-red-600 p-6">
        You do not have permission to manage admin users.
      </div>
    )
  }

  const users = usersQuery.data || []

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold text-gray-900">Admin Users</h2>
        <button
          onClick={() => setShowAdd(!showAdd)}
          className="px-4 py-2 bg-blue-600 text-white text-sm font-medium rounded-md hover:bg-blue-700"
        >
          {showAdd ? "Cancel" : "Add Admin"}
        </button>
      </div>

      {showAdd && (
        <div className="bg-white shadow rounded-lg p-6 mb-6">
          <h3 className="text-lg font-medium text-gray-900 mb-4">
            Add Admin User
          </h3>
          <div className="space-y-4">
            <div>
              <label
                htmlFor="new-email"
                className="block text-sm font-medium text-gray-700"
              >
                Email
              </label>
              <input
                id="new-email"
                type="email"
                value={newEmail}
                onChange={(e) => setNewEmail(e.target.value)}
                className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                placeholder="colleague@example.com"
              />
            </div>
            <div>
              <label
                htmlFor="new-role"
                className="block text-sm font-medium text-gray-700"
              >
                Role
              </label>
              <select
                id="new-role"
                value={newRole}
                onChange={(e) =>
                  setNewRole(e.target.value as "viewer" | "admin")
                }
                className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
              >
                <option value="viewer">Viewer (read-only)</option>
                <option value="admin">Admin (can add users)</option>
              </select>
              <p className="mt-1 text-xs text-gray-500">
                Only super admins can assign the super_admin role directly via
                the database.
              </p>
            </div>

            {error && (
              <div className="text-red-600 text-sm">{error}</div>
            )}

            <button
              onClick={() => addMutation.mutate()}
              disabled={addMutation.isPending || !newEmail}
              className="px-4 py-2 bg-green-600 text-white text-sm font-medium rounded-md hover:bg-green-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {addMutation.isPending ? "Adding..." : "Add User"}
            </button>
          </div>
        </div>
      )}

      {usersQuery.isLoading ? (
        <div className="text-gray-500">Loading...</div>
      ) : (
        <div className="bg-white shadow overflow-hidden sm:rounded-md">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Email
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Role
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Added
                </th>
                {isSuperAdmin && (
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Actions
                  </th>
                )}
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {users.length === 0 ? (
                <tr>
                  <td
                    colSpan={isSuperAdmin ? 4 : 3}
                    className="px-6 py-4 text-center text-gray-500"
                  >
                    No admin users found.
                  </td>
                </tr>
              ) : (
                users.map((u) => (
                  <tr key={u.id} className="hover:bg-gray-50">
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                      {u.email}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap">
                      <RoleBadge role={u.role} />
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {new Date(u.created_at).toLocaleDateString()}
                    </td>
                    {isSuperAdmin && (
                      <td className="px-6 py-4 whitespace-nowrap text-right text-sm">
                        <button
                          onClick={() => {
                            if (
                              confirm(
                                `Remove ${u.email} from admin users?`
                              )
                            ) {
                              deleteMutation.mutate(u.id)
                            }
                          }}
                          disabled={deleteMutation.isPending}
                          className="text-red-600 hover:text-red-900 disabled:opacity-50"
                        >
                          Remove
                        </button>
                      </td>
                    )}
                  </tr>
                ))
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

function RoleBadge({ role }: { role: string }) {
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
      {role.replace("_", " ")}
    </span>
  )
}
