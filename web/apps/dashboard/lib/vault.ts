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
  items: Record<string, string>;
};

type EncryptBulkResponseItem = {
  encrypted: string;
  keyId: string;
};

type EncryptBulkResponse = {
  items: Record<string, EncryptBulkResponseItem>;
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

    const body: EncryptResponse = await res.json();
    return {
      encrypted: body.encrypted,
      keyId: body.keyId,
    };
  }

  public async encryptBulk(req: EncryptBulkRequest): Promise<EncryptBulkResponse> {
    const url = `${this.baseUrl}/vault.v1.VaultService/EncryptBulk`;
    const res = await fetch(url, {
      method: "POST",
      headers: this.getHeaders(),
      body: JSON.stringify(req),
    });

    if (!res.ok) {
      const errorText = await res.text();
      throw new Error(`unable to encrypt bulk, fetch error: ${errorText}`);
    }

    const body: EncryptBulkResponse = await res.json();
    return { items: body.items };
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

    const body: DecryptResponse = await res.json();
    return {
      plaintext: body.plaintext,
    };
  }

  private getHeaders(): HeadersInit {
    return {
      "Content-Type": "application/json",
      Authorization: `Bearer ${this.token}`,
    };
  }
}
