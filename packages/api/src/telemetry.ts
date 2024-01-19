import { version } from "../package.json";
import { UnkeyOptions } from "./client";

export type Telemetry = {
  /**
   * Unkey-Telemetry-Sdk
   * @example @unkey/api@v1.1.1
   */
  sdkVersions: string[];
  /**
   * Unkey-Telemetry-Platform
   * @example cloudflare
   */
  platform: string;
  /**
   * Unkey-Telemetry-Runtime
   * @example node@v18
   */
  runtime: string;
};

export function getTelemetry(opts: UnkeyOptions) {
  let platform = "unknown";
  let runtime = "unknown";
  const sdkVersions = [`@unkey/api@${version}`];

  try {
    if (typeof process !== "undefined") {
      if (process.env.UNKEY_DISABLE_TELEMETRY) {
        return;
      }
      platform = process.env.VERCEL ? "vercel" : process.env.AWS_REGION ? "aws" : "unknown";

      // @ts-ignore
      if (typeof EdgeRuntime !== "undefined") {
        runtime = "edge-light";
      } else {
        runtime = `node@${process.version}`;
      }
    }

    if (opts.wrapperSdkVersion) {
      sdkVersions.push(opts.wrapperSdkVersion);
    }
  } catch (_error) {}

  return { platform, runtime, sdkVersions };
}
