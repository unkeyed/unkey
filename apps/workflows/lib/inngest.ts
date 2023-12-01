import { Inngest, EventSchemas } from "inngest";
import { type Workspace } from "./db";



type Events = {
  "billing/create.invoice": {
    data: {
      year: number;
      month: number;
      workspace: Workspace
    }
  }
}

export const inngest = new Inngest({ id: "workflows", schemas: new EventSchemas().fromRecord<Events>() });
