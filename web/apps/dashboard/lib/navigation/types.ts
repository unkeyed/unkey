import type { ElementType, ReactNode } from "react";

// A single renderable navigation row. Pre-resolved: hrefs are built,
// active state is computed against the current route. Producers (the
// build functions in ./leaves.ts) flatten URL params + segments into
// this shape so the renderer stays a dumb map.
//
// This is the *rendering* shape — not a registry of destinations. When
// cmd-k lands it'll need a parallel NavDescriptor type (keywords,
// visibility predicate, path builder) plus a resolver that produces
// ResolvedNavLink[] from descriptors + context. Don't reuse this type
// for that purpose.
export type ResolvedNavLink = {
  key: string;
  label: ReactNode;
  href: string;
  icon?: ElementType;
  isActive: boolean;
  disabled?: boolean;
  external?: boolean;
  tag?: ReactNode;
};
