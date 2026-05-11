import { useEffect, useState } from "react";
import { api } from "../lib/api";
import type { UserSummary } from "../gen/apps/webapp/v1/api_pb";

export function UsersPage() {
  const [users, setUsers] = useState<UserSummary[]>([]);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .listUsers({ pageSize: 50 })
      .then((res) => setUsers(res.users))
      .catch((err) => setError(err instanceof Error ? err.message : String(err)));
  }, []);

  return (
    <section className="space-y-4">
      <h1 className="text-xl font-semibold">Users</h1>
      {error && <p className="text-sm text-red-600">{error}</p>}
      <table className="w-full table-auto border-collapse text-sm">
        <thead>
          <tr className="text-left">
            <th className="border-b py-2 pr-4">Email</th>
            <th className="border-b py-2 pr-4">Name</th>
            <th className="border-b py-2">Roles</th>
          </tr>
        </thead>
        <tbody>
          {users.map((u) => (
            <tr key={u.id}>
              <td className="border-b py-2 pr-4">{u.email}</td>
              <td className="border-b py-2 pr-4">{u.name}</td>
              <td className="border-b py-2">{u.roles.join(", ") || "—"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </section>
  );
}
