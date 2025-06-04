type EncryptRequest = {
  keyring: string;
  data: string;
};

type EncryptResponse = {
  encrypted: string;
  keyId: string;
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

  constructor(config: { baseUrl: string; token: string }) {
    this.baseUrl = config.baseUrl;
    this.token = config.token;
  }

  private getHeaders(): HeadersInit {
    const headers: HeadersInit = {
      "Content-Type": "application/json",
      Authorization: `Bearer ${this.token}`,
    };

    return headers;
  }

  public async encrypt(req: EncryptRequest): Promise<EncryptResponse> {
    const url = `${this.baseUrl}/vault.v1.VaultService/Encrypt`;
    const res = await fetch(url, {
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

  public async decrypt(req: DecryptRequest): Promise<DecryptResponse> {
    const url = `${this.baseUrl}/vault.v1.VaultService/Decrypt`;
    const res = await fetch(url, {
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
