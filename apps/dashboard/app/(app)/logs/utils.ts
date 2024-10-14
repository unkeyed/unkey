import type { Log, ResponseBody } from "./types";

export const getResponseBodyFieldOutcome = <K extends keyof ResponseBody>(
  log: Log,
  fieldName: K,
): ResponseBody[K] | null => {
  try {
    const parsedBody = JSON.parse(log.response_body) as ResponseBody;

    if (typeof parsedBody !== "object" || parsedBody === null) {
      throw new Error("Parsed response body is not an object");
    }

    if (!(fieldName in parsedBody)) {
      throw new Error(`Field "${String(fieldName)}" not found in response body`);
    }

    return parsedBody[fieldName];
  } catch (error) {
    if (error instanceof Error) {
      console.error(`Error parsing response body or accessing field: ${error.message}`);
    } else {
      console.error("An unknown error occurred while parsing response body");
    }
    return null;
  }
};

export const getObjectsFromLogs = (log: Log): string => {
  const obj: Record<string, unknown> = {
    responseHeaders: log.response_headers,
    requestHeaders: log.request_headers,
  };

  try {
    obj.responseBody = JSON.parse(log.response_body);
  } catch (error) {
    console.error(
      "Error parsing response_body:",
      error instanceof Error ? error.message : "Unknown error",
    );
    obj.responseBody = { error: "Malformed response body" };
  }

  try {
    // Ensure we're returning a valid JSON string
    return JSON.stringify(obj, null, 2);
  } catch (error) {
    console.error(
      "Error stringifying object:",
      error instanceof Error ? error.message : "Unknown error",
    );
    // In case of stringification error, return a simple error JSON
    return JSON.stringify({ error: "Failed to stringify log data" }, null, 2);
  }
};

export const getRequestHeader = (log: Log, headerName: string): string | null => {
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
