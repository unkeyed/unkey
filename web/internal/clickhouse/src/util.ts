import { z } from "zod";
// clickhouse DateTime returns a string, which we need to parse
export const dateTimeToUnix = z.string().transform((t) => new Date(t).getTime());

export function normalizeTimeRange(startTime: number, endTime: number): {
	startTime: number;
	endTime: number;
} {
	if (startTime <= endTime) {
		return { startTime, endTime };
	}
	return { startTime: endTime, endTime: startTime };
}
