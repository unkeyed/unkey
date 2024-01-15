import { type AnyFunction } from "bun";
import { type Mock, expect, mock } from "vitest";

interface TestOneFetchCallArgs<T> {
  url: string;
  method?: string;
  headers?: Record<string, string>;
  jsonBody?: T;
  execute: (jsonBody: T) => Promise<unknown>;
}

const originalFetch = globalThis.fetch;
export function resetFetchMock() {
  globalThis.fetch = originalFetch;
}

export async function testOneFetchCall<T>({
  url,
  method,
  headers,
  jsonBody,
  execute,
}: TestOneFetchCallArgs<T>): Promise<Mock<AnyFunction>> {
  const mockFetch = mock(async (_url, _options) => ({
    ok: true,
    json: async () => ({}),
  }));

  // @ts-ignore
  globalThis.fetch = mockFetch;
  await execute(jsonBody as T);

  expect(mockFetch).toHaveBeenCalledTimes(1);

  const [callUrl, callOptions] = mockFetch.mock.lastCall!;
  expect(callUrl.toString()).toBe(url);
  expect<string | undefined>(callOptions.method).toBe(method);
  expect(callOptions.headers).toMatchObject({
    "Content-Type": "application/json",
    Authorization: "Bearer root key",
    ...(headers ?? {}),
  });
  expect<string | undefined>(callOptions.body).toBe(
    jsonBody === undefined ? undefined : JSON.stringify(jsonBody),
  );

  return mockFetch;
}
