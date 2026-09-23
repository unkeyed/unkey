import type { PropsWithChildren } from "react";
import { ProjectGuard } from "./project-guard";

// The deploy data provider lives lower (apps/[appId], logs, requests) so the
// project home and the resource pages render without it.
export default function ProjectLayout({ children }: PropsWithChildren) {
  return <ProjectGuard>{children}</ProjectGuard>;
}
