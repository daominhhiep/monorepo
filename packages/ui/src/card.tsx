import type { HTMLAttributes } from "react";

export function Card({ className = "", ...rest }: HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={`rounded-lg border border-neutral-200 bg-white p-4 shadow-sm ${className}`}
      {...rest}
    />
  );
}
