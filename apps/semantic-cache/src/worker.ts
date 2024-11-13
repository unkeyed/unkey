import { cors, init, ratelimit } from "@/pkg/middleware";
import { ConsoleLogger } from "@unkey/worker-logging";
import { OpenAI } from "openai";
import { type Env, zEnv } from "./pkg/env";
import { newApp } from "./pkg/hono/app";
import { handleNonStreamingRequest, handleStreamingRequest } from "./pkg/streaming";

const app = newApp();

app.use("*", init());
app.use("*", ratelimit());
app.use("*", cors());

app.all("*", async (c) => {
  const url = new URL(c.req.url);
  let subdomain = url.hostname.replace(`.${c.env.APEX_DOMAIN}`, "");
  if (subdomain === url.hostname || (subdomain === "" && c.env.FALLBACK_SUBDOMAIN)) {
    subdomain = c.env.FALLBACK_SUBDOMAIN!;
  }
  if (!subdomain) {
    return c.notFound();
  }

  const bearer = c.req.header("Authorization");
  if (!bearer) {
    return new Response("No API key", { status: 401 });
  }
  const apiKey = bearer.replace("Bearer ", "");
  const openai = new OpenAI({
    apiKey,
    baseURL: c.req.header("X-Base-Url"),
  });
  const request = (await c.req.json()) as OpenAI.Chat.Completions.ChatCompletionCreateParams;
  const { db } = c.get("services");

  const gw = await db.query.llmGateways.findFirst({
    where: (table, { eq }) => eq(table.subdomain, subdomain),
  });
  if (!gw) {
    return c.text("No gateway found", { status: 404 });
  }

  if (request.stream) {
    return await handleStreamingRequest(c, request, openai);
  }
  return await handleNonStreamingRequest(c, request, openai);
});

const handler = {
  fetch: (req: Request, rawEnv: Env, executionCtx: ExecutionContext) => {
    const parsedEnv = zEnv.safeParse(rawEnv);
    if (!parsedEnv.success) {
      new ConsoleLogger({
        requestId: "",
        environment: rawEnv.ENVIRONMENT,
        application: "semantic-cache",
      }).fatal(`BAD_ENVIRONMENT: ${parsedEnv.error.message}`);
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
