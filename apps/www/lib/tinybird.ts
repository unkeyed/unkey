import { NoopTinybird, Tinybird } from "@chronark/zod-bird";
import { z } from "zod";
import { env } from "./env";

const token = env().TINYBIRD_TOKEN;
const tb = token ? new Tinybird({ token }) : new NoopTinybird();

export const getTotalVerifications = tb.buildPipe({
  pipe: "endpoint__all_verifications__v1",
  data: z.object({ verifications: z.number() }),
});
