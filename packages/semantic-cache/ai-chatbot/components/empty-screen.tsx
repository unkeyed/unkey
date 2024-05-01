import type { UseChatHelpers } from "ai/react";

import { Button } from "@/components/ui/button";
import { IconArrowRight } from "@/components/ui/icons";

const exampleMessages = [
  {
    heading: "Explain technical concepts",
    message: `What is a "serverless function"?`,
  },
  {
    heading: "Summarize an article",
    message: "Summarize the following article for a 2nd grader: \n",
  },
  {
    heading: "Draft an email",
    message: "Draft an email to my boss about the following: \n",
  },
];

export function EmptyScreen({ setInput }: Pick<UseChatHelpers, "setInput">) {
  return (
    <div className="mx-auto max-w-2xl px-4">
      <div className="rounded-lg border bg-background p-8">
        <h1 className="mb-2 text-lg font-semibold">Unkey Semantic Caching demo</h1>
        <p className="leading-normal text-muted-foreground">
          Try prompting the chatbot with an in-depth question. The first time you ask, it will be
          streamed as normal. Subsequent times, it will be served from the cache near-instantly.
        </p>
        <p className="leading-normal text-muted-foreground mt-2">
          Because the caching is semantic, identical questions with different phrasing will return
          the same answer from the cache.
        </p>
        <div className="mt-4 flex flex-col items-start space-y-2">
          {exampleMessages.map((message) => (
            <Button
              key={message.heading}
              variant="link"
              className="h-auto p-0 text-base"
              onClick={() => setInput(message.message)}
            >
              <IconArrowRight className="mr-2 text-muted-foreground" />
              {message.heading}
            </Button>
          ))}
        </div>
      </div>
    </div>
  );
}
