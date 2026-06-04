import type { PropsWithChildren } from "react";

// Project-level pass-through. The app-scoped (data provider, navigation)
// lives in apps/[appSlug]/layout.tsx so the project home can render without it.
export default function ProjectLayout({ children }: PropsWithChildren) {
  return children;
}
