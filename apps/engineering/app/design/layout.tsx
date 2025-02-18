import { componentSource } from "@/app/source";
import "@unkey/ui/css";
import { DocsLayout } from "fumadocs-ui/layouts/notebook";
import type { ReactNode } from "react";
import { baseOptions } from "../layout.config";
export default function Layout({ children }: { children: ReactNode }) {
  return (
    <DocsLayout tree={componentSource.pageTree} {...baseOptions}>
      {children}
    </DocsLayout>
  );
}
