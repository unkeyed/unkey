"use client";

import { notFound } from "next/navigation";

// Client so it only runs once the environment layout has resolved the slug;
// a legacy path such as /apps/x/deployments/<id> redirects there instead.
export default function UnknownAppPage() {
  notFound();
}
