const buf = crypto.getRandomValues(new Uint8Array(4));

/**
 * epoch starts more recently so that the 32-bit number space gives a
 * significantly higher useful lifetime of around 136 years
 * from 2023-11-14T22:13:20.000Z to 2159-12-22T04:41:36.000Z.
 */
const EPOCH_TIMESTAMP = 1_700_000_000_000;

const t = new Date(2023, 12, 2).getTime() - EPOCH_TIMESTAMP;

buf[0] = (t >>> 24) & 255;
buf[1] = (t >>> 16) & 255;
buf[2] = (t >>> 8) & 255;
buf[3] = t & 255;

console.log(buf);
