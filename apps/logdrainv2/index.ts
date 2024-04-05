Bun.serve({
  port: 8000,
  async fetch(req) {
    const body = await req.text();

    console.log(req.url, JSON.stringify(req.headers), body);

    return new Response("ok");
  },
});
