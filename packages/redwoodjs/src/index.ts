import { Unkey } from "@unkey/api";

import { version } from "../package.json";

export async function doTheMagicHere() {
  const _unkey = new Unkey({
    rootKey: "GET_THIS_FROM_ENV_SOMEHOW",
    wrapperSdkVersion: `@unkey/redwoodjs@${version}`,
    disableTelemetry: false, // TODO: andreas
  });

  // do stuff :)
}
