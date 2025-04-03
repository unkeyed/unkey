import { Unkey } from "@unkey/api";

export type OverrideConfig = {
  /**
   * @default https://api.unkey.com
   */
  baseUrl?: string;

  /**
   * The unkey root key. You can create one at https://unkey.dev/app/settings/root-keys
   *
   * Make sure the root key has permissions to use overrides.
   */
  rootKey: string;
};

export class Overrides {
  private readonly unkey: Unkey;

  constructor(config: OverrideConfig) {
    this.unkey = new Unkey({
      baseUrl: config.baseUrl,
      rootKey: config.rootKey,
    });
  }

  public get getOverride() {
    return this.unkey.ratelimits.getOverride;
  }
  public get setOverride() {
    return this.unkey.ratelimits.setOverride;
  }
  public get deleteOverride() {
    return this.unkey.ratelimits.deleteOverride;
  }
  public get listOverrides() {
    return this.unkey.ratelimits.listOverrides;
  }
}
