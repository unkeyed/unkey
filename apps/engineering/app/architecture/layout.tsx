import { architectureSource } from "@/app/source";
import { DocsLayout } from "fumadocs-ui/layouts/notebook";
import type { ReactNode } from "react";
import { baseOptions } from "../layout.config";

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <div className="font-mono">
      <DocsLayout tree={architectureSource.pageTree} {...baseOptions}>
        {children}
      </DocsLayout>
    </div>
  );
}
