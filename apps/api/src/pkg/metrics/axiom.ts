import { Axiom } from "@axiomhq/js";
import type { Metric } from "@unkey/metrics";
import type { Env } from "../env";
import type { Metrics } from "./interface";
export class AxiomMetrics implements Metrics {
  private readonly axiomDataset: string;
  private readonly ax: Axiom;
  private readonly defaultFields: Record<string, unknown>;

  /**
   * @param opts.axiomToken The token to use to authenticate with axiom
   * @param opts.defaultFields Any additional defaultFields to add to the metrics by default
   */
  constructor(opts: {
    axiomToken: string;
    defaultFields?: Record<string, unknown>;
    environment: Env["ENVIRONMENT"];
  }) {
    this.axiomDataset = `cf_api_metrics_${opts.environment}`;
    this.ax = new Axiom({
      token: opts.axiomToken,
    });
    this.defaultFields = opts.defaultFields ?? {};
  }

  public emit(metric: Metric): void {
    this.ax.ingest(this.axiomDataset, [
      {
        _time: Date.now(),
        ...this.defaultFields,
        metric,
      },
    ]);
  }

  /**
   * flush sends the metrics to axiom
   *
   * Call this at the end of the request handler with .waitUntil()
   */

  public async flush(): Promise<void> {
    await this.ax.flush().catch((err) => {
      console.error("unable to flush logs to axiom", err);
    });
  }
}
