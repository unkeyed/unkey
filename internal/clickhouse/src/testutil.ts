import { execa } from "execa";
import { GenericContainer, Network, type StartedTestContainer } from "testcontainers";
import type { TaskContext } from "vitest";

export class ClickHouseContainer {
  static readonly username = "default";
  static readonly password = "password";
  private readonly container: StartedTestContainer;

  private constructor(container: StartedTestContainer) {
    this.container = container;
  }

  public url(): string {
    return `http://${ClickHouseContainer.username}:${
      ClickHouseContainer.password
    }@${this.container.getHost()}:${this.container.getMappedPort(8123)}`;
  }

  // Stops the container
  //
  // You do not need to call this every time.
  // The container is automatically stopped when the test ends
  public async stop(): Promise<void> {
    await this.container.stop();
  }

  static async start(
    t: TaskContext,
    opts?: { keepContainer?: boolean },
  ): Promise<ClickHouseContainer> {
    const network = await new Network().start();

    const container = await new GenericContainer("bitnami/clickhouse:latest")
      .withEnvironment({
        CLICKHOUSE_ADMIN_USER: ClickHouseContainer.username,
        CLICKHOUSE_ADMIN_PASSWORD: ClickHouseContainer.password,
      })
      .withNetworkMode(network.getName())
      .withExposedPorts(8123, 9000)
      .start();
    if (!opts?.keepContainer) {
      t.onTestFinished(async () => {
        await container.stop();
      });
    }

    const dsn = `tcp://${ClickHouseContainer.username}:${
      ClickHouseContainer.password
    }@localhost:${container.getMappedPort(9000)}`;

    await execa("goose", ["-dir=./schema", "clickhouse", dsn, "up"]);

    return new ClickHouseContainer(container);
  }
}
