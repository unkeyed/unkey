import config from "./embed/gateway";

export default {
  async fetch(request: Request): Promise<Response> {
    console.log("xxx");
    const url = new URL(request.url);

    const origin = new URL(url.pathname, config.origin);
    console.log("origin", origin);

    const res = fetch(origin);
    // @ts-expect-error
    return res;
  },
};
