import { Kafka as KafkaJs, logLevel } from "kafkajs";
import { env } from "./env";
import { z } from "zod";
import { Logger } from "./logger";

const schema = z.object({
  key: z.object({
    id: z.string(),
    hash: z.string(),
  }),
});
export class Kafka {
  private readonly kafka: KafkaJs;
  private readonly logger: Logger;

  constructor(opts: { logger: Logger }) {
    this.logger = opts.logger;
    this.kafka = new KafkaJs({
      logLevel: logLevel.INFO,
      logCreator: (level) => (logEntry) => {
        switch (level) {
          case logLevel.ERROR:
            this.logger.error("kafkajs.log", { logEntry });
            break;
          case logLevel.WARN:
            this.logger.warn("kafkajs.log", { logEntry });
            break;
          case logLevel.INFO:
            this.logger.info("kafkajs.log", { logEntry });
            break;
          case logLevel.DEBUG:
            this.logger.debug("kafkajs.log", { logEntry });

          default:
            break;
        }
      },
      brokers: [env.KAFKA_BROKER],
      sasl: {
        mechanism: "scram-sha-256",
        username: env.KAFKA_USERNAME,
        password: env.KAFKA_PASSWORD,
      },
      ssl: true,
    });

    const admin = this.kafka.admin();
    admin
      .connect()
      .then(async () => {
        this.logger.info("kafka.topics", { topics: await admin.listTopics() });
      })
      .finally(() => {
        admin.disconnect();
      });
  }

  public async onKeyDeleted(callback: (message: z.infer<typeof schema>) => void) {
    const consumerGroupId = `${env.FLY_REGION}-${env.FLY_ALLOC_ID}`;
    this.logger.info("Starting kafka consumer", { consumerGroupId });
    const consumer = this.kafka.consumer({ groupId: consumerGroupId });
    await consumer.connect();
    this.logger.info("Kafka consumer connected", { consumerGroupId });

    const topic = "key.deleted";
    await consumer.subscribe({ topic: "key.deleted" });
    this.logger.info("Kafka consumer subscribed", { consumerGroupId, topic });

    const signalTraps = ["SIGTERM", "SIGINT", "SIGUSR2"];
    signalTraps.forEach((type) => {
      process.once(type, async () => {
        await consumer.disconnect();
      });
    });

    await consumer.run({
      eachMessage: async ({ topic, message }) => {
        this.logger.info("Kafka message received", { consumerGroupId, topic });
        if (!message.value) {
          this.logger.error("Kafka message has no value", { consumerGroupId, topic });
          return;
        }
        const parsed = schema.safeParse(JSON.parse(message.value.toString()));
        if (!parsed.success) {
          this.logger.error("Kafka message could not be parsed", {
            consumerGroupId,
            topic,
            error: parsed.error,
          });
          throw new Error(parsed.error.message);
        }
        this.logger.info("Kafka message parsed", { consumerGroupId, topic, parsed });
        callback(parsed.data);
      },
    });
  }
}
