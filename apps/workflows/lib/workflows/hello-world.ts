import { inngest } from "@/lib/inngest";
export const helloWorld = inngest.createFunction(
  { id: "hello-world" },
  { event: "test/hello.world" },
  async ({ event, step }) => {
    await step.sleep("wait-a-moment", "1s");
    return {
      event,
      body: "hello world 4",
    };
  },
);
