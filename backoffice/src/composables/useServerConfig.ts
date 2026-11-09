// Backend URL: BACKEND_SERVER_URL from the root .env at build time,
// overridden at runtime by public/config.json when it has a valid URL (boot/config.ts).
export const serverConfig = {
  url: import.meta.env.BACKEND_SERVER_URL || 'http://localhost:4000/',
};
