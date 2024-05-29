import { cors, init } from "@/pkg/middleware";
import { ConsoleLogger } from "@unkey/worker-logging";
import { OpenAI } from "openai";
import { handleNonStreamingRequest, handleStreamingRequest } from "./index";
import { type Env, zEnv } from "./pkg/env";
import { newApp } from "./pkg/hono/app";

const app = newApp();

app.use("*", init());
app.use("*", cors());

app.all("*", async (c) => {
  const url = new URL(c.req.url);
  console.log(url, url.hostname, c.env.APEX_DOMAIN);
  const subdomain = url.hostname.replace(`.${c.env.APEX_DOMAIN}`, "");
  if (!subdomain) {
    console.log("no subdomain");
    return c.notFound();
  }

  console.log({ subdomain });

  const bearer = c.req.header("Authorization");
  if (!bearer) {
    return new Response("No API key", { status: 401 });
  }
  const apiKey = bearer.replace("Bearer ", "");
  const openai = new OpenAI({
    apiKey,
  });
  const request = await c.req.json();

  const { db } = c.get("services");

  const gw = await db.query.llmGateways.findFirst({
    where: (table, { eq }) => eq(table.subdomain, subdomain),
  });
  if (!gw) {
    return c.text("No gateway found", { status: 404 });
  }

  console.info("running");
  console.info("request", c.req.url);

  if (request.stream) {
    return handleStreamingRequest(c, request, openai);
  }
  return handleNonStreamingRequest(c, request, openai);
});

const handler = {
  fetch: (req: Request, rawEnv: Env, executionCtx: ExecutionContext) => {
    const parsedEnv = zEnv.safeParse(rawEnv);
    if (!parsedEnv.success) {
      new ConsoleLogger({ requestId: "" }).fatal(`BAD_ENVIRONMENT: ${parsedEnv.error.message}`);
      return Response.json(
        {
          code: "BAD_ENVIRONMENT",
          message: "Some environment variables are missing or are invalid",
          errors: parsedEnv.error,
        },
        { status: 500 },
      );
    }
    return app.fetch(req, parsedEnv.data, executionCtx);
  },
} satisfies ExportedHandler<Env>;

export default handler;
