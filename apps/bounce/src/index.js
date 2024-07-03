/**
 * Welcome to Cloudflare Workers! This is your first worker.
 *
 * - Run `npm run dev` in your terminal to start a development server
 * - Open a browser tab at http://localhost:8787/ to see your worker in action
 * - Run `npm run deploy` to publish your worker
 *
 * Learn more at https://developers.cloudflare.com/workers/
 */

const ids = new Set();

function getIdentifier() {
  if (ids.size === 0 || Math.random() > 0.9) {
    ids.add(Math.random().toString().slice(0, 4));
  }

  return Array.from(ids.values())[Math.floor(Math.random() * ids.size)];
}

export default {
  async fetch(request) {
    const pathname = new URL(request.url).pathname;

    const identifier = getIdentifier();

    const url = `https://api.unkey.cloud${pathname}`;
    const res = await fetch(url, {
      method: request.method,
      headers: request.headers,
      body: JSON.stringify({
        identifier,
        limit: 10000,
        duration: 60000,
      }),
    });

    const body = await res.text();
    if (res.status !== 200) {
      console.log("response", res.status, body);
    }

    return new Response(body, {
      headers: res.headers,
      status: res.status,
    });
  },
};
