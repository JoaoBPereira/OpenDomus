import { getLatestTagValueFloat, getTagValueFloat, type TagValueFloat } from '../../services/api';
import {
  TAG_TEMP_EDUARDO_ROOM,
  TAG_TEMP_FILIPE_ROOM,
  TAG_TEMP_HALLWAY,
  TAG_TEMP_KIDS_OFFICE,
  TAG_TEMP_LIVING_ROOM,
  TAG_TEMP_MAIN_ROOM,
  TAG_TEMP_MEN_CAVE,
} from '../../constants/tagIds';

export interface Room {
  area: string;
  name: string;
  tagId?: string;
  shared?: boolean;
}

export interface TemperatureReading {
  value?: number;
  timestamp?: Date;
  delta?: number;
  stale?: boolean;
  error?: string;
  historyError?: string;
}

export type TemperatureReadings = Partial<Record<string, TemperatureReading>>;

export const groundRooms: Room[] = [
  { area: 'office', name: 'Kids Office', tagId: TAG_TEMP_KIDS_OFFICE },
  { area: 'living', name: 'Living Room', tagId: TAG_TEMP_LIVING_ROOM },
  { area: 'kitchen', name: 'Kitchen', tagId: TAG_TEMP_LIVING_ROOM, shared: true },
];

export const firstRooms: Room[] = [
  { area: 'main', name: 'Main Room', tagId: TAG_TEMP_MAIN_ROOM },
  { area: 'eduardo', name: 'Eduardo Room', tagId: TAG_TEMP_EDUARDO_ROOM },
  { area: 'filipe', name: 'Filipe Room', tagId: TAG_TEMP_FILIPE_ROOM },
  { area: 'wc', name: 'WC' },
  { area: 'cave', name: 'ManCave', tagId: TAG_TEMP_MEN_CAVE },
  { area: 'hallway', name: 'Hallway', tagId: TAG_TEMP_HALLWAY },
];

const HOUR = 60 * 60 * 1000;
const TOLERANCE = 15 * 60 * 1000;

const COOL_RGB = [107, 140, 255];
const NEUTRAL_RGB = [128, 128, 136];
const WARM_RGB = [255, 146, 58];
const BASE_RGB = [40, 40, 44];
const TONE_STRENGTH = 0.3;

function mix(from: number[], to: number[], amount: number) {
  return from.map((channel, index) => Math.round(channel + (to[index] - channel) * amount));
}

function scaleRgb(value: number, min: number, max: number) {
  const position = Math.min(Math.max((value - min) / (max - min), 0), 1);
  const [from, to, amount] = position < 0.5 ? [COOL_RGB, NEUTRAL_RGB, position * 2] : [NEUTRAL_RGB, WARM_RGB, (position - 0.5) * 2];
  return mix(from, to, amount);
}

// The dashboard's temperature scale at full strength, for text, lines and dots.
export function temperatureColor(value: number, min: number, max: number) {
  return `rgb(${scaleRgb(value, min, max).join(', ')})`;
}

// Blue at min, neutral grey halfway, orange at max, blended into a dark base.
// Solid colors keep overlapping tiles, like Kitchen over Living Room, the same shade.
export function temperatureTone(value: number, min: number, max: number) {
  const [red, green, blue] = mix(BASE_RGB, scaleRgb(value, min, max), TONE_STRENGTH);
  return `rgb(${red}, ${green}, ${blue})`;
}

function parseReading(raw: TagValueFloat): { value: number; timestamp: Date } {
  const timestamp = new Date(raw.timestamp ?? raw.time);
  if (!Number.isFinite(raw.value) || !Number.isFinite(timestamp.getTime())) {
    throw new Error('Invalid temperature reading');
  }
  return { value: raw.value, timestamp };
}

export function hourlyDelta(value: number, timestamp: Date, history: TagValueFloat[], now: number): number | undefined {
  // An old "latest" reading must not look like a live hourly trend.
  if (Math.abs(now - timestamp.getTime()) > TOLERANCE) return undefined;
  const target = now - HOUR;
  let nearest: { value: number; distance: number } | undefined;
  for (const raw of history) {
    const time = Date.parse(raw.timestamp ?? raw.time);
    const distance = Math.abs(time - target);
    if (Number.isFinite(raw.value) && distance <= TOLERANCE && (!nearest || distance < nearest.distance)) {
      nearest = { value: raw.value, distance };
    }
  }
  return nearest ? Math.round((value - nearest.value) * 10) / 10 : undefined;
}

export function formatDelta(delta?: number) {
  if (delta === undefined) return '\u2014';
  return `${delta > 0 ? '+' : ''}${delta.toFixed(1)}\u00b0`;
}

export async function loadTemperature(tagId: string, now: number): Promise<TemperatureReading> {
  const [latest, history] = await Promise.allSettled([
    getLatestTagValueFloat(tagId),
    getTagValueFloat(tagId, new Date(now - HOUR - TOLERANCE).toISOString(), new Date().toISOString()),
  ]);
  if (latest.status === 'rejected') {
    return { error: 'Temperature unavailable. Retrying every minute.' };
  }
  let reading: { value: number; timestamp: Date };
  try {
    reading = parseReading(latest.value);
  } catch {
    return { error: 'Invalid temperature data. Retrying every minute.' };
  }
  const stale = now - reading.timestamp.getTime() > 24 * HOUR;
  const historyAvailable = history.status === 'fulfilled' && Array.isArray(history.value);
  return {
    ...reading,
    stale,
    delta: history.status === 'fulfilled' && historyAvailable ? hourlyDelta(reading.value, reading.timestamp, history.value, now) : undefined,
    historyError: !historyAvailable ? 'Hourly history unavailable. Retrying every minute.' : undefined,
  };
}
