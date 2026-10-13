import type { SettingsPage } from "../settings-page";
import { Command } from "./command";
import { Healthcheck } from "./healthcheck";
import { Port } from "./port-settings";

export const RUNTIME_SETTINGS: SettingsPage = {
  title: "Runtime",
  description: "Port, start command and health check. The defaults suit most apps.",
  summary: ({ port, command, healthcheck }) =>
    [
      `Port ${port}`,
      command.length > 0 ? "Custom command" : "Image command",
      healthcheck ? `${healthcheck.method} ${healthcheck.path}` : "No health check",
    ].join(" · "),
  groups: [
    {
      title: "Process",
      rows: (
        <>
          <Port />
          <Command />
        </>
      ),
    },
    { title: "Health check", rows: <Healthcheck /> },
  ],
};
