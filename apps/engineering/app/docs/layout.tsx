import { source } from "@/app/source";
import { RootToggle } from "fumadocs-ui/components/layout/root-toggle";
import { DocsLayout } from "fumadocs-ui/layout";
import { Code, Component, Handshake, Terminal } from "lucide-react";
import type { ReactNode } from "react";
import { baseOptions } from "../layout.config";

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <DocsLayout
      tree={source.pageTree}
      {...baseOptions}
      sidebar={{
        banner: (
          <RootToggle
            options={[
              {
                title: "Contributing",
                description: "Create your first PR",
                url: "/docs/contributing",
                icon: <Code className="size-4 text-blue-600 dark:text-blue-400" />,
              },
              {
                title: "Company",
                description: "How we work",
                url: "/docs/company",
                icon: <Handshake className="size-4 text-amber-600 dark:text-amber-400" />,
              },
              {
                title: "API Design",
                description: "Look and feel",
                url: "/docs/api-design",
                icon: <Terminal className="size-4 text-emerald-600 dark:text-emerald-400" />,
              },
              {
                title: "Architecture",
                description: "How does Unkey work",
                url: "/docs/architecture",
                icon: <Component className="size-4 text-purple-600 dark:text-purple-400" />,
              },
            ]}
          />
        ),
      }}
    >
      {children}
    </DocsLayout>
  );
}
