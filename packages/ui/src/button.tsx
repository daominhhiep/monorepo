import type { ButtonHTMLAttributes } from "react";

type Variant = "primary" | "ghost";

export function Button({
  variant = "primary",
  className = "",
  ...rest
}: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant }) {
  const base = "rounded px-4 py-2 text-sm font-medium transition disabled:opacity-60";
  const variants: Record<Variant, string> = {
    primary: "bg-neutral-900 text-white hover:bg-neutral-700",
    ghost: "bg-transparent text-neutral-900 hover:bg-neutral-100",
  };
  return <button className={`${base} ${variants[variant]} ${className}`} {...rest} />;
}
