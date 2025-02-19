import { infrastructureSource } from "@/app/source";
import { DocsLayout } from "fumadocs-ui/layouts/notebook";
import type { ReactNode } from "react";
import { baseOptions } from "../layout.config";

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <div>
      <DocsLayout tree={infrastructureSource.pageTree} {...baseOptions}>
        {children}
      </DocsLayout>
    </div>
  );
}
