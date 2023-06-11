import baseX from "base-x";

const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";
export function toBase58(buf: Uint8Array): string {
  return baseX(alphabet).encode(buf);
}
