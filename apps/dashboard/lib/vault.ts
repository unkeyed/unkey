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
  private readonly requestId?: string;
  private readonly fetchFn: typeof fetch;

  constructor(config: {
    baseUrl: string;
    token: string;
    requestId?: string;
    fetchFn?: typeof fetch;
  }) {
    this.baseUrl = config.baseUrl;
    this.token = config.token;
    this.requestId = config.requestId;
    this.fetchFn = config.fetchFn || instrumentedFetch;
  }

  private getHeaders(): HeadersInit {
    const headers: HeadersInit = {
      "Content-Type": "application/json",
      Authorization: `Bearer ${this.token}`,
    };

    if (this.requestId) {
      headers["Unkey-Request-Id"] = this.requestId;
    }

    return headers;
  }

  public async encrypt(req: EncryptRequest): Promise<EncryptResponse> {
    const url = `${this.baseUrl}/vault.v1.VaultService/Encrypt`;
    const res = await this.fetchFn(url, {
      method: "POST",
      headers: this.getHeaders(),
      body: JSON.stringify(req),
    });

    if (!res.ok) {
      const errorText = await res.text();
      throw new Error(`unable to encrypt, fetch error: ${errorText}`);
    }

    const body = (await res.json()) as EncryptResponse;
    return {
      encrypted: body.encrypted,
      keyId: body.keyId,
    };
  }

  public async encryptBulk(req: EncryptBulkRequest): Promise<EncryptBulkResponse> {
    const url = `${this.baseUrl}/vault.v1.VaultService/EncryptBulk`;
    const res = await this.fetchFn(url, {
      method: "POST",
      headers: this.getHeaders(),
      body: JSON.stringify(req),
    });

    if (!res.ok) {
      const errorText = await res.text();
      throw new Error(`unable to encryptBulk, fetch error: ${errorText}`);
    }

    const body = (await res.json()) as EncryptBulkResponse;
    return {
      encrypted: body.encrypted,
    };
  }

  public async decrypt(req: DecryptRequest): Promise<DecryptResponse> {
    const url = `${this.baseUrl}/vault.v1.VaultService/Decrypt`;
    const res = await this.fetchFn(url, {
      method: "POST",
      headers: this.getHeaders(),
      body: JSON.stringify(req),
    });

    if (!res.ok) {
      const errorText = await res.text();
      throw new Error(`unable to decrypt, fetch error: ${errorText}`);
    }

    const body = (await res.json()) as DecryptResponse;
    return {
      plaintext: body.plaintext,
    };
  }
}

export async function instrumentedFetch(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<Response> {
  const url = input instanceof Request ? input.url : input.toString();
  const method = init?.method || (input instanceof Request ? input.method : "GET");

  try {
    const response = await fetch(input, init);

    console.info({
      type: "http_request",
      method,
      url,
      status: response.status,
      timestamp: Date.now(),
    });

    return response;
  } catch (error) {
    console.error({
      type: "http_request_error",
      method,
      url,
      error: error instanceof Error ? error.message : String(error),
      timestamp: Date.now(),
    });

    throw error;
  }
}
