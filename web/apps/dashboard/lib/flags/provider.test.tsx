import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { act, cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, beforeEach, expect, it, vi } from "vitest";
import { FlagProvider, useFlag, useFlags } from "./provider";

vi.mock("next/navigation", () => ({ usePathname: () => "/workspace/settings/platform-features" }));

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

it("shares one authenticated API read and preserves explicit false before code fallbacks", async () => {
  const requests: Request[] = [];
  vi.stubGlobal("fetch", async (request: Request) => {
    requests.push(request);
    return new Response(
      JSON.stringify({
        meta: { requestId: crypto.randomUUID() },
        data: {
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
        },
      }),
      { headers: { "Content-Type": "application/json" } },
    );
  });
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <FlagProvider orgId={crypto.randomUUID()} userId={crypto.randomUUID()} impersonating={false}>
        <Consumer />
        <Consumer />
      </FlagProvider>
    </QueryClientProvider>,
  );
  await waitFor(() =>
    expect(screen.getAllByRole("status").map((node) => node.textContent)).toEqual([
      '{"billing":false,"projects":true,"upgrades":true}',
      '{"billing":false,"projects":true,"upgrades":true}',
    ]),
  );
  expect(requests).toHaveLength(1);
  expect(new URL(requests[0].url).pathname).toBe("/proxy/v2/workspace.getFlags");
  expect(requests[0].headers.has("Authorization")).toBe(false);
  client.clear();
});

it("isolates organizations while loading, reports read errors, and never fetches signed-out flags", async () => {
  let fail = false;
  const fetcher = vi.fn(async () =>
    fail
      ? new Response(null, { status: 403 })
      : new Response(
          JSON.stringify({
            meta: { requestId: crypto.randomUUID() },
            data: {
              projects_nav: {
                value: true,
                description: "Navigation",
                allowEnable: true,
                allowDisable: true,
              },
            },
          }),
          { headers: { "Content-Type": "application/json" } },
        ),
  );
  vi.stubGlobal("fetch", fetcher);
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const userId = crypto.randomUUID();
  function State() {
    const flags = useFlags();
    return (
      <>
        <Consumer />
        {flags.isLoading && <span>Loading</span>}
        {flags.error && <span role="alert">Read failed</span>}
      </>
    );
  }
  function Tree({ orgId }: { orgId: string | null }) {
    return (
      <QueryClientProvider client={client}>
        <FlagProvider orgId={orgId} userId={orgId ? userId : null} impersonating={false}>
          <State />
        </FlagProvider>
      </QueryClientProvider>
    );
  }
  const view = render(<Tree orgId={crypto.randomUUID()} />);
  await waitFor(() => expect(screen.getByRole("status").textContent).toContain('"projects":true'));
  fail = true;
  view.rerender(<Tree orgId={crypto.randomUUID()} />);
  expect(screen.getByRole("status").textContent).toBe(
    '{"billing":true,"projects":false,"upgrades":true}',
  );
  expect(await screen.findByRole("alert")).toHaveProperty("textContent", "Read failed");
  expect(fetcher).toHaveBeenCalledTimes(2);
  view.rerender(<Tree orgId={null} />);
  await act(async () => {});
  expect(fetcher).toHaveBeenCalledTimes(2);
  expect(screen.queryByRole("alert")).toBeNull();
  client.clear();
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
    return reads === 2 ? oldRead : response(reads > 2);
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
          onClick={() => flags.applyUpdate({ slug: "projects_nav", value: true })}
        >
          Saved
        </button>
      </>
    );
  }
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  render(
    <QueryClientProvider client={client}>
      <FlagProvider orgId={crypto.randomUUID()} userId={crypto.randomUUID()} impersonating={false}>
        <Controls />
      </FlagProvider>
    </QueryClientProvider>,
  );
  await waitFor(() => expect(reads).toBe(1));
  fireEvent.click(screen.getByText("Read"));
  await waitFor(() => expect(reads).toBe(2));
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
  expect(reads).toBe(2);
  client.clear();
});
