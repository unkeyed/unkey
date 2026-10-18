import { z } from "zod";
import { useSettingField } from "./use-setting-form";

export function useBuildFields() {
  const dockerfile = useSettingField("dockerfile", z.string());
  const buildCommand = useSettingField("buildCommand", z.string().max(1000));
  return {
    dockerfile,
    buildCommand,
    dockerfileConfigured: dockerfile.form.watch("dockerfile").trim() !== "",
    onDockerfileSelect: (path: string) => {
      if (path.trim() !== "") {
        buildCommand.form.resetField("buildCommand");
      }
    },
  };
}
