import { z } from "zod";

export const emptyHeader = { name: "", value: "" };

const headerNamePattern = /^[!#$%&'*+.^_`|~0-9A-Za-z-]+$/;
function isValidHttpHeaderValue(value: string): boolean {
  for (const character of value) {
    const codePoint = character.codePointAt(0);
    if (codePoint === undefined || (codePoint < 32 && codePoint !== 9) || codePoint === 127) {
      return false;
    }
  }
  return true;
}

const headerFieldSchema = z.object({
  name: z.string().max(256, "Header name must be 256 characters or less"),
  value: z
    .string()
    .max(8192, "Header value must be 8,192 characters or less")
    .refine(isValidHttpHeaderValue, "Enter a valid header value"),
});

type HeaderForValidation = z.infer<typeof headerFieldSchema>;

function validateHeaderFields(headers: HeaderForValidation[], context: z.RefinementCtx) {
  const names = new Set<string>();

  for (const [index, header] of headers.entries()) {
    const name = header.name.trim();
    const hasName = name !== "";
    const hasValue = header.value !== "";

    if (!hasName && !hasValue) {
      continue;
    }
    if (!hasName) {
      context.addIssue({
        code: "custom",
        path: [index, "name"],
        message: "Enter a header name",
      });
      continue;
    }
    if (!headerNamePattern.test(name)) {
      context.addIssue({
        code: "custom",
        path: [index, "name"],
        message: "Enter a valid header name",
      });
    }
    const normalizedName = name.toLowerCase();
    if (!hasValue) {
      context.addIssue({
        code: "custom",
        path: [index, "value"],
        message: "Enter a header value",
      });
    }

    if (names.has(normalizedName)) {
      context.addIssue({
        code: "custom",
        path: [index, "name"],
        message: "Header name is duplicated",
      });
    }
    names.add(normalizedName);
  }
}

export const headerFieldsSchema = z
  .array(headerFieldSchema)
  .min(1)
  .max(32, "A maximum of 32 headers is supported")
  .superRefine(validateHeaderFields);

export type HeaderField = z.infer<typeof headerFieldsSchema>[number];

export function toHeaderRecord(headers: HeaderField[]): Record<string, string> {
  return Object.fromEntries(
    headers
      .filter((header) => header.name.trim() !== "" || header.value !== "")
      .map((header) => [header.name.trim(), header.value]),
  );
}
