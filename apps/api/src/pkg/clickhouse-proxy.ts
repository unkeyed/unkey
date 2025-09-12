/**
 * HTTP client for the ClickHouse proxy service
 * Sends batched events to the internal Go API proxy endpoints
 */
export class ClickHouseProxyClient {
  private readonly baseUrl: string;
  private readonly token: string;

  constructor(baseUrl: string, token: string) {
    this.baseUrl = baseUrl.replace(/\/$/, ""); // Remove trailing slash
    this.token = token;
  }

  /**
   * Insert key verification events
   */
  async insertVerifications(
    events: Array<{
      request_id: string;
      time: number;
      workspace_id: string;
      key_space_id: string;
      key_id: string;
      region: string;
      outcome: string;
      identity_id?: string;
      tags?: string[];
    }>,
  ): Promise<void> {
    await this.sendEvents("/_internal/chproxy/verifications", events);
  }

  /**
   * Insert API request metric events
   */
  async insertApiRequests(
    events: Array<{
      workspace_id: string;
      request_id: string;
      time: number;
      host: string;
      method: string;
      path: string;
      request_headers: string[];
      request_body: string;
      response_status: number;
      response_headers: string[];
      response_body: string;
      error: string;
      service_latency: number;
      user_agent: string;
      ip_address: string;
      region: string;
    }>,
  ): Promise<void> {
    await this.sendEvents("/_internal/chproxy/metrics", events);
  }

  /**
   * Insert ratelimit events
   */
  async insertRatelimits(
    events: Array<{
      request_id: string;
      time: number;
      workspace_id: string;
      namespace_id: string;
      identifier: string;
      passed: boolean;
    }>,
  ): Promise<void> {
    await this.sendEvents("/_internal/chproxy/ratelimits", events);
  }

  /**
   * Generic method to send events to any endpoint
   */
  private async sendEvents(endpoint: string, events: unknown[]): Promise<void> {
    if (events.length === 0) {
      return;
    }

    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${this.token}`,
        "X-Unkey-Metrics": "disabled",
      },
      body: JSON.stringify(events),
    });

    if (!response.ok) {
      const errorText = await response.text();
      throw new Error(
        `Failed to send events to ${endpoint}: ${response.status} ${response.statusText} - ${errorText}`,
      );
    }
  }
}
