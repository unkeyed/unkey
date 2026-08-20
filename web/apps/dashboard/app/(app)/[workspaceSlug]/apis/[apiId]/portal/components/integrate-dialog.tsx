"use client";

import {
  Code,
  CopyButton,
  DialogContainer,
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@unkey/ui";

type Props = {
  /** The portal's slug. `portal.createSession` accepts a portal id or a slug. */
  slug: string;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
};

export function IntegrateDialog({ slug, isOpen, onOpenChange }: Props) {
  const curl = `curl -X POST https://api.unkey.com/v2/portal.createSession \\
  -H "Authorization: Bearer $UNKEY_ROOT_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "portal": "${slug}",
    "externalId": "user_123",
    "scopes": ["keys:read", "analytics:read"]
  }'`;

  const ts = `// server-side, after you've authenticated the user
const { result } = await unkey.portal.createSession({
  portal: "${slug}",
  externalId: user.id,
  scopes: ["keys:read", "analytics:read"],
  // Optional. A URL you control, never one read from the request.
  returnUrl: "https://app.example.com/settings/api-keys",
});

redirect(result.data.url); // send them to their portal`;

  const go = `// server-side, after you've authenticated the user
res, err := client.Portal.CreateSession(ctx, components.V2PortalCreateSessionRequestBody{
	Portal:     "${slug}",
	ExternalID: user.ID,
	Scopes: []components.Scope{
		components.ScopeKeysRead,
		components.ScopeAnalyticsRead,
	},
	// Optional. A URL you control, never one read from the request.
	ReturnURL: unkey.String("https://app.example.com/settings/api-keys"),
})
if err != nil {
	return err
}

http.Redirect(w, r, res.V2PortalCreateSessionResponseBody.Data.URL, http.StatusFound)`;

  return (
    <DialogContainer isOpen={isOpen} onOpenChange={onOpenChange} title="How to integrate">
      <div className="flex flex-col gap-5">
        <p className="text-gray-11 text-[13px]">
          Create a session for a user you've already signed in, then redirect them to the portal.
          The scopes you pass decide which tabs they see: any <code>keys:</code> scope shows the
          keys tab, <code>analytics:read</code> shows analytics. Pick from <code>keys:read</code>,{" "}
          <code>keys:create</code>, <code>keys:reroll</code>, and <code>analytics:read</code>.
        </p>

        <div className="flex flex-col gap-2">
          <p className="text-gray-9 text-[11px] uppercase tracking-wide">
            Step 1 · Create a session (server-side)
          </p>
          <Tabs defaultValue="curl">
            <TabsList>
              <TabsTrigger value="curl">cURL</TabsTrigger>
              <TabsTrigger value="ts">TypeScript</TabsTrigger>
              <TabsTrigger value="go">Go</TabsTrigger>
            </TabsList>
            <TabsContent value="curl">
              <Code copyButton={<CopyButton value={curl} />}>{curl}</Code>
            </TabsContent>
            <TabsContent value="ts">
              <Code copyButton={<CopyButton value={ts} />}>{ts}</Code>
            </TabsContent>
            <TabsContent value="go">
              <Code copyButton={<CopyButton value={go} />}>{go}</Code>
            </TabsContent>
          </Tabs>
        </div>

        <div className="flex flex-col gap-2">
          <p className="text-gray-9 text-[11px] uppercase tracking-wide">
            Step 2 · Redirect the user
          </p>
          <Code copyButton={<CopyButton value="redirect(result.data.url)" />}>
            redirect(result.data.url)
          </Code>
          <p className="text-gray-11 text-[13px]">
            The returned URL carries a single-use exchange code that is valid for 15 minutes.
          </p>
        </div>

        <div className="flex flex-col gap-2">
          <p className="text-gray-9 text-[11px] uppercase tracking-wide">Return URL (optional)</p>
          <p className="text-gray-11 text-[13px]">
            <code>returnUrl</code> is set per session, not on the portal, so one portal can send
            each user back to the page they came from. It must be an absolute <code>https://</code>{" "}
            URL you control, at most 500 characters. Never take it from the incoming request: wiring
            it from something like a <code>?next=</code> parameter turns your sign-in flow into an
            open redirect for your own users. Omit it and the portal shows no return link.
          </p>
        </div>

        <a
          href="https://www.unkey.com/docs"
          target="_blank"
          rel="noopener noreferrer"
          className="text-accent-11 text-[13px] underline"
        >
          Full documentation →
        </a>
      </div>
    </DialogContainer>
  );
}
