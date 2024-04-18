import { zEnv, type Env } from "./env";
import type { Request } from "@cloudflare/workers-types/experimental";

export default {
  async fetch(req: Request, rawEnv: Env, _ctx: ExecutionContext): Promise<Response> {
    console.info("req");
    try {
      const env = zEnv.parse(rawEnv);

      const url = new URL(req.url);
      const host = url.hostname;
      console.info({ host });

      if (host === "fallback.unkey.ui") {
        return Response.json({
          message: "fallback",
        });
      }

      const worker = env.DISPATCH.get(`customer_gateway::${host}`);
      return worker.fetch(req);
    } catch (e) {
      const err = e as Error;

      console.error(err);
      return Response.json({
        error: err.message,
      });
    }
  },
};
