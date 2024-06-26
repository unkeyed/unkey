import { z } from "zod";

export const protectedApiRequestSchema = z.object({
  url: z.string(),
  method: z.string(),
  jsonBody: z.string().optional(),
});
