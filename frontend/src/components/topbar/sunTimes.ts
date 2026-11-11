// Sunrise and sunset from the standard sunrise equation (NOAA-style approximation, about a
// minute off), so no API or library is needed. Longitude is east-positive.
const RAD = Math.PI / 180;
const J1970 = 2440587.5;
const J2000 = 2451545;

const toJulian = (date: Date) => date.getTime() / 86400000 + J1970;
const fromJulian = (julian: number) => new Date((julian - J1970) * 86400000);

// Returns undefined during polar day or polar night, when the sun does not cross the horizon.
export function sunTimes(day: Date, lat: number, lon: number): { sunrise: Date; sunset: Date } | undefined {
  // Local noon picks the right calendar day whatever the time and the timezone.
  const noon = new Date(day.getFullYear(), day.getMonth(), day.getDate(), 12);
  const n = Math.round(toJulian(noon) - J2000 - 0.0009 + lon / 360);
  const meanSolarNoon = n + 0.0009 - lon / 360;
  const anomaly = ((357.5291 + 0.98560028 * meanSolarNoon) % 360) * RAD;
  const center = 1.9148 * Math.sin(anomaly) + 0.02 * Math.sin(2 * anomaly) + 0.0003 * Math.sin(3 * anomaly);
  const longitude = ((anomaly / RAD + center + 180 + 102.9372) % 360) * RAD;
  const transit = J2000 + meanSolarNoon + 0.0053 * Math.sin(anomaly) - 0.0069 * Math.sin(2 * longitude);
  const declination = Math.asin(Math.sin(longitude) * Math.sin(23.4397 * RAD));
  // -0.833 degrees: refraction plus the sun's radius, as in published sunrise tables.
  const cosHourAngle = (Math.sin(-0.833 * RAD) - Math.sin(lat * RAD) * Math.sin(declination)) / (Math.cos(lat * RAD) * Math.cos(declination));
  if (Math.abs(cosHourAngle) > 1) return undefined;
  const halfDay = Math.acos(cosHourAngle) / RAD / 360;
  return { sunrise: fromJulian(transit - halfDay), sunset: fromJulian(transit + halfDay) };
}
