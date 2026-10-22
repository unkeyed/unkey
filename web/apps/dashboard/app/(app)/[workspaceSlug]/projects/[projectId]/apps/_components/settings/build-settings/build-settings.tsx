"use client";

import { match } from "@unkey/match";
import { SettingsForm, SettingsGroup, SettingsGroupContent } from "@unkey/ui";
import { useBuildRecipe } from "../hooks/use-build-recipe";
import { useBuildSource } from "../hooks/use-build-source";
import { OCIImage } from "../oci-image";
import { BuildCommand } from "./build-command-settings";
import { Dockerfile } from "./dockerfile-settings";
import { GitHub } from "./github-settings";
import { RootDirectory } from "./root-directory-settings";
import { WatchPaths } from "./watch-paths-settings";

const SWITCH_TO_GITHUB_DESCRIPTION =
  "Connecting a repository switches this app from the image to GitHub builds.";

export function BuildSettings({ githubReadOnly = false }: { githubReadOnly?: boolean }) {
  const { app, hasRepository } = useBuildSource();

  return (
    <SettingsGroup>
      <SettingsGroupContent>
        {app
          ? match(app.sourceType)
              .with("oci", () => (
                <>
                  <OCIImage appId={app.id} imageReference={app.imageReference ?? ""} />
                  {githubReadOnly ? null : <GitHub description={SWITCH_TO_GITHUB_DESCRIPTION} />}
                </>
              ))
              .with("git", "unknown", () => <GitHub readOnly={githubReadOnly} />)
              .exhaustive()
          : null}
        {hasRepository ? <GitBuildRows /> : null}
      </SettingsGroupContent>
    </SettingsGroup>
  );
}

function GitBuildRows() {
  const { form, formProps } = useBuildRecipe();
  return (
    <>
      <SettingsForm {...formProps}>
        <RootDirectory form={form} />
        <Dockerfile form={form} />
        <BuildCommand form={form} />
      </SettingsForm>
      <WatchPaths />
    </>
  );
}
