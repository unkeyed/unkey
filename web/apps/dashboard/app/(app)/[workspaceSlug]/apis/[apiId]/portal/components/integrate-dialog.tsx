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
  portalId: string;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
};

export function IntegrateDialog({ portalId, isOpen, onOpenChange }: Props) {
  const curl = `curl -X POST https://api.unkey.com/v2/portal.createSession \\
  -H "Authorization: Bearer $UNKEY_ROOT_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "portalId": "${portalId}",
    "externalId": "user_123",
    "permissions": ["keys.read", "keys.create"]
  }'`;

  const ts = `// server-side, after you've authenticated the user
const { url } = await unkey.portal.createSession({
  portalId: "${portalId}",
  externalId: user.id,
  permissions: ["keys.read", "keys.create"],
});

redirect(url); // send them to their portal`;

  return (
    <DialogContainer isOpen={isOpen} onOpenChange={onOpenChange} title="How to integrate">
      <div className="flex flex-col gap-5">
        <p className="text-gray-11 text-[13px]">
          Create a session for a user you've already signed in, then redirect them to the portal.
          The permissions you pass decide which tabs they see.
        </p>

        <div className="flex flex-col gap-2">
          <p className="text-gray-9 text-[11px] uppercase tracking-wide">
            Step 1 · Create a session (server-side)
          </p>
          <Tabs defaultValue="curl">
            <TabsList>
              <TabsTrigger value="curl">cURL</TabsTrigger>
              <TabsTrigger value="ts">TypeScript</TabsTrigger>
            </TabsList>
            <TabsContent value="curl">
              <Code copyButton={<CopyButton value={curl} />}>{curl}</Code>
            </TabsContent>
            <TabsContent value="ts">
              <Code copyButton={<CopyButton value={ts} />}>{ts}</Code>
            </TabsContent>
          </Tabs>
        </div>

        <div className="flex flex-col gap-2">
          <p className="text-gray-9 text-[11px] uppercase tracking-wide">
            Step 2 · Redirect the user
          </p>
          <Code copyButton={<CopyButton value="redirect(session.url)" />}>
            redirect(session.url)
          </Code>
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
