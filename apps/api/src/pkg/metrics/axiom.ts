import { Metrics, Metric } from "./interface"




export class AxiomMetrics implements Metrics {
  private readonly axiomDataset: string
  private readonly axiomToken: string
  private readonly defaultFields: Record<string, unknown>
  private buffer: unknown[] = []

  /**
 * @param opts.axiomToken The token to use to authenticate with axiom
 * @param opts.defaultFields Any additional defaultFields to add to the metrics by default
  */
  constructor(opts: { axiomToken: string, defaultFields?: Record<string, unknown> }) {
    this.axiomDataset = "cf_api_metrics"
    this.axiomToken = opts.axiomToken
    this.defaultFields = opts.defaultFields ?? {}

  }



  public emit<TMetric extends keyof Metric>(metric: TMetric, e: Metric[TMetric]): void {
    this.buffer.push({
      _time: Date.now(),
      ...this.defaultFields,
      metric,
      ...e,
    })

  }

  /**
 * flush sends the metrics to axiom
 *
 * Call this at the end of the request handler with .waitUntil()
  */
  public async flush(): Promise<void> {
    const copy = this.buffer.slice()
    this.buffer = []
    await fetch(`https://api.axiom.co/v1/datasets/${this.axiomDataset}/ingest`, {
      method: "POST",
      body: JSON.stringify(copy),
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${this.axiomToken}`,
      }
    }).catch(err => {
      console.error("unable to ingest to axiom", err)
    })
  }
}
