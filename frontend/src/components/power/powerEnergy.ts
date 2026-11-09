import { getPowerConsumption } from '../../services/api';

// Energy meters report in watt-minutes (Wm), watt-hours (Wh) or kWh.
export function toKwh(value: number, scale: string) {
  if (scale === 'Wm') return value / 60 / 1000;
  if (scale === 'Wh') return value / 1000;
  return value;
}

export function startOfDay(date = new Date()) {
  const start = new Date(date);
  start.setHours(0, 0, 0, 0);
  return start;
}

export function startOfMonth(date = new Date()) {
  const start = startOfDay(date);
  start.setDate(1);
  return start;
}

export async function getKwh(tagId: number, scale: string, from: Date, to = new Date()) {
  const value = await getPowerConsumption(tagId, from.toISOString(), to.toISOString());
  return toKwh(Number(value), scale);
}
