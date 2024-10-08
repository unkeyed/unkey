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

  public async encrypt(c: RequestContext, req: EncryptRequest): Promise<EncryptResponse> {
    const url = `${this.baseUrl}/vault.v1.VaultService/Encrypt`;
    const res = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${this.token}`,
        "Unkey-Request-Id": c.requestId,
      },
      body: JSON.stringify(req),
    });

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
    const res = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${this.token}`,
        "Unkey-Request-Id": c.requestId,
      },
      body: JSON.stringify(req),
    });

    if (!res.ok) {
      throw new Error(`Unable to encryptBulk, fetch error: ${await res.text()}`);
    }

    return res.json();
  }

  public async decrypt(c: RequestContext, req: DecryptRequest): Promise<DecryptResponse> {
    const url = `${this.baseUrl}/vault.v1.VaultService/Decrypt`;
    const res = await fetch(url, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${this.token}`,
        "Unkey-Request-Id": c.requestId,
      },
      body: JSON.stringify(req),
    });

    if (!res.ok) {
      throw new Error(`Unable to decrypt, fetch error: ${await res.text()}`);
    }

    return res.json();
  }
}
