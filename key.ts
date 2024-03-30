import { base64 } from "./internal/encoding/src";

const key = await crypto.subtle.generateKey(
  {
    name: "AES-GCM",
    length: 256,
  },
  true,
  ["encrypt", "decrypt"],
);

const exportedKey = await crypto.subtle.exportKey("raw", key);

console.log(base64.encode(exportedKey));
