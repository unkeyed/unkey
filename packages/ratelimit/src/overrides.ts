import { Unkey } from "@unkey/api";

export type OverrideConfig = {
  /**
   * @default https://api.unkey.com
   */
  baseUrl?: string;

  /**
   * The unkey root key. You can create one at https://app.unkey.com/settings/root-keys
   *
   * Make sure the root key has permissions to use overrides.
   */
  rootKey: string;
};

export class Overrides {
  private readonly unkey: Unkey;

  constructor(config: OverrideConfig) {
    this.unkey = new Unkey({
      serverURL: config.baseUrl,
      rootKey: config.rootKey,
    });
  }

  public get getOverride() {
    return this.unkey.ratelimit.getOverride;
  }
  public get setOverride() {
    return this.unkey.ratelimit.setOverride;
  }
  public get deleteOverride() {
    return this.unkey.ratelimit.deleteOverride;
  }
  public get listOverrides() {
    return this.unkey.ratelimit.listOverrides;
  }
}
