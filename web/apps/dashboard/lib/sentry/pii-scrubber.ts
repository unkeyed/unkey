import type { ErrorEvent, EventHint, NodeOptions, replayIntegration } from "@sentry/nextjs";
import type { Router } from "../trpc/routers";

export type TransactionEvent = Parameters<NonNullable<NodeOptions["beforeSendTransaction"]>>[0];

const REDACTED = "[REDACTED]";

const SENSITIVE_NAME_KEYS = new Set(
  [
    "key",
    "apikey",
    "api_key",
    "rootkey",
    "root_key",
    "token",
    "access_token",
    "refresh_token",
    "id_token",
    "secret",
    "client_secret",
    "password",
    "pwd",
    "code",
    "state",
    "jwt",
    "authorization",
    "auth",
    "session",
    "email",
    "phone",
  ].map((k) => k.toLowerCase()),
);

const URL_ONLY_SENSITIVE_PARAM_KEYS = new Set(["input"]);

const TOKEN_LIKE = /[A-Za-z0-9_-]{20,}/g;

const EMAIL_LIKE = /[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}/g;

function isSensitiveKey(name: string): boolean {
  return SENSITIVE_NAME_KEYS.has(name.toLowerCase());
}

function redactTokenLike(value: string): string {
  return value.replace(EMAIL_LIKE, REDACTED).replace(TOKEN_LIKE, REDACTED);
}

function redactDigitBearingTokens(text: string): string {
  return text.replace(TOKEN_LIKE, (run) => (/\d/.test(run) ? REDACTED : run));
}

function scrubUrlPath(path: string): string {
  const emailSafe = path
    .split("/")
    .map((segment) => {
      let decoded = segment;
      try {
        decoded = decodeURIComponent(segment);
      } catch {
      }
      if (decoded.replace(EMAIL_LIKE, REDACTED) !== decoded) {
        return REDACTED;
      }
      return segment;
    })
    .join("/");
  return redactDigitBearingTokens(emailSafe);
}

const AMBIGUOUS_BRACKET_LEAVES = new Set(["key", "state", "code", "auth", "session"]);

function bracketLeafSegment(name: string): string | null {
  if (!name.includes("[")) {
    return null;
  }
  const segments = name.split(/[[\]]+/).filter((segment) => segment.length > 0);
  return segments[segments.length - 1] ?? null;
}

function isSensitiveParamKey(name: string): boolean {
  const lower = name.toLowerCase();
  if (lower.startsWith("param_")) {
    return true;
  }
  if (isSensitiveKey(lower) || URL_ONLY_SENSITIVE_PARAM_KEYS.has(lower)) {
    return true;
  }
  const leaf = bracketLeafSegment(lower);
  return leaf !== null && isSensitiveKey(leaf) && !AMBIGUOUS_BRACKET_LEAVES.has(leaf);
}

function scrubParamValue(name: string, value: string): string {
  if (isSensitiveParamKey(name)) {
    return REDACTED;
  }
  return redactTokenLike(value);
}

function scrubSearchParams(params: URLSearchParams): void {
  const entries = [...params.entries()];
  for (const [name] of entries) {
    params.delete(name);
  }
  for (const [name, value] of entries) {
    params.append(name, scrubParamValue(name, value));
  }
}

export function scrubUrl(url: string): string {
  if (typeof url !== "string" || url.length === 0) {
    return url;
  }

  const opaqueScheme = url.match(/^(mailto|tel|blob|data):(?!\/\/)/i)?.[1];
  if (opaqueScheme) {
    return `${opaqueScheme}:${REDACTED}`;
  }

  try {
    const base = "http://scrub.local";
    const parsed = new URL(url, base);

    parsed.username = "";
    parsed.password = "";
    scrubSearchParams(parsed.searchParams);
    const hadScheme = /^[a-z][a-z0-9+.-]*:/i.test(url);
    if (hadScheme && parsed.host === "" && !url.startsWith("//")) {
      const scrubbedPath = scrubUrlPath(parsed.pathname);
      return `${parsed.protocol}${scrubbedPath}${parsed.search}`;
    }

    if (!parsed.pathname.startsWith("/_next/static/")) {
      parsed.pathname = scrubUrlPath(parsed.pathname);
    }

    parsed.hash = "";

    if (url.startsWith("//")) {
      return `//${parsed.host}${parsed.pathname}${parsed.search}`;
    }

    const wasRelative = !/^[a-z][a-z0-9+.-]*:\/\//i.test(url);
    if (wasRelative) {
      return `${parsed.pathname}${parsed.search}`;
    }
    return parsed.toString();
  } catch {
    return redactTokenLike(url);
  }
}

function scrubQueryParamsString(queryString: string): string {
  const params = new URLSearchParams(queryString);
  scrubSearchParams(params);
  return params.toString();
}

function scrubQueryString(
  queryString: NonNullable<NonNullable<ErrorEvent["request"]>["query_string"]>,
): NonNullable<NonNullable<ErrorEvent["request"]>["query_string"]> {
  if (typeof queryString === "string") {
    return scrubQueryParamsString(queryString);
  }

  if (Array.isArray(queryString)) {
    return queryString.map(([name, value]): [string, string] => [
      name,
      scrubParamValue(name, value),
    ]);
  }

  if (queryString && typeof queryString === "object") {
    const result: Record<string, string> = {};
    for (const [name, value] of Object.entries(queryString)) {
      result[name] = scrubParamValue(name, value);
    }
    return result;
  }

  return queryString;
}

function scrubBreadcrumbs(event: ErrorEvent | TransactionEvent, run: StepRunner): void {
  if (!event.breadcrumbs) {
    return;
  }
  for (const breadcrumb of event.breadcrumbs) {
    run(() => scrubBreadcrumb(breadcrumb));
  }
}

function scrubBreadcrumbMessage(category: string | undefined, message: string): string {
  return category?.startsWith("ui.") ? message.replace(EMAIL_LIKE, REDACTED) : scrubText(message);
}

function scrubConsoleArguments(category: string | undefined, data: unknown): void {
  if (category !== "console" || !data || typeof data !== "object") {
    return;
  }
  const record = data as Record<string, unknown>;
  if (Array.isArray(record.arguments)) {
    record.arguments = record.arguments.map(redactLogValue);
  }
}

function scrubBreadcrumb(breadcrumb: NonNullable<ErrorEvent["breadcrumbs"]>[number]): void {
  if (typeof breadcrumb.message === "string") {
    breadcrumb.message = scrubBreadcrumbMessage(breadcrumb.category, breadcrumb.message);
  }
  const data = breadcrumb.data;
  if (!data) {
    return;
  }
  scrubConsoleArguments(breadcrumb.category, data);
  for (const field of ["url", "from", "to"] as const) {
    const value = data[field];
    if (typeof value === "string") {
      data[field] = scrubUrl(value);
    }
  }
}

export function scrubEventPii(event: ErrorEvent, _hint?: EventHint): void {
  isolate(() => scrubTrpcInput(event));
  isolate(() => scrubExtra(event));
  isolate(() => scrubExceptions(event));
  isolate(() => scrubMessage(event));
  isolate(() => scrubTransactionName(event));
  isolate(() => scrubSpanAttributes(event.contexts?.trace?.data));
  scrubRequest(event.request, isolate);
  scrubBreadcrumbs(event, isolate);
}

function scrubTransactionName(event: ErrorEvent | TransactionEvent): void {
  if (typeof event.transaction !== "string") {
    return;
  }
  event.transaction = isSqlSpan({
    op: event.contexts?.trace?.op,
    data: event.contexts?.trace?.data,
  })
    ? maskSqlLiterals(event.transaction)
    : scrubUrlsInText(event.transaction);
}

type StepRunner = (step: () => void) => void;

function isolate(step: () => void): void {
  try {
    step();
  } catch {
  }
}

function propagate(step: () => void): void {
  step();
}

const FORM_ENCODED = /^[\w.%+\-[\]]+=[^&]*(?:&[\w.%+\-[\]]+=[^&]*)*$/;

function scrubRequestData(data: unknown): unknown {
  if (typeof data !== "string") {
    return redactSensitiveValues(data);
  }

  let parsed: unknown;
  let isJson = false;
  try {
    parsed = JSON.parse(data);
    isJson = true;
  } catch {
  }

  if (isJson && parsed && typeof parsed === "object") {
    return JSON.stringify(redactSensitiveValues(parsed));
  }
  if (FORM_ENCODED.test(data)) {
    return scrubQueryParamsString(data);
  }
  return redactTokenLike(data);
}

const URL_HEADERS = new Set(["referer", "referrer", "location", ":path"]);

const HEADER_SCRUB_EXEMPT = new Set([
  ":authority",
  ":method",
  ":scheme",
  "host",
  "x-request-id",
  "x-correlation-id",
  "x-amzn-trace-id",
  "x-vercel-id",
  "cf-ray",
  "traceparent",
  "tracestate",
  "sentry-trace",
  "baggage",
  "etag",
  "if-none-match",
  "if-match",
]);

const SENSITIVE_HEADERS = new Set([
  "authorization",
  "proxy-authorization",
  "cookie",
  "set-cookie",
  "x-api-key",
]);

const SENSITIVE_HEADER_PATTERNS = ["auth", "token", "secret", "session", "credential", "key"];

function isSensitiveHeaderName(lowerName: string): boolean {
  if (SENSITIVE_HEADERS.has(lowerName)) {
    return true;
  }
  return SENSITIVE_HEADER_PATTERNS.some((pattern) => lowerName.includes(pattern));
}

function scrubRequest(
  request: ErrorEvent["request"] | TransactionEvent["request"],
  run: StepRunner,
): void {
  if (!request) {
    return;
  }

  run(() => {
    if (typeof request.url === "string") {
      request.url = scrubUrl(request.url);
    }
  });

  run(() => {
    if (request.query_string != null) {
      request.query_string = scrubQueryString(request.query_string);
    }
  });

  run(() => {
    if (request.data == null) {
      return;
    }
    try {
      request.data = scrubRequestData(request.data);
    } catch {
      delete request.data;
    }
  });

  run(() => {
    if (!request.cookies) {
      return;
    }
    for (const name of Object.keys(request.cookies)) {
      request.cookies[name] = REDACTED;
    }
  });

  run(() => {
    if (!request.headers) {
      return;
    }
    for (const [name, value] of Object.entries(request.headers)) {
      const lowerName = name.toLowerCase();
      if (HEADER_SCRUB_EXEMPT.has(lowerName)) {
        continue;
      }
      if (isSensitiveHeaderName(lowerName)) {
        request.headers[name] = REDACTED;
      } else if (URL_HEADERS.has(lowerName) && typeof value === "string") {
        request.headers[name] = scrubUrl(value);
      } else if (typeof value === "string") {
        request.headers[name] = redactTokenLike(value);
      }
    }
  });
}

const SENSITIVE_INPUT_KEYS = new Set(["value"]);

function isSensitiveInputKey(key: string): boolean {
  return SENSITIVE_INPUT_KEYS.has(key.toLowerCase()) || isSensitiveKey(key);
}

type ShareProcedurePath = `share.${Extract<keyof Router["share"]["_def"]["record"], string>}`;

const CREDENTIAL_INPUT_PROCEDURES: ReadonlySet<string> = new Set<ShareProcedurePath>([
  "share.reveal",
]);

function redactStructured(
  value: unknown,
  scrubString: (s: string) => string,
  ancestors: WeakSet<object>,
): unknown {
  if (typeof value === "string") {
    return scrubString(value);
  }
  if (!value || typeof value !== "object") {
    return value;
  }
  if (ancestors.has(value)) {
    return REDACTED;
  }

  if (value instanceof Error) {
    ancestors.add(value);
    try {
      return redactError(value, scrubString, ancestors);
    } finally {
      ancestors.delete(value);
    }
  }
  if (value instanceof Map && [...value.keys()].every((key) => typeof key === "string")) {
    ancestors.add(value);
    try {
      return redactStructuredRecord(
        Object.fromEntries(value) as Record<string, unknown>,
        scrubString,
        ancestors,
      );
    } finally {
      ancestors.delete(value);
    }
  }

  const prototype = Object.getPrototypeOf(value);
  if (!Array.isArray(value) && prototype !== Object.prototype && prototype !== null) {
    return value;
  }

  ancestors.add(value);
  try {
    if (Array.isArray(value)) {
      return value.map((item) => redactStructured(item, scrubString, ancestors));
    }
    return redactStructuredRecord(value as Record<string, unknown>, scrubString, ancestors);
  } finally {
    ancestors.delete(value);
  }
}

function redactError(
  error: Error,
  scrubString: (s: string) => string,
  ancestors: WeakSet<object>,
): Record<string, unknown> {
  const result: Record<string, unknown> = { name: error.name };
  if (typeof error.message === "string") {
    result.message = scrubString(error.message);
  }
  if (typeof error.stack === "string") {
    result.stack = scrubErrorStack(error.stack, scrubString);
  }
  if ("cause" in error && error.cause !== undefined) {
    result.cause = redactStructured(error.cause, scrubString, ancestors);
  }
  if (error instanceof AggregateError && Array.isArray(error.errors)) {
    result.errors = error.errors.map((nested) => redactStructured(nested, scrubString, ancestors));
  }
  for (const [key, nested] of Object.entries(error)) {
    if (key === "name" || key === "message" || key === "stack" || key === "cause") {
      continue;
    }
    result[key] = isSensitiveInputKey(key)
      ? REDACTED
      : redactStructured(nested, scrubString, ancestors);
  }
  return result;
}

function scrubErrorStack(stack: string, scrubString: (s: string) => string): string {
  const lines = stack.split("\n");
  const firstFrame = lines.findIndex((line) => /^\s+at\s/.test(line));
  const headerEnd = firstFrame === -1 ? lines.length : firstFrame;
  return lines.map((line, index) => (index < headerEnd ? scrubString(line) : line)).join("\n");
}

function redactStructuredRecord(
  record: Record<string, unknown>,
  scrubString: (s: string) => string,
  ancestors: WeakSet<object>,
): Record<string, unknown> {
  const result: Record<string, unknown> = {};
  for (const [key, nested] of Object.entries(record)) {
    result[key] = isSensitiveInputKey(key)
      ? REDACTED
      : redactStructured(nested, scrubString, ancestors);
  }
  return result;
}

function redactSensitiveValues(value: unknown): unknown {
  return redactStructured(value, redactTokenLike, new WeakSet());
}

function redactSensitiveRecord(record: Record<string, unknown>): Record<string, unknown> {
  return redactStructuredRecord(record, redactTokenLike, new WeakSet());
}

function scrubTrpcInput(event: ErrorEvent | TransactionEvent): void {
  const trpcContext = event.contexts?.trpc;
  if (!trpcContext || !("input" in trpcContext)) {
    return;
  }

  try {
    const procedurePath = trpcContext.procedure_path;
    if (typeof procedurePath === "string" && CREDENTIAL_INPUT_PROCEDURES.has(procedurePath)) {
      trpcContext.input = REDACTED;
      return;
    }

    trpcContext.input = redactSensitiveValues(trpcContext.input);
  } catch {
    trpcContext.input = REDACTED;
  }
}

const URL_ATTRIBUTE_KEYS = ["http.url", "url.full", "url", "http.target", "url.path", "next.url"];

const QUERY_ATTRIBUTE_KEYS = ["http.query", "url.query"];

const FRAGMENT_ATTRIBUTE_KEYS = ["http.fragment", "url.fragment"];

const TEXT_ATTRIBUTE_KEYS = ["transaction"];

const SQL_ATTRIBUTE_KEYS = ["db.query.text", "db.statement"];

const SCRUBBED_ATTRIBUTE_KEYS = [
  ...URL_ATTRIBUTE_KEYS,
  ...QUERY_ATTRIBUTE_KEYS,
  ...FRAGMENT_ATTRIBUTE_KEYS,
  ...TEXT_ATTRIBUTE_KEYS,
  ...SQL_ATTRIBUTE_KEYS,
];

function maskSqlLiterals(sql: string): string {
  let body = sql;
  let trailer = "";
  if (sql.trimEnd().endsWith("*/")) {
    const openIndex = sql.lastIndexOf("/*");
    if (openIndex >= 0) {
      body = sql.slice(0, openIndex);
      trailer = sql.slice(openIndex);
    }
  }
  return (
    body
      .replace(/'(?:[^'\\]|\\.)*'/g, "?")
      .replace(/"(?:[^"\\]|\\.)*"/g, "?")
      .replace(/\b(?:0x[0-9a-fA-F]+|\d+(?:\.\d+)?(?:[eE][+-]?\d+)?)\b/g, "?") + trailer
  );
}

const HEADER_ATTRIBUTE_PREFIXES = ["http.request.header.", "http.response.header."];

function scrubSpanAttributes(attributes: Record<string, unknown> | undefined): void {
  if (!attributes) {
    return;
  }
  for (const key of URL_ATTRIBUTE_KEYS) {
    const value = attributes[key];
    if (typeof value === "string") {
      attributes[key] = scrubUrl(value);
    }
  }
  for (const key of QUERY_ATTRIBUTE_KEYS) {
    const value = attributes[key];
    if (typeof value === "string") {
      const scrubbed = scrubQueryParamsString(value);
      attributes[key] = value.startsWith("?") ? `?${scrubbed}` : scrubbed;
    }
  }
  for (const key of FRAGMENT_ATTRIBUTE_KEYS) {
    delete attributes[key];
  }
  for (const key of TEXT_ATTRIBUTE_KEYS) {
    const value = attributes[key];
    if (typeof value === "string") {
      attributes[key] = scrubUrlsInText(value);
    }
  }
  for (const key of SQL_ATTRIBUTE_KEYS) {
    const value = attributes[key];
    if (typeof value === "string") {
      attributes[key] = maskSqlLiterals(value);
    }
  }
  for (const [key, value] of Object.entries(attributes)) {
    if (typeof value !== "string") {
      continue;
    }
    const prefix = HEADER_ATTRIBUTE_PREFIXES.find((p) => key.startsWith(p));
    if (prefix && URL_HEADERS.has(key.slice(prefix.length))) {
      attributes[key] = scrubUrl(value);
    }
  }
}

const EMBEDDED_URL = /^(.*?[=(])((?:https?:\/\/|\/(?!\*)).*?)(\)*)$/i;

function scrubUrlsInText(text: string): string {
  return mapTextTokens(text.replace(EMAIL_LIKE, REDACTED), (token) => token);
}

function mapTextTokens(text: string, scrubNonUrlToken: (token: string) => string): string {
  return text
    .split(/(\s+)/)
    .map((token) => {
      if (token.length === 0 || /^\s/.test(token)) {
        return token;
      }
      if (token.startsWith("/*")) {
        return token;
      }
      if (token.startsWith("/") || /^https?:\/\//i.test(token)) {
        return scrubUrl(token);
      }
      const embedded = token.match(EMBEDDED_URL);
      if (embedded) {
        return embedded[1] + scrubUrl(embedded[2]) + embedded[3];
      }
      return scrubNonUrlToken(token);
    })
    .join("");
}

function scrubText(text: string): string {
  return mapTextTokens(text.replace(EMAIL_LIKE, REDACTED), redactDigitBearingTokens);
}

function scrubExceptions(event: ErrorEvent): void {
  for (const exception of event.exception?.values ?? []) {
    if (typeof exception.value === "string") {
      exception.value = scrubText(exception.value);
    }
  }
}

function scrubMessage(event: ErrorEvent): void {
  if (typeof event.message === "string") {
    event.message = scrubText(event.message);
  }

  const logentry = event.logentry;
  if (!logentry) {
    return;
  }
  const hasParams = Array.isArray(logentry.params) && logentry.params.length > 0;
  if (typeof logentry.message === "string" && !hasParams) {
    logentry.message = scrubText(logentry.message);
  }
  if (Array.isArray(logentry.params)) {
    logentry.params = logentry.params.map(redactLogValue);
  }
}

function redactLogValue(value: unknown): unknown {
  return redactStructured(value, scrubText, new WeakSet());
}

const CORRELATION_ATTRIBUTE_KEYS = new Set([
  "version",
  "user_id",
  "workspace_id",
  "request_id",
  "resource_id",
  "trpc_procedure",
]);

const CORRELATION_ID_VALUE_SHAPE = /^[A-Za-z0-9_.-]{1,64}$/;

function redactLogRecord(record: Record<string, unknown>): Record<string, unknown> {
  const result: Record<string, unknown> = {};
  const ancestors = new WeakSet<object>();
  for (const [key, value] of Object.entries(record)) {
    if (
      CORRELATION_ATTRIBUTE_KEYS.has(key) &&
      typeof value === "string" &&
      CORRELATION_ID_VALUE_SHAPE.test(value)
    ) {
      result[key] = value;
      continue;
    }
    if (key === "sentry.message.template" && typeof value === "string") {
      result[key] = value;
      continue;
    }
    result[key] = isSensitiveInputKey(key)
      ? REDACTED
      : redactStructured(value, scrubText, ancestors);
  }
  return result;
}

function scrubExtra(event: ErrorEvent | TransactionEvent): void {
  if (!event.extra) {
    return;
  }
  try {
    event.extra = redactSensitiveRecord(event.extra);
  } catch {
    delete event.extra;
  }
}

function isSqlSpan(span: { op?: string; data?: Record<string, unknown> }): boolean {
  if (span.op?.startsWith("db")) {
    return true;
  }
  return SQL_ATTRIBUTE_KEYS.some((key) => typeof span.data?.[key] === "string");
}

function scrubSpanFields(span: {
  op?: string;
  description?: string;
  data?: Record<string, unknown>;
}): void {
  if (typeof span.description === "string") {
    span.description = isSqlSpan(span)
      ? maskSqlLiterals(span.description)
      : scrubUrlsInText(span.description);
  }
  scrubSpanAttributes(span.data);
}

export function scrubTransactionPii(
  event: TransactionEvent,
  _hint?: EventHint,
): TransactionEvent | null {
  try {
    scrubTransactionName(event);
    scrubRequest(event.request, propagate);
    scrubBreadcrumbs(event, propagate);
    scrubTrpcInput(event);
    scrubExtra(event);
    scrubSpanAttributes(event.contexts?.trace?.data);

    for (const span of event.spans ?? []) {
      scrubSpanFields(span);
    }
    return event;
  } catch {
    return null;
  }
}

export type SpanJson = Parameters<NonNullable<NodeOptions["beforeSendSpan"]>>[0];

export function scrubSpanPii(span: SpanJson): SpanJson {
  let copy: SpanJson;
  try {
    copy = { ...span, data: span.data ? { ...span.data } : span.data };
  } catch {
    return span;
  }

  try {
    scrubSpanFields(copy);
    return copy;
  } catch {
    delete copy.description;
    if (copy.data) {
      for (const key of SCRUBBED_ATTRIBUTE_KEYS) {
        delete copy.data[key];
      }
      for (const key of Object.keys(copy.data)) {
        if (HEADER_ATTRIBUTE_PREFIXES.some((p) => key.startsWith(p))) {
          delete copy.data[key];
        }
      }
    }
    return copy;
  }
}

export type SentryLog = Parameters<NonNullable<NodeOptions["beforeSendLog"]>>[0];

export function scrubLog(log: SentryLog): SentryLog | null {
  try {
    const message: unknown = log.message;
    if (typeof message === "string") {
      log.message = scrubText(message);
    } else if (message instanceof String) {
      log.message = scrubText(message.valueOf());
    }
    if (log.attributes) {
      log.attributes = redactLogRecord(log.attributes);
    }
    return log;
  } catch {
    return null;
  }
}

export type ReplayFrameEvent = Parameters<
  NonNullable<NonNullable<Parameters<typeof replayIntegration>[0]>["beforeAddRecordingEvent"]>
>[0];

type ReplayBreadcrumbPayload = Extract<ReplayFrameEvent["data"], { tag: "breadcrumb" }>["payload"];
type ReplaySpanPayload = Extract<ReplayFrameEvent["data"], { tag: "performanceSpan" }>["payload"];

const REPLAY_URL_DATA_FIELDS = new Set(["url", "from", "to", "previous"]);

export function scrubReplayFrame(frame: ReplayFrameEvent): ReplayFrameEvent | null {
  try {
    const data = frame.data;
    if (data?.tag === "breadcrumb") {
      scrubReplayBreadcrumb(data.payload);
    } else if (data?.tag === "performanceSpan") {
      scrubReplaySpan(data.payload);
    }
    return frame;
  } catch {
    return null;
  }
}

function scrubReplayBreadcrumb(payload: ReplayBreadcrumbPayload): void {
  if (typeof payload.message === "string") {
    payload.message = scrubBreadcrumbMessage(payload.category, payload.message);
  }
  scrubConsoleArguments(payload.category, payload.data);
  scrubReplayUrlDataFields(payload.data);
}

function scrubReplaySpan(payload: ReplaySpanPayload): void {
  if (typeof payload.description === "string") {
    payload.description = scrubUrlsInText(payload.description);
  }
  scrubReplayUrlDataFields(payload.data);
}

function scrubReplayUrlDataFields(data: unknown): void {
  if (!data || typeof data !== "object") {
    return;
  }
  const record = data as Record<string, unknown>;
  for (const key of Object.keys(record)) {
    const value = record[key];
    if (typeof value === "string" && REPLAY_URL_DATA_FIELDS.has(key)) {
      record[key] = scrubUrl(value);
    }
  }
}
