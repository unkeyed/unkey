import { decompressSync, strFromU8, strToU8 } from "fflate";

Bun.serve({
  port: 8000,
  async fetch(req) {
    const b = await req.blob();

    const buf = await b.arrayBuffer();

    const dec = decompressSync(new Uint8Array(buf));
    const data = strFromU8(dec);

    console.log(data);

    return new Response("ok");
  },
});
