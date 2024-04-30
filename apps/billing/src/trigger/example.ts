import { logger, task, wait } from "@trigger.dev/sdk/v3";

export const helloWorld2Task = task({
  id: "hello-world-2",
  run: async (payload: any, { ctx }) => {
    logger.log("Hello, world!", { payload, ctx });

    await wait.for({ seconds: 5 });

    return {
      message: "Hello, world!",
    };
  },
});
