/**
 * Validates Docker/OCI image references against the grammar registries enforce, so
 * a bad reference fails in the form rather than as a failed pull mid-deploy.
 *
 * Constants cite their counterpart in github.com/distribution/reference
 * (as reference/<file>) and github.com/opencontainers/go-digest
 */

/** reference/regexp.go domainNameComponent. */
const DOMAIN_NAME_COMPONENT = "(?:[a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9-]*[a-zA-Z0-9])";

/** reference/regexp.go ipv6address. */
const IPV6_ADDRESS = "\\[(?:[a-fA-F0-9:]+)\\]";

/** reference/regexp.go domainAndPort. */
const DOMAIN = new RegExp(
  `^(?:${DOMAIN_NAME_COMPONENT}(?:\\.${DOMAIN_NAME_COMPONENT})*|${IPV6_ADDRESS})(?::[0-9]+)?$`,
);

/** reference/regexp.go pathComponent. */
const PATH_COMPONENT = /^[a-z0-9]+(?:(?:[._]|__|[-]+)[a-z0-9]+)*$/;

/** reference/regexp.go tag, without its {0,127} bound. */
const TAG = /^\w[\w.-]*$/;

/**
 * Accepted by both reference/regexp.go digestPat and go-digest/digest.go
 * DigestRegexp: the first wants a leading letter, the second wants lowercase.
 */
const DIGEST = /^([a-z][a-z0-9]*(?:[-_+.][a-z][a-z0-9]*)*):([0-9a-fA-F]+)$/;

/** reference/reference.go RepositoryNameTotalLengthMax. */
const MAX_PATH_LENGTH = 255;

/** reference/regexp.go tag bound. */
const MAX_TAG_LENGTH = 128;

/** The public API contracts cap image references at 256 characters. */
const MAX_REFERENCE_LENGTH = 256;

/**
 * go-digest/algorithm.go anchoredEncodedRegexps. Any other algorithm parses but
 * fails to pull with "unsupported digest algorithm".
 */
const DIGEST_HEX_LENGTHS: Record<string, number> = { sha256: 64, sha384: 96, sha512: 128 };

/** reference/normalize.go splitDockerDomain canonicalizes these to Docker Hub. */
const DOCKER_HUB_DOMAINS = new Set(["docker.io", "index.docker.io"]);

/** A URL scheme, e.g. the `https://` on a reference copied out of a browser. */
const SCHEME = /^[a-z][a-z0-9+.-]*:\/\//i;

const SURROUNDING_QUOTES = /^["'`]+|["'`]+$/g;

export type ImageRefParts = {
  /** Absent when the reference resolves to Docker Hub. */
  domain?: string;
  path: string;
  tag?: string;
  digest?: string;
};

export type ImageRefValidation =
  | { ok: true; parts: ImageRefParts; warning?: string }
  | { ok: false; error: string };

/**
 * Removes what a reference picks up in transit and cannot legally contain: quotes
 * from a YAML file, a registry URL's scheme, surrounding space.
 *
 * The name is lowercased because registries store it that way, which makes a path
 * copied from a GitHub URL usable. Tags and digests keep their case: `:Latest` and
 * `:latest` are different tags. Anything with whitespace left in it is a command
 * line, not a reference, and validateImageRef rejects it.
 */
export const sanitizeImageRef = (input: string): string =>
  lowercaseName(input.trim().replace(SURROUNDING_QUOTES, "").replace(SCHEME, "").trim());

export const validateImageRef = (input: string): ImageRefValidation => {
  const ref = input.trim();

  if (!ref) {
    return { ok: false, error: "An image reference is required." };
  }
  if (/\s/.test(ref)) {
    return { ok: false, error: "An image reference cannot contain spaces." };
  }
  if (SCHEME.test(ref)) {
    return { ok: false, error: "Drop the protocol prefix, for example ghcr.io/acme/api:v1." };
  }
  if (ref.length > MAX_REFERENCE_LENGTH) {
    return {
      ok: false,
      error: `An image reference cannot be longer than ${MAX_REFERENCE_LENGTH} characters.`,
    };
  }

  const digestParts = ref.split("@");
  if (digestParts.length > 2) {
    return { ok: false, error: "An image reference can only contain one @." };
  }
  const [named, rawDigest] = digestParts;

  let digest: string | undefined;
  if (rawDigest !== undefined) {
    const match = DIGEST.exec(rawDigest);
    if (!match) {
      return { ok: false, error: "The digest must look like sha256:<hex>." };
    }
    const [, algorithm, hex] = match;
    const expectedLength = DIGEST_HEX_LENGTHS[algorithm];
    if (expectedLength === undefined) {
      return {
        ok: false,
        error: `"${algorithm}" digests cannot be pulled. Use sha256, sha384, or sha512.`,
      };
    }
    if (hex.length !== expectedLength) {
      return {
        ok: false,
        error: `A ${algorithm} digest needs ${expectedLength} hex characters, this one has ${hex.length}.`,
      };
    }
    if (hex !== hex.toLowerCase()) {
      return { ok: false, error: "A digest must be lowercase hex." };
    }
    digest = rawDigest;
  }

  const lastColon = named.lastIndexOf(":");
  const lastSlash = named.lastIndexOf("/");
  const hasTag = lastColon > lastSlash;
  const name = hasTag ? named.slice(0, lastColon) : named;
  const tag = hasTag ? named.slice(lastColon + 1) : undefined;

  if (tag !== undefined) {
    if (!tag) {
      return { ok: false, error: "The tag after : is empty." };
    }
    if (tag.length > MAX_TAG_LENGTH) {
      return { ok: false, error: `A tag cannot be longer than ${MAX_TAG_LENGTH} characters.` };
    }
    if (!TAG.test(tag)) {
      return {
        ok: false,
        error: "A tag can only contain letters, digits, and . _ - after its first character.",
      };
    }
  }

  if (!name) {
    return { ok: false, error: "The image name is missing." };
  }

  const components = name.split("/");
  if (components.some((component) => component === "")) {
    return { ok: false, error: "The image name has an empty path segment." };
  }

  const first = components[0];
  const isDomain =
    components.length > 1 && (first.includes(".") || first.includes(":") || first === "localhost");
  const domain = isDomain ? first : undefined;
  const pathComponents = isDomain ? components.slice(1) : components;
  const path = pathComponents.join("/");

  if (domain && !DOMAIN.test(domain)) {
    return { ok: false, error: `"${domain}" is not a valid registry host.` };
  }

  // reference/normalize.go expands a single-component Docker Hub name to
  // `library/<name>` and measures the length after that, so 248 already overflows.
  const normalizedPath =
    (!domain || DOCKER_HUB_DOMAINS.has(domain)) && !path.includes("/") ? `library/${path}` : path;
  if (normalizedPath.length > MAX_PATH_LENGTH) {
    return {
      ok: false,
      error: `An image name cannot be longer than ${MAX_PATH_LENGTH} characters.`,
    };
  }
  for (const component of pathComponents) {
    if (component !== component.toLowerCase()) {
      return { ok: false, error: "Image names must be lowercase." };
    }
    if (!PATH_COMPONENT.test(component)) {
      return {
        ok: false,
        error: "An image name can only contain lowercase letters, digits, and . _ - separators.",
      };
    }
  }

  const parts: ImageRefParts = { domain, path, tag, digest };

  if (!tag && !digest) {
    return { ok: true, parts, warning: "No tag given, so :latest is pulled at deploy time." };
  }
  if (tag === "latest" && !digest) {
    return {
      ok: true,
      parts,
      warning: ":latest is mutable. Pin a version or digest to redeploy the same build.",
    };
  }

  return { ok: true, parts };
};

/**
 * Lowercases the name and leaves the tag and digest alone. Splits at the first `@`,
 * then at the last `:` after the last `/`, so a registry port stays in the name.
 */
const lowercaseName = (ref: string): string => {
  const at = ref.indexOf("@");
  const named = at === -1 ? ref : ref.slice(0, at);
  const lastColon = named.lastIndexOf(":");
  const lastSlash = named.lastIndexOf("/");
  const nameEnd = lastColon > lastSlash ? lastColon : named.length;
  return named.slice(0, nameEnd).toLowerCase() + named.slice(nameEnd) + ref.slice(named.length);
};
