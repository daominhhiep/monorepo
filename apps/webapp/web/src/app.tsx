import { Link, Route, Routes } from "react-router-dom";
import { ProtectedRoute, useAuth } from "./lib/auth";
import { LoginPage } from "./pages/login";
import { RegisterPage } from "./pages/register";
import { HomePage } from "./pages/home";
import { UsersPage } from "./pages/users";

export function App() {
  const { principal, logout } = useAuth();
  return (
    <div className="mx-auto max-w-5xl p-6">
      <header className="mb-6 flex items-center justify-between border-b border-neutral-200 pb-4">
        <Link to="/" className="text-xl font-semibold">base-microservice</Link>
        <nav className="flex items-center gap-4 text-sm">
          {principal ? (
            <>
              <Link to="/users" className="hover:underline">Users</Link>
              <span className="text-neutral-500">{principal.email}</span>
              <button
                onClick={() => void logout()}
                className="rounded bg-neutral-900 px-3 py-1 text-white hover:bg-neutral-700"
              >
                Logout
              </button>
            </>
          ) : (
            <>
              <Link to="/login" className="hover:underline">Login</Link>
              <Link to="/register" className="hover:underline">Register</Link>
            </>
          )}
        </nav>
      </header>
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route
          path="/users"
          element={
            <ProtectedRoute>
              <UsersPage />
            </ProtectedRoute>
          }
        />
      </Routes>
    </div>
  );
}
