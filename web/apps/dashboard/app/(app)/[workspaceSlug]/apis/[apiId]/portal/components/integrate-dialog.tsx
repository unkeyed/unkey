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
    "scopes": ["keys:read", "keys:reroll"]
  }'`;

  const ts = `// on your server, once the user is signed in
const { result } = await unkey.portal.createSession({
  portal: "${slug}",
  externalId: user.id,
  scopes: ["keys:read", "keys:reroll"],
  // Optional. Your own URL, never one from the request.
  returnUrl: "https://app.example.com/settings/api-keys",
});

redirect(result.data.url);`;

  const go = `// on your server, once the user is signed in
body := components.V2PortalCreateSessionRequestBody{
	Portal:     "${slug}",
	ExternalID: user.ID,
	Scopes: []components.Scope{
		components.ScopeKeysRead,
		components.ScopeKeysReroll,
	},
	// Optional. Your own URL, never one from the request.
	ReturnURL: unkey.String("https://app.example.com/settings/api-keys"),
}

res, err := client.Portal.CreateSession(ctx, body)
if err != nil {
	return err
}

url := res.V2PortalCreateSessionResponseBody.Data.URL
http.Redirect(w, r, url, http.StatusFound)`;

  return (
    <DialogContainer isOpen={isOpen} onOpenChange={onOpenChange} title="How to integrate">
      <div className="flex flex-col gap-5">
        <p className="text-gray-11 text-[13px]">
          Sign the user in yourself, create a session for them, then send them to the portal. Scopes
          decide what the user can do there. The two you can pass are <code>keys:read</code> and{" "}
          <code>keys:reroll</code>.
        </p>

        <div className="flex flex-col gap-2">
          <p className="text-gray-9 text-[11px] uppercase tracking-wide">
            Step 1 · Create a session
          </p>
          <Tabs defaultValue="curl">
            <TabsList>
              <TabsTrigger value="curl">cURL</TabsTrigger>
              <TabsTrigger value="ts">TypeScript</TabsTrigger>
              <TabsTrigger value="go">Go</TabsTrigger>
            </TabsList>
            <TabsContent value="curl">
              <Code preClassName="min-w-0" copyButton={<CopyButton value={curl} />}>
                {curl}
              </Code>
            </TabsContent>
            <TabsContent value="ts">
              <Code preClassName="min-w-0" copyButton={<CopyButton value={ts} />}>
                {ts}
              </Code>
            </TabsContent>
            <TabsContent value="go">
              <Code preClassName="min-w-0" copyButton={<CopyButton value={go} />}>
                {go}
              </Code>
            </TabsContent>
          </Tabs>
        </div>

        <div className="flex flex-col gap-2">
          <p className="text-gray-9 text-[11px] uppercase tracking-wide">
            Step 2 · Send them to the portal
          </p>
          <Code
            preClassName="min-w-0"
            copyButton={<CopyButton value="redirect(result.data.url)" />}
          >
            redirect(result.data.url)
          </Code>
          <p className="text-gray-11 text-[13px]">
            That URL carries a one-time exchange code. It expires after 15 minutes.
          </p>
        </div>

        <div className="flex flex-col gap-2">
          <p className="text-gray-9 text-[11px] uppercase tracking-wide">Optional return URL</p>
          <p className="text-gray-11 text-[13px]">
            Pass <code>returnUrl</code> per session rather than setting it on the portal, so each
            user lands back where they started. It has to be an absolute <code>https://</code> URL
            you control, 500 characters or fewer. Never take it from the incoming request. A{" "}
            <code>?next=</code> value your user picked is an open redirect. Leave it out and the
            portal shows no return link.
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
