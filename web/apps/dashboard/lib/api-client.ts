import { z } from "zod";

const apiErrorSchema = z.object({
  error: z.object({
    title: z.string(),
    type: z.string(),
    detail: z.string(),
    status: z.number(),
  }),
  meta: z.object({
    requestId: z.string(),
  }),
});

export type ApiError = z.infer<typeof apiErrorSchema>;

const unknownError: ApiError = {
  error: {
    title: "Unknown Error",
    type: "unknown",
    detail: "An unexpected error occurred",
    status: 500,
  },
  meta: { requestId: "" },
};

export type ApiResponse<T> =
  | { success: true; data: T; requestId: string }
  | { success: false; error: ApiError };

/**
 * Low-level function for calling the Unkey Go API with a session token.
 * The token should be obtained via the `getAccessToken()` server action.
 *
 * This runs on the client — the browser makes the request directly to the Go API.
 */
export async function apiClient<T>(
  accessToken: string,
  path: string,
  options: {
    method: "GET" | "POST" | "PUT" | "DELETE";
    body?: unknown;
    responseSchema: z.ZodType<T>;
  },
): Promise<ApiResponse<T>> {
  const baseUrl = process.env.NEXT_PUBLIC_UNKEY_API_URL ?? "http://localhost:7070";

  const res = await fetch(`${baseUrl}${path}`, {
    method: options.method,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${accessToken}`,
    },
    body: options.body ? JSON.stringify(options.body) : undefined,
  });

  let json: unknown;
  try {
    json = await res.json();
  } catch {
    return {
      success: false,
      error: {
        error: {
          title: "Parse Error",
          type: "parse_error",
          detail: "Invalid JSON response from API",
          status: res.status,
        },
        meta: { requestId: "" },
      },
    };
  }

  if (!res.ok) {
    const parsed = apiErrorSchema.safeParse(json);
    if (parsed.success) {
      return { success: false, error: parsed.data };
    }
    return {
      success: false,
      error: {
        error: {
          title: "Unknown Error",
          type: "unknown",
          detail: "An unexpected error occurred",
          status: res.status,
        },
        meta: { requestId: "" },
      },
    };
  }

  const envelope = z
    .object({
      data: options.responseSchema,
      meta: z.object({ requestId: z.string() }).optional(),
    })
    .safeParse(json);

  if (!envelope.success) {
    return {
      success: false,
      error: {
        error: {
          title: "Parse Error",
          type: "parse_error",
          detail: "Unexpected response shape from API",
          status: res.status,
        },
        meta: { requestId: "" },
      },
    };
  }

  return {
    success: true,
    data: envelope.data.data,
    requestId: envelope.data.meta?.requestId ?? "",
  };
}

// Typed API methods

const createApiResponseSchema = z.object({
  apiId: z.string(),
});

export type CreateApiResponse = z.infer<typeof createApiResponseSchema>;

export async function createApi(
  accessToken: string,
  name: string,
): Promise<ApiResponse<CreateApiResponse>> {
  return apiClient<CreateApiResponse>(accessToken, "/v2/apis.createApi", {
    method: "POST",
    body: { name },
    responseSchema: createApiResponseSchema,
  });
}
