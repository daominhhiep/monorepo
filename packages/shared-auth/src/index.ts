// Cross-app helpers for the FE auth slice. Per-app providers (the one in
// each apps/*/web/src/lib/auth.tsx) are intentionally kept local — this
// package only exposes pure utilities shared across apps.

export type Role = string;

export function hasRole(roles: Role[] | undefined, required: Role): boolean {
  return !!roles?.includes(required);
}

export function hasAnyRole(roles: Role[] | undefined, required: Role[]): boolean {
  if (!roles) return false;
  return required.some((r) => roles.includes(r));
}
