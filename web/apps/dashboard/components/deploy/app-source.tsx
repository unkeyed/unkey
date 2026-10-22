import type { App } from "@/lib/collections/deploy/apps";
import { Github, IconLayers2Outline18, IconTerminalOutline18 } from "@unkey/icons";
import { match } from "@unkey/match";

export type AppSource = "git" | "image" | "legacy";

export function appSource(app: Pick<App, "sourceType" | "repositoryFullName">): AppSource {
  return match(app.sourceType)
    .returnType<AppSource>()
    .with("git", () => "git")
    .with("oci", () => "image")
    .with("unknown", () => (app.repositoryFullName ? "git" : "legacy"))
    .exhaustive();
}

export function SourceIcon({ source, className }: { source: AppSource; className?: string }) {
  return match(source)
    .with("git", () => <Github className={className} />)
    .with("image", () => <IconLayers2Outline18 className={className} />)
    .with("legacy", () => <IconTerminalOutline18 className={className} />)
    .exhaustive();
}
