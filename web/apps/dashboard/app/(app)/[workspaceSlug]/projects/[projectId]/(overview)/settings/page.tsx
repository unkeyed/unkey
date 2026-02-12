"use client";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { parseAsString, useQueryState } from "nuqs";
import { useProjectData } from "../data-provider";
import { BuildSettings } from "./components/build-settings";
import { GitHubSettingsClient } from "./components/github-settings-client";
import { RuntimeApplicationSettings } from "./components/runtime-application-settings";
import { RuntimeScalingSettings } from "./components/runtime-scaling-settings";

export default function SettingsPage() {
  const { environments } = useProjectData();
  const [environmentId, setEnvironmentId] = useQueryState(
    "environmentId",
    parseAsString.withDefault(environments.length > 0 ? environments[0].id : "").withOptions({
      history: "replace",
      shallow: true,
    }),
  );

  return (
    <div className="py-3 w-full flex items-center justify-center">
      <div className="w-[900px] flex flex-col justify-center items-center gap-5 mx-6">
        <div className="w-full text-accent-12 font-semibold text-lg py-6 text-left border-b border-gray-4">
          Project Settings
        </div>
        <div className="flex flex-col w-full gap-6">
          <section>
            <h2 className="text-accent-12 font-medium text-base mb-3">Source</h2>
            <GitHubSettingsClient />
          </section>
          <div className="w-full border-b border-gray-4" />
          <div className="w-full">
            <h2 className="text-accent-12 font-medium text-base mb-3 block">Environment</h2>
            <Select value={environmentId ?? undefined} onValueChange={setEnvironmentId}>
              <SelectTrigger>
                <SelectValue placeholder="Select environment" />
              </SelectTrigger>
              <SelectContent>
                {environments?.map((env) => (
                  <SelectItem key={env.id} value={env.id}>
                    {env.slug}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          {environmentId !== null && (
            <div key={environmentId} className="flex flex-col w-full gap-6">
              <section>
                <h3 className="text-accent-12 font-medium text-base mb-3">Build</h3>
                <BuildSettings environmentId={environmentId} />
              </section>
              <section>
                <h3 className="text-accent-12 font-medium text-base mb-3">Runtime</h3>
                <RuntimeApplicationSettings environmentId={environmentId} />
              </section>
              <section>
                <h3 className="text-accent-12 font-medium text-base mb-3">Scaling</h3>
                <RuntimeScalingSettings environmentId={environmentId} />
              </section>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
