import { useAuth } from "../lib/auth";

export function HomePage() {
  const { principal } = useAuth();
  return (
    <section className="space-y-4">
      <h1 className="text-2xl font-semibold">Welcome</h1>
      {principal ? (
        <p>
          Signed in as <strong>{principal.name}</strong> ({principal.email}).
        </p>
      ) : (
        <p className="text-neutral-600">Log in or register to continue.</p>
      )}
    </section>
  );
}
