import type { SettingsPage } from "../settings-page";
import { OpenapiSpecPath } from "./openapi-spec-path";
import { PROTOCOLS, UpstreamProtocol } from "./upstream-protocol";

export const ADVANCED_SETTINGS: SettingsPage = {
  title: "Advanced",
  description: "OpenAPI spec path and upstream protocol.",
  summary: ({ upstreamProtocol, openapiSpecPath }) =>
    `${PROTOCOLS[upstreamProtocol].label} · ${openapiSpecPath?.trim() || "No OpenAPI spec"}`,
  groups: [
    { title: "API", rows: <OpenapiSpecPath /> },
    { title: "Networking", rows: <UpstreamProtocol /> },
  ],
};
