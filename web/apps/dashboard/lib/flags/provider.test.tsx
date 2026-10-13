import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import type { V2WorkspaceGetFlagsResponseBody } from "@unkey/api/models/components";
import { useState } from "react";
import { renderToString } from "react-dom/server";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { FlagProvider, useFlag, useFlags } from "./provider";

beforeEach(() => {
  vi.stubEnv("UNKEY_ROOT_KEY", undefined);
});

afterEach(() => {
  cleanup();
  vi.unstubAllGlobals();
  vi.unstubAllEnvs();
});

function Consumer() {
  const billing = useFlag("deployBilling");
  const projects = useFlag("projectsNav");
  const upgrades = useFlag("billingUIUpgrades");
  return <output>{JSON.stringify({ billing, projects, upgrades })}</output>;
}

it("renders server flags immediately without a duplicate request and preserves false before fallbacks", async () => {
  const fetcher = vi.fn();
  vi.stubGlobal("fetch", fetcher);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const tree = (
    <QueryClientProvider client={client}>
      <FlagProvider
        initialFlags={{
          deploy_billing: {
            value: false,
            description: "Billing",
            allowEnable: false,
            allowDisable: false,
          },
          projects_nav: {
            value: true,
            description: "Navigation",
            allowEnable: true,
            allowDisable: true,
          },
        }}
      >
        <Consumer />
        <Consumer />
      </FlagProvider>
    </QueryClientProvider>
  );
  const container = document.createElement("div");
  document.body.append(container);
  container.innerHTML = renderToString(tree);
  expect(
    screen.getAllByRole("status").map((node) => JSON.parse(node.textContent ?? "null")),
  ).toEqual([
    { billing: false, projects: true, upgrades: true },
    { billing: false, projects: true, upgrades: true },
  ]);
  render(tree, { container, hydrate: true });
  await act(async () => {});
  expect(fetcher).not.toHaveBeenCalled();
  client.clear();
});

it("isolates flags when the auth boundary remounts, retries failed server reads, and skips anonymous reads", async () => {
  const fetcher = vi.fn(async () => new Response(null, { status: 403 }));
  vi.stubGlobal("fetch", fetcher);
  function State() {
    const flags = useFlags();
    return (
      <>
        <Consumer />
        {flags.isLoading && <span>Loading</span>}
        {flags.error && <span role="alert">Read failed</span>}
        <button type="button" onClick={() => flags.refetch()}>
          Refresh
        </button>
      </>
    );
  }
  const clients: QueryClient[] = [];
  function Tree({
    initialFlags,
  }: { initialFlags: V2WorkspaceGetFlagsResponseBody["data"] | null | undefined }) {
    const [client] = useState(() => {
      const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
      clients.push(client);
      return client;
    });
    return (
      <QueryClientProvider client={client}>
        <FlagProvider initialFlags={initialFlags}>
          <State />
        </FlagProvider>
      </QueryClientProvider>
    );
  }
  const view = render(
    <Tree
      key={crypto.randomUUID()}
      initialFlags={{
        projects_nav: {
          value: true,
          description: "Navigation",
          allowEnable: true,
          allowDisable: true,
        },
      }}
    />,
  );
  expect(JSON.parse(screen.getByRole("status").textContent ?? "null")).toEqual({
    billing: true,
    projects: true,
    upgrades: true,
  });
  view.rerender(<Tree key={crypto.randomUUID()} initialFlags={undefined} />);
  expect(JSON.parse(screen.getByRole("status").textContent ?? "null")).toEqual({
    billing: true,
    projects: false,
    upgrades: true,
  });
  expect(screen.getByText("Loading")).toBeDefined();
  expect(await screen.findByRole("alert")).toHaveProperty("textContent", "Read failed");
  expect(fetcher).toHaveBeenCalledTimes(1);
  view.rerender(<Tree key={crypto.randomUUID()} initialFlags={null} />);
  await act(async () => {});
  expect(fetcher).toHaveBeenCalledTimes(1);
  expect(screen.queryByRole("alert")).toBeNull();
  for (const client of clients) {
    client.clear();
  }
});

it("applies the saved value without losing metadata or letting an older read overwrite it", async () => {
  const entry = { value: false, description: "Navigation", allowEnable: true, allowDisable: false };
  const response = (value: boolean) =>
    new Response(
      JSON.stringify({
        meta: { requestId: crypto.randomUUID() },
        data: { projects_nav: { ...entry, value } },
      }),
      { headers: { "Content-Type": "application/json" } },
    );
  let finishOldRead: (response: Response) => void = () => {
    throw new Error("Old read did not start");
  };
  const oldRead = new Promise<Response>((resolve) => {
    finishOldRead = resolve;
  });
  let reads = 0;
  vi.stubGlobal("fetch", async () => {
    reads++;
    return oldRead;
  });
  function Controls() {
    const flags = useFlags();
    return (
      <>
        <Consumer />
        <pre data-testid="entry">{JSON.stringify(flags.data?.projects_nav)}</pre>
        <button type="button" onClick={() => flags.refetch()}>
          Read
        </button>
        <button
          type="button"
          onClick={() => flags.setCachedFlagValue({ slug: "projects_nav", value: true })}
        >
          Saved
        </button>
      </>
    );
  }
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <FlagProvider initialFlags={{ projects_nav: entry }}>
        <Controls />
      </FlagProvider>
    </QueryClientProvider>,
  );
  expect(reads).toBe(0);
  fireEvent.click(screen.getByText("Read"));
  await waitFor(() => expect(reads).toBe(1));
  fireEvent.click(screen.getByText("Saved"));
  await waitFor(() => expect(screen.getByRole("status").textContent).toContain('"projects":true'));
  await act(async () => finishOldRead(response(false)));
  await waitFor(() => expect(screen.getByRole("status").textContent).toContain('"projects":true'));
  expect(JSON.parse(screen.getByTestId("entry").textContent ?? "null")).toEqual({
    value: true,
    description: "Navigation",
    allowEnable: true,
    allowDisable: false,
  });
  expect(reads).toBe(1);
  client.clear();
});
