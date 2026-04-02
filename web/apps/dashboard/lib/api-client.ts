export type ApiError = {
  error: {
    title: string;
    type: string;
    detail: string;
    status: number;
  };
  meta: {
    requestId: string;
  };
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

  const json = await res.json();

  if (!res.ok) {
    return {
      success: false,
      error: json as ApiError,
    };
  }

  return {
    success: true,
    data: json.data as T,
    requestId: json.meta?.requestId ?? "",
  };
}

// Typed API methods

export type CreateApiResponse = {
  apiId: string;
};

export async function createApi(
  accessToken: string,
  name: string,
): Promise<ApiResponse<CreateApiResponse>> {
  return apiClient<CreateApiResponse>(accessToken, "/v2/apis.createApi", {
    method: "POST",
    body: { name },
  });
}
