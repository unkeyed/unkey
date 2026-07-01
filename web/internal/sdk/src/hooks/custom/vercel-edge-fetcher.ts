import { HTTPClient, type Fetcher } from "../../lib/http.js";
import type { SDKInitHook, SDKInitOptions } from "../types.js";

function isVercelEdgeRuntime(): boolean {
  // https://vercel.com/docs/functions/runtimes/edge#check-if-you're-running-on-the-edge-runtime
  if ("EdgeRuntime" in globalThis) return true;
  if (process.env["NEXT_RUNTIME"] === "edge") return true;
  return false;
}

const vercelEdgeFetcher: Fetcher = (input, init?) => {
  // Edge runtime fix: Request objects may not be recognized by instanceof
  // Check if it's a Request-like object by checking for .url property
  const isRequestLike =
    typeof input === "object" &&
    input !== null &&
    "url" in input &&
    "method" in input &&
    "headers" in input;

  if (isRequestLike && !init) {
    // For Edge runtime: extract URL and reconstruct request
    const req = input as Request;
    return fetch(req.url, {
      method: req.method,
      headers: req.headers,
      body: req.body,
      mode: req.mode,
      credentials: req.credentials,
      cache: req.cache,
      redirect: req.redirect,
      referrer: req.referrer,
      integrity: req.integrity,
      signal: req.signal,
    });
  }

  // If input is a Request and init is undefined, Bun will discard the method,
  // headers, body and other options that were set on the request object.
  // Node.js and browers would ignore an undefined init value. This check is
  // therefore needed for interop with Bun.
  if (init == null) {
    return fetch(input);
  } else {
    return fetch(input, init);
  }
};

export class FetcherOverrideForVercelEdgeHook implements SDKInitHook {
  sdkInit(opts: SDKInitOptions) {
    if (!isVercelEdgeRuntime()) {
      return opts;
    }

    const client = new HTTPClient({
      fetcher: vercelEdgeFetcher,
    });

    return {
      ...opts,
      client,
    };
  }
}
