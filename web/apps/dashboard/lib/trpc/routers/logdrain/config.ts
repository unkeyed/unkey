import { ConfigSchema, HttpBodyFormat } from "@/gen/proto/logdrain/v1/config_pb";
import { VaultService } from "@/gen/proto/vault/v1/service_pb";
import { createVaultClient } from "@/lib/vault-client";
import { create, fromBinary, toBinary } from "@bufbuild/protobuf";

/** EncryptedHttpHeader stores one HTTP header name and its Vault ciphertext. */
export type EncryptedHttpHeader = {
  name: string;
  encryptedValue: string;
};

/** LogdrainConfig is the typed dashboard representation of the stored provider config. */
export type LogdrainConfig = {
  stream:
    | { kind: "audit_logs"; eventTypes: string[] }
    | { kind: "key_verifications"; outcomes: string[]; keySpaceIds: string[] }
    | {
        kind: "gateway_requests";
        statusClasses: number[];
        projectIds: string[];
        appIds: string[];
        environmentIds: string[];
      };
} & (
  | {
      kind: "http";
      url: string;
      format: "json" | "ndjson";
      headers: EncryptedHttpHeader[];
    }
  | {
      kind: "axiom";
      dataset: string;
      encryptedToken: string;
    }
);

/** encodeLogdrainConfig encodes the complete provider configuration as protobuf. */
export function encodeLogdrainConfig(config: LogdrainConfig): Buffer {
  const stream = encodeStream(config.stream);
  switch (config.kind) {
    case "http":
      return Buffer.from(
        toBinary(
          ConfigSchema,
          create(ConfigSchema, {
            stream,
            destination: {
              case: config.kind,
              value: {
                url: config.url,
                format: config.format === "ndjson" ? HttpBodyFormat.NDJSON : HttpBodyFormat.JSON,
                headers: config.headers,
              },
            },
          }),
        ),
      );
    case "axiom":
      return Buffer.from(
        toBinary(
          ConfigSchema,
          create(ConfigSchema, {
            stream,
            destination: {
              case: config.kind,
              value: {
                dataset: config.dataset,
                encryptedToken: config.encryptedToken,
              },
            },
          }),
        ),
      );
    default:
      throw new Error(`Unsupported log drain sink: ${config satisfies never}`);
  }
}

/** decodeLogdrainConfig decodes a stored provider configuration. */
export function decodeLogdrainConfig(raw: Uint8Array): LogdrainConfig {
  const config = fromBinary(ConfigSchema, raw);
  const { destination } = config;
  let stream: LogdrainConfig["stream"];
  switch (config.stream.case) {
    case "gatewayRequests":
      stream = {
        kind: "gateway_requests",
        statusClasses: config.stream.value.statusClasses,
        projectIds: config.stream.value.projectIds,
        appIds: config.stream.value.appIds,
        environmentIds: config.stream.value.environmentIds,
      };
      break;
    case "keyVerifications":
      stream = {
        kind: "key_verifications",
        outcomes: config.stream.value.outcomes,
        keySpaceIds: config.stream.value.keySpaceIds,
      };
      break;
    case "auditLogs":
      stream = {
        kind: "audit_logs",
        eventTypes: config.stream.value.eventTypes,
      };
      break;
    case undefined:
      stream = { kind: "audit_logs", eventTypes: [] };
      break;
    default:
      throw new Error(`Unsupported log drain stream: ${config.stream satisfies never}`);
  }
  switch (destination.case) {
    case "http":
      return {
        kind: destination.case,
        stream,
        url: destination.value.url,
        format: decodeHttpFormat(destination.value.format),
        headers: destination.value.headers.map(({ name, encryptedValue }) => ({
          name,
          encryptedValue,
        })),
      };
    case "axiom":
      return {
        kind: destination.case,
        stream,
        dataset: destination.value.dataset,
        encryptedToken: destination.value.encryptedToken,
      };
    case undefined:
      throw new Error("Log drain provider is not set");
    default:
      throw new Error(`Unsupported log drain sink: ${destination satisfies never}`);
  }
}

/** encryptHttpHeaders encrypts each value and sorts headers by name. */
export async function encryptHttpHeaders(
  workspaceId: string,
  headers: Record<string, string>,
): Promise<EncryptedHttpHeader[]> {
  const names = Object.keys(headers).sort();
  if (names.length === 0) {
    return [];
  }
  const response = await createVaultClient(VaultService).encryptBulk({
    keyring: workspaceId,
    items: headers,
  });
  return names.map((name) => {
    const item = response.items[name];
    if (!item?.encrypted) {
      throw new Error(`Vault did not encrypt HTTP header ${name}`);
    }
    return { name, encryptedValue: item.encrypted };
  });
}

function encodeStream(stream: LogdrainConfig["stream"]) {
  switch (stream.kind) {
    case "audit_logs":
      return { case: "auditLogs" as const, value: { eventTypes: stream.eventTypes } };
    case "key_verifications":
      return {
        case: "keyVerifications" as const,
        value: { outcomes: stream.outcomes, keySpaceIds: stream.keySpaceIds },
      };
    case "gateway_requests":
      return {
        case: "gatewayRequests" as const,
        value: {
          statusClasses: stream.statusClasses,
          projectIds: stream.projectIds,
          appIds: stream.appIds,
          environmentIds: stream.environmentIds,
        },
      };
    default:
      throw new Error(`Unsupported log drain stream: ${stream satisfies never}`);
  }
}

function decodeHttpFormat(format: HttpBodyFormat): "json" | "ndjson" {
  switch (format) {
    case HttpBodyFormat.UNSPECIFIED:
    case HttpBodyFormat.JSON:
      return "json";
    case HttpBodyFormat.NDJSON:
      return "ndjson";
    default:
      throw new Error(`Unknown HTTP body format ${format}`);
  }
}
