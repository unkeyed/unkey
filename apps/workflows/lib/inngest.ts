import { EventSchemas, Inngest } from "inngest";

type Events = {
  // biome-ignore lint/complexity/noBannedTypes: inngest doesn't like never or unknown
  "billing/invoicing": {};
  "billing/create.invoice": {
    name: "billing/create.invoice";
    data: {
      year: number;
      month: number;
      workspaceId: string;
    };
  };
};

export const inngest = new Inngest({
  id: "workflows",
  schemas: new EventSchemas().fromRecord<Events>(),
});
