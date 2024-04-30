import { logger, task, wait } from "@trigger.dev/sdk/v3";

type Payload = {
  gateway: {
    id: string;
  };
};

export const deployTask = task({
  id: "deploy",
  run: async (payload: Payload, { ctx }) => {
    logger.log("Hello, world!", { payload, ctx });

    await wait.for({ seconds: 5 });

    return {
      message: "Hello, world!",
    };
  },
});
