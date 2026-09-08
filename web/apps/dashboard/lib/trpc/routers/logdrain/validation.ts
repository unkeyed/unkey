import { z } from "zod";

/** Matches the RFC 9110 token characters that are valid in an HTTP field name. */
const httpHeaderNamePattern = /^[!#$%&'*+.^_`|~0-9A-Za-z-]+$/;

function isValidHttpHeaderValue(value: string): boolean {
  for (const character of value) {
    const codePoint = character.codePointAt(0);
    if (codePoint === undefined || (codePoint < 32 && codePoint !== 9) || codePoint === 127) {
      return false;
    }
  }
  return true;
}

/** httpsUrl accepts destination URLs that use HTTPS and omit credentials. */
export const httpsUrl = z
  .string()
  .url()
  .refine((url) => new URL(url).protocol === "https:", {
    message: "URL must use HTTPS",
  })
  .refine((url) => new URL(url).username === "" && new URL(url).password === "", {
    message: "URL must not contain credentials",
  });

/** httpFormatSchema defines the supported HTTP request body formats. */
export const httpFormatSchema = z.enum(["json", "ndjson"]);

/** httpHeadersSchema validates bounded HTTP header records at the API boundary. */
export const httpHeadersSchema = z
  .record(
    z.string(),
    z.string().min(1).max(8192).refine(isValidHttpHeaderValue, "Invalid header value"),
  )
  .refine((headers) => Object.keys(headers).length <= 32, "A maximum of 32 headers is supported")
  .superRefine((headers, context) => {
    const names = new Set<string>();
    for (const name of Object.keys(headers)) {
      if (name.length === 0 || name.length > 256 || !httpHeaderNamePattern.test(name)) {
        context.addIssue({
          code: "custom",
          path: [name],
          message: "Invalid header name",
        });
      }
      const normalizedName = name.toLowerCase();
      if (names.has(normalizedName)) {
        context.addIssue({
          code: "custom",
          path: [name],
          message: "Header name is duplicated",
        });
      }
      names.add(normalizedName);
    }
  });

const httpHeaderUpdateSchema = z.discriminatedUnion("mode", [
  z.object({
    mode: z.literal("preserve"),
    name: z.string(),
  }),
  z.object({
    mode: z.literal("set"),
    name: z.string(),
    value: z
      .string()
      .min(1, "Invalid header value")
      .max(8192)
      .refine(isValidHttpHeaderValue, "Invalid header value"),
  }),
]);

/** httpHeaderUpdatesSchema describes the desired header set without exposing stored values. */
export const httpHeaderUpdatesSchema = z
  .array(httpHeaderUpdateSchema)
  .max(32, "A maximum of 32 headers is supported")
  .superRefine((headers, context) => {
    const names = new Set<string>();
    for (const [index, header] of headers.entries()) {
      if (
        header.name.length === 0 ||
        header.name.length > 256 ||
        !httpHeaderNamePattern.test(header.name)
      ) {
        context.addIssue({
          code: "custom",
          path: [index, "name"],
          message: "Invalid header name",
        });
      }
      const normalizedName = header.name.toLowerCase();
      if (names.has(normalizedName)) {
        context.addIssue({
          code: "custom",
          path: [index, "name"],
          message: "Header name is duplicated",
        });
      }
      names.add(normalizedName);
    }
  });

/** One write-only HTTP header update. */
export type HttpHeaderUpdate = z.infer<typeof httpHeaderUpdateSchema>;
