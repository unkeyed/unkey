import { sha256 } from "@unkey/hash";
import { KeyV1 } from "./v1";

export async function newKey(opts: {
  prefix?: string;
  byteLength: number;
}): Promise<{
  key: string;
  hash: string;
  prefix: string;
  start: string;
  end: string;
}> {
  const key = new KeyV1({
    byteLength: opts.byteLength,
    prefix: opts.prefix,
  }).toString();
  const randomStart = opts.prefix ? opts.prefix.length + 1 : 0;
  const start = key.slice(randomStart, randomStart + 4);
  const end = key.slice(-4);
  const hash = await sha256(key);

  return { key, hash, prefix: opts.prefix ?? "", start, end };
}
