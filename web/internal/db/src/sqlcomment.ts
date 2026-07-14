export type SqlCommentStaticTags = {
  application: string;
  service: string;
  region?: string;
  releaseSha?: string;
};

/** Per-request tags set through AsyncLocalStorage (for example tRPC middleware). */
export type SqlCommentDynamicTags = {
  /** tRPC procedure path, webhook route, or other entrypoint identifier. */
  route?: string;
  /** Call site family, for example `trpc`, `webhook`, or `server-action`. */
  source?: string;
};

function escapeTagValue(value: string): string {
  const encoded = encodeURIComponent(value);
  return encoded.replaceAll("'", "\\'");
}

function formatComment(
  staticTags: SqlCommentStaticTags,
  dynamicTags: SqlCommentDynamicTags,
): string {
  const entries: Array<[string, string | undefined]> = [
    ["application", staticTags.application],
    ["service", staticTags.service],
    ["region", staticTags.region],
    ["release_sha", staticTags.releaseSha],
    ["route", dynamicTags.route],
    ["source", dynamicTags.source],
  ];

  const parts = entries
    .filter((entry): entry is [string, string] => Boolean(entry[1]))
    .map(([key, value]) => `${key}='${escapeTagValue(value)}'`);

  if (parts.length === 0) {
    return "";
  }

  return `/*${parts.join(",")}*/`;
}

/** Appends SQLCommenter metadata to a Drizzle/mysql2 SQL string. */
export function annotateSql(
  query: string,
  staticTags: SqlCommentStaticTags,
  dynamicTags: SqlCommentDynamicTags = {},
): string {
  if (!staticTags.service) {
    return query;
  }

  const comment = formatComment(staticTags, dynamicTags);
  if (!comment) {
    return query.trim();
  }
  return `${query.trim()} ${comment}`;
}

export function staticTagsFromEnv(service: string): SqlCommentStaticTags {
  const revision = process.env.UNKEY_GIT_COMMIT_SHA ?? process.env.GIT_COMMIT ?? "";
  const releaseSha = revision === "" || revision === "unknown" ? undefined : revision.slice(0, 7);

  return {
    application: "unkey",
    service,
    region: process.env.UNKEY_REGION ?? process.env.REGION,
    releaseSha,
  };
}
