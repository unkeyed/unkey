export class Loops {
  private readonly apiKey: string;
  private readonly baseUrl: string;

  constructor(opts: { apiKey: string }) {
    this.apiKey = opts.apiKey;
    this.baseUrl = "https://app.loops.so/api";
  }

  private async fetch<TResult>(req: {
    path: string[];
    method: "GET" | "POST" | "PUT" | "DELETE";
    body?: unknown;
  }): Promise<TResult> {
    const url = `${this.baseUrl}/${req.path.join("/")}`;

    const res = await fetch(url, {
      method: req.method,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${this.apiKey}`,
      },
      body: JSON.stringify(req.body),
    });
    if (res.ok) {
      return await res.json();
    }
    throw new Error(`error from loops api: ${await res.text()}`);
  }

  public async sendTrialEnds(req: { email: string; name: string; date: Date }): Promise<void> {
    await this.fetch({
      path: ["v1", "transactional"],
      method: "POST",
      body: {
        transactionalId: "cllt7wzoc00rqmh0ohcumn2c4",
        email: req.email,
        dataVariables: {
          name: req.name,
          date: req.date.toDateString(),
        },
      },
    });
  }
}
