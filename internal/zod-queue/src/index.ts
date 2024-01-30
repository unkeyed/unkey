import type { MessageBatch, Queue, QueueSendOptions } from "@cloudflare/workers-types";
import { z } from "zod";

export type Config<TSchema extends z.ZodSchema<any>> = {
  schema: TSchema;
  queue: Queue<z.input<TSchema>[]>;
  queueSendOptions?: QueueSendOptions;
};

/**
 * BufferQueue is a utility to send multiple messages at different times as a single message.
 *
 * Often you want to send a message in different parts of the code, but don't want to keep track of
 * other messages. At the same time it's cheaper to send a single array message than multiple
 * individual messages.
 *
 * This is the solution.
 *
 * use `.buffer(m)` to buffer as many messages as you want through your code.
 *
 * before quitting the worker handler, call `await .flush()` to send all the messages as a single
 * item to the queue.
 *
 * On the other side, you can use `.unpack` to unpack the array into individual messages again.
 */
export class BufferQueue<TSchema extends z.ZodType<any>, TMessage = z.input<TSchema>> {
  private readonly schema: TSchema;
  private readonly queue: Queue<TMessage[]>;
  private readonly queueSendOptions?: QueueSendOptions;
  private messages: TMessage[] = [];

  constructor(config: Config<TSchema>) {
    this.schema = config.schema;
    this.queue = config.queue;
    this.queueSendOptions = config.queueSendOptions;
  }

  /**
   * Buffer the message. Use `.flush()` to send to the queue and await all acks later.
   */
  buffer(message: TMessage): void {
    this.messages.push(this.schema.parse(message));
  }

  /**
   * Blocks until all remaining messages are sent to the queue
   */
  async flush(): Promise<void> {
    await this.queue.send(this.messages, this.queueSendOptions);
    this.messages = [];
  }

  /**
   *
   * unpack extracts the buffered messages and returns them as one single array
   */
  unpack(
    batch: MessageBatch<TMessage[]>,
  ): Omit<MessageBatch<TMessage[]>, "messages"> & { messages: TMessage[] } {
    const { messages, ...rest } = batch;
    const parsedMessages: TMessage[] = [];
    for (const m of messages) {
      parsedMessages.push(...z.array(this.schema).parse(m.body));
    }

    return {
      messages: parsedMessages,
      ...rest,
    };
  }
}
