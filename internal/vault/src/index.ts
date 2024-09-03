export type EncryptRequest = {
  keyring: string;
  data: string;
};

export type EncryptResponse = {
  encrypted: string;
  keyId: string;
};

export type EncryptBulkRequest = {
  keyring: string;
  data: string[];
};

export type EncryptBulkResponse = {
  encrypted: EncryptResponse[];
};

export type DecryptRequest = {
  keyring: string;
  encrypted: string;
};

export type DecryptResponse = {
  plaintext: string;
};

export type RequestContext = {
  requestId: string;
};

export class Vault {
  private readonly baseUrl: string;
  private readonly token: string;

  constructor(baseUrl: string, token: string) {
    this.baseUrl = baseUrl;
    this.token = token;
  }

  private async fetchWithMetrics(
    url: string,
    options: RequestInit,
    _op: string,
  ): Promise<Response> {
    const _start = performance.now();

    const res = await fetch(url, options);

    const _latency = performance.now() - _start;

    return res;
  }

  public async encrypt(c: RequestContext, req: EncryptRequest): Promise<EncryptResponse> {
    const url = `${this.baseUrl}/vault.v1.VaultService/Encrypt`;
    const res = await this.fetchWithMetrics(
      url,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${this.token}`,
          "Unkey-Request-Id": c.requestId,
        },
        body: JSON.stringify(req),
      },
      "encrypt",
    );

    if (!res.ok) {
      throw new Error(`Unable to encrypt, fetch error: ${await res.text()}`);
    }

    return res.json();
  }

  public async encryptBulk(
    c: RequestContext,
    req: EncryptBulkRequest,
  ): Promise<EncryptBulkResponse> {
    const url = `${this.baseUrl}/vault.v1.VaultService/EncryptBulk`;
    const res = await this.fetchWithMetrics(
      url,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${this.token}`,
          "Unkey-Request-Id": c.requestId,
        },
        body: JSON.stringify(req),
      },
      "encryptBulk",
    );

    if (!res.ok) {
      throw new Error(`Unable to encryptBulk, fetch error: ${await res.text()}`);
    }

    return res.json();
  }

  public async decrypt(c: RequestContext, req: DecryptRequest): Promise<DecryptResponse> {
    const url = `${this.baseUrl}/vault.v1.VaultService/Decrypt`;
    const res = await this.fetchWithMetrics(
      url,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${this.token}`,
          "Unkey-Request-Id": c.requestId,
        },
        body: JSON.stringify(req),
      },
      "decrypt",
    );

    if (!res.ok) {
      throw new Error(`Unable to decrypt, fetch error: ${await res.text()}`);
    }

    return res.json();
  }
}
