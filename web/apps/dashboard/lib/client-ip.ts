import { z } from "zod";

const ipSchema = z.union([z.ipv4(), z.ipv6()]);

/**
 * Header precedence, most trustworthy first.
 *
 * On Vercel all three are set by the platform from the connecting IP and carry the same value, so
 * the order only decides anything on a deployment Vercel does not front. There, no forwarding
 * header is authoritative on its own, and precedence is a judgement call rather than a guarantee:
 * `x-vercel-forwarded-for` leads because nothing but Vercel has cause to send it, and
 * `x-forwarded-for` outranks `x-real-ip` because it is the header a reverse proxy conventionally
 * rewrites, and the one the dashboard already preferred before this list existed.
 *
 * Precedence is the weaker half of the defense. What actually stops a client from choosing its own
 * address is that the value must parse as an IP, which `getClientIp` enforces below. A deployment
 * that genuinely sits behind a proxy needs a trusted-hop count to be correct, not a header ranking.
 */
const IP_HEADERS = ["x-vercel-forwarded-for", "x-forwarded-for", "x-real-ip"] as const;

/** An IPv6 literal wrapped in brackets, with an optional port: `[::1]:8080` -> `::1`. */
const BRACKETED_IPV6 = /^\[(.+?)\](?::\d+)?$/;

/** An address carrying a port, where the address itself holds no colon: `1.2.3.4:8080` -> `1.2.3.4`. */
const ADDRESS_WITH_PORT = /^([^:]+):\d+$/;

/**
 * Removes the port, and the brackets an IPv6 literal is wrapped in when it carries one.
 *
 * A bare IPv6 address is returned untouched, because its colons separate groups rather than a port
 * and only the bracketed form can express one unambiguously.
 */
function stripPort(value: string): string {
  const bracketed = value.match(BRACKETED_IPV6);
  if (bracketed) {
    return bracketed[1];
  }

  const withPort = value.match(ADDRESS_WITH_PORT);
  if (withPort) {
    return withPort[1];
  }

  return value;
}

/**
 * Resolves the IP of the client that made the request.
 *
 * Only the leftmost entry of a forwarding chain is considered, and it must parse as an IP address.
 * Anything else is discarded rather than passed through, so a client that injects a value into a
 * forwarding header cannot get arbitrary text recorded as its address.
 */
export function getClientIp(headers: Pick<Headers, "get">): string | undefined {
  for (const header of IP_HEADERS) {
    const value = headers.get(header);
    if (!value) {
      continue;
    }

    const parsed = ipSchema.safeParse(stripPort(value.split(",")[0].trim()));
    if (parsed.success) {
      return parsed.data;
    }
  }

  return undefined;
}
