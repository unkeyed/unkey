Bun.serve({
  port: 8000,
  async fetch(req) {
    const body = await req.text();

    console.log(JSON.stringify(req.headers));

    const buf = Buffer.from(body, "utf-8");
    console.log("base64", buf.toString("base64"));
    const data = Bun.gunzipSync(buf);
    console.log(data);

    return new Response("ok");
  },
});
