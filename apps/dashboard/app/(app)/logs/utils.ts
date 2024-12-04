import type { Log, ResponseBody } from "./types";

class ResponseBodyParseError extends Error {
  constructor(
    message: string,
    public readonly context?: unknown,
  ) {
    super(message);
    this.name = "ResponseBodyParseError";
  }
}

export const getResponseBodyFieldOutcome = <K extends keyof ResponseBody>(
  log: Log,
  fieldName: K,
): ResponseBody[K] | null => {
  if (!log?.response_body) {
    console.error("Invalid log or missing response_body");
    return null;
  }

  try {
    const parsedBody = JSON.parse(log.response_body) as ResponseBody;

    if (typeof parsedBody !== "object" || parsedBody === null) {
      throw new ResponseBodyParseError("Parsed response body is not an object", parsedBody);
    }

    if (!(fieldName in parsedBody)) {
      throw new ResponseBodyParseError(`Field "${String(fieldName)}" not found in response body`, {
        availableFields: Object.keys(parsedBody),
      });
    }

    return parsedBody[fieldName];
  } catch (error) {
    if (error instanceof ResponseBodyParseError) {
      console.error(`Error parsing response body or accessing field: ${error.message}`, {
        context: error.context,
        fieldName,
        logId: log.request_id,
      });
    } else {
      console.error("An unknown error occurred while parsing response body");
    }
    return null;
  }
};

export const getRequestHeader = (log: Log, headerName: string): string | null => {
  if (!headerName.trim()) {
    console.error("Invalid header name provided");
    return null;
  }

  if (!Array.isArray(log.request_headers)) {
    console.error("request_headers is not an array");
    return null;
  }

  const lowerHeaderName = headerName.toLowerCase();
  const header = log.request_headers.find((h) => h.toLowerCase().startsWith(`${lowerHeaderName}:`));

  if (!header) {
    console.warn(`Header "${headerName}" not found in request headers`);
    return null;
  }

  const [, value] = header.split(":", 2);
  return value ? value.trim() : null;
};
