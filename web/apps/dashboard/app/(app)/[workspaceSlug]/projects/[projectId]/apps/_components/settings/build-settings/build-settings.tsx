"use client";

import { match } from "@unkey/match";
import { SettingsGroup, SettingsGroupContent } from "@unkey/ui";
import { useBuildSource } from "../hooks/use-build-source";
import { OCIImage } from "../oci-image";
import { BuildCommand } from "./build-command-settings";
import { Dockerfile } from "./dockerfile-settings";
import { GitHub } from "./github-settings";
import { RootDirectory } from "./root-directory-settings";
import { WatchPaths } from "./watch-paths-settings";

export function BuildSettings({ githubReadOnly = false }: { githubReadOnly?: boolean }) {
  const { app, hasRepository } = useBuildSource();

  return (
    <SettingsGroup>
      <SettingsGroupContent>
        {app
          ? match(app.sourceType)
              .with("oci", () => (
                <OCIImage appId={app.id} imageReference={app.imageReference ?? ""} />
              ))
              .with("git", "unknown", () => <GitHub readOnly={githubReadOnly} />)
              .exhaustive()
          : null}
        {hasRepository ? (
          <>
            <RootDirectory />
            <Dockerfile />
            <BuildCommand />
            <WatchPaths />
          </>
        ) : null}
      </SettingsGroupContent>
    </SettingsGroup>
  );
}
