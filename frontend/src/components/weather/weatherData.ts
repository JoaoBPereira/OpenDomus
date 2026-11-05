import { temperatureColor } from '../temperature/temperatureData';

// Outdoor temperatures: blue at 8° or below, grey at 16°, orange from 24°.
const SCALE_MIN = 8;
const SCALE_MAX = 24;

export function weatherColor(temp: number) {
  return temperatureColor(temp, SCALE_MIN, SCALE_MAX);
}

// Night is 21:00 to 06:59.
export function isNight(time: Date) {
  const hour = time.getHours();
  return hour >= 21 || hour <= 6;
}
