import { z } from "zod";

const MAX_DOMAIN_LENGTH = 253;
const MAX_LABEL_LENGTH = 63;
const FORMAT_ERROR = "Enter a domain like api.example.com";

const LABEL = /^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$/;
const IP_LIKE_LABEL = /^(?:[0-9]+|0x[0-9a-f]+)$/;
const NOT_A_HOSTNAME = /[\s*:/?#@%\\[\]]/;

type ParsedDomain = { ok: true; domain: string } | { ok: false; error: string };

/**
 * Mirrors `domaingate.ParseDomain` in pkg/domain/domaingate and returns the
 * same canonical form: lowercase ASCII with Unicode labels Punycode encoded.
 * The public-suffix check stays on the server, which holds the suffix list.
 */
export function parseDomain(input: string): ParsedDomain {
  const hostname = toAscii(input);
  if (hostname === null || hostname.endsWith(".")) {
    return { ok: false, error: FORMAT_ERROR };
  }
  const labels = hostname.split(".");
  if (labels.some((label) => label.length > MAX_LABEL_LENGTH)) {
    return {
      ok: false,
      error: `Each part between dots must be at most ${MAX_LABEL_LENGTH} characters`,
    };
  }
  if (hostname.length > MAX_DOMAIN_LENGTH) {
    return { ok: false, error: `Domain must be at most ${MAX_DOMAIN_LENGTH} characters` };
  }
  const tld = labels[labels.length - 1];
  if (
    labels.length < 2 ||
    !labels.every(isHostnameLabel) ||
    tld.length < 2 ||
    IP_LIKE_LABEL.test(tld)
  ) {
    return { ok: false, error: FORMAT_ERROR };
  }
  return { ok: true, domain: hostname };
}

function toAscii(input: string): string | null {
  if (NOT_A_HOSTNAME.test(input)) {
    return null;
  }
  try {
    return new URL(`http://${input}`).hostname;
  } catch {
    return null;
  }
}

// IDNA reserves "--" in the third and fourth place for Punycode labels.
function isHostnameLabel(label: string): boolean {
  return LABEL.test(label) && (label.slice(2, 4) !== "--" || label.startsWith("xn--"));
}

export const customDomainSchema = z.object({
  environmentId: z.string().min(1, "Environment is required"),
  domain: z
    .string()
    .trim()
    .min(1, "Domain is required")
    .transform((value, ctx) => {
      const parsed = parseDomain(value);
      if (!parsed.ok) {
        ctx.addIssue({ code: "custom", message: parsed.error });
        return z.NEVER;
      }
      return parsed.domain;
    }),
});

export type CustomDomainFormValues = z.infer<typeof customDomainSchema>;
