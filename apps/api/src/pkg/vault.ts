import type { Context } from "./hono/app";
import type { Metrics } from "./metrics";
import { instrumentedFetch } from "./util/instrument-fetch";

type EncryptRequest = {
  keyring: string;
  data: string;
};

type EncryptResponse = {
  encrypted: string;
  keyId: string;
};

type EncryptBulkRequest = {
  keyring: string;
  data: string[];
};

type EncryptBulkResponse = {
  encrypted: EncryptResponse[];
};

type DecryptRequest = {
  keyring: string;
  encrypted: string;
};

type DecryptResponse = {
  plaintext: string;
};

export class Vault {
  private readonly baseUrl: string;
  private readonly token: string;
  private readonly metrics: Metrics;

  constructor(baseUrl: string, token: string, metrics: Metrics) {
    this.baseUrl = baseUrl;
    this.token = token;
    this.metrics = metrics;
  }

  public async encrypt(c: Context, req: EncryptRequest): Promise<EncryptResponse> {
    const start = performance.now();

    const url = `${this.baseUrl}/vault.v1.VaultService/Encrypt`;
    const res = await instrumentedFetch(c)(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${this.token}`,
        "Unkey-Request-Id": c.get("requestId"),
      },
      body: JSON.stringify(req),
    });

    if (!res.ok) {
      throw new Error(`unable to encrypt, fetch error: ${await res.text()}`);
    }
    const body = await res.json<EncryptResponse>();

    this.metrics.emit({
      metric: "metric.agent.latency",
      op: "encrypt",
      latency: performance.now() - start,
    });
    return {
      encrypted: body.encrypted,
      keyId: body.keyId,
    };
  }
  public async encryptBulk(c: Context, req: EncryptBulkRequest): Promise<EncryptBulkResponse> {
    const start = performance.now();
    const url = `${this.baseUrl}/vault.v1.VaultService/EncryptBulk`;
    const res = await instrumentedFetch(c)(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${this.token}`,
        "Unkey-Request-Id": c.get("requestId"),
      },
      body: JSON.stringify(req),
    });

    if (!res.ok) {
      throw new Error(`unable to encryptBulk, fetch error: ${await res.text()}`);
    }
    const body = await res.json<EncryptBulkResponse>();

    this.metrics.emit({
      metric: "metric.agent.latency",
      op: "encrypt",
      latency: performance.now() - start,
    });
    return {
      encrypted: body.encrypted,
    };
  }

  public async decrypt(c: Context, req: DecryptRequest): Promise<DecryptResponse> {
    const start = performance.now();

    const res = await instrumentedFetch(c)(`${this.baseUrl}/vault.v1.VaultService/Decrypt`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${this.token}`,
        "Unkey-Request-Id": c.get("requestId"),
      },
      body: JSON.stringify(req),
    });
    if (!res.ok) {
      throw new Error(`unable to decrypt, fetch error: ${await res.text()}`);
    }
    const body = await res.json<DecryptResponse>();

    this.metrics.emit({
      metric: "metric.agent.latency",
      op: "decrypt",
      latency: performance.now() - start,
    });
    return {
      plaintext: body.plaintext,
    };
  }
}
