import { create, fromBinary, toBinary } from "@bufbuild/protobuf";
import { ConfigSchema, HttpBodyFormat } from "@/gen/proto/logdrain/v1/config_pb";
import { VaultService } from "@/gen/proto/vault/v1/service_pb";
import { createVaultClient } from "@/lib/vault-client";

/** EncryptedHttpHeader stores one HTTP header name and its Vault ciphertext. */
export type EncryptedHttpHeader = {
  name: string;
  encryptedValue: string;
};

/** LogdrainConfig is the typed dashboard representation of the stored provider config. */
export type LogdrainConfig =
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
    };

/** encodeLogdrainConfig encodes the complete provider configuration as protobuf. */
export function encodeLogdrainConfig(config: LogdrainConfig): Buffer {
  switch (config.kind) {
    case "http":
      return Buffer.from(
        toBinary(
          ConfigSchema,
          create(ConfigSchema, {
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
  const { destination } = fromBinary(ConfigSchema, raw);
  switch (destination.case) {
    case "http":
      return {
        kind: destination.case,
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
