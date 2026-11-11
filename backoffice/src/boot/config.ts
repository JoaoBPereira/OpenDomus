import { defineBoot } from '#q-app';
import { serverConfig } from '@/composables/useServerConfig';

type RuntimeConfig = {
  serverUrl?: unknown;
};

function isHttpUrl(value: unknown): value is string {
  if (typeof value !== 'string') return false;
  try {
    const { protocol } = new URL(value);
    return protocol === 'http:' || protocol === 'https:';
  } catch {
    return false;
  }
}

export default defineBoot(async () => {
  try {
    const res = await fetch(`${import.meta.env.BASE_URL}config.json`, { cache: 'no-store' });
    if (!res.ok) return;

    const config = (await res.json()) as RuntimeConfig;

    // Only a valid http(s) URL replaces the default.
    if (isHttpUrl(config.serverUrl)) serverConfig.url = config.serverUrl;
  } catch {
    // ignore config load errors
  }
});
