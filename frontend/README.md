# OpenDomus dashboard

The wall dashboard of OpenDomus: temperatures, power, weather, gate, car and PC,
read from the backend REST API. Built with Vue 3, Vite, TypeScript and ECharts.

## Responsive SCSS

Vue SCSS blocks automatically load [mixins.scss](src/styles/mixins.scss) through
the Vite configuration. Put default styles directly in each class and nest
`@include mobile { ... }` inside it for mobile overrides. The mobile breakpoint
includes widths up to 600px; wider screens use the default styles.

## House temperature floor plans

The temperature section shows both floor plans, the Outside card, the boiler, and
the hot-water tank in a single row on desktop. On mobile the floors stack, then
Outside and hot water share a row, then the boiler gets its own row. Rooms are
clickable and show their temperature and change over the last hour. Data
refreshes every minute.
The hourly baseline is the closest historical reading to one hour ago, within
15 minutes; missing history or a latest reading older than 15 minutes shows a
dash instead of an estimated trend. History request failures are shown separately.

The floor plans have no card or floor label. Room shading uses each room's latest
temperature on a fixed scale: blue at 16° or below, neutral grey at 21°, and
orange at 26° or above. Kitchen shares the Living Room sensor, noted only in its
tooltip; WC has no sensor. Rooms do not show warmest, coolest, or shared-sensor
labels. The Outside card shows the latest outside
temperature and its change over the last hour, shaded on the same colour scale
from 0° (blue) through 20° (grey) to 40° (orange); clicking it opens the outside
chart. The hot-water tank shows
the latest hot-water temperature and its change over the last hour, shaded on
the same colour scale from 20° (blue) through 42.5° (grey) to 65° (orange);
clicking it opens its chart.

The boiler panel shows the water tank (shaded on a 20°–70° scale) with the water
temperature's change over the last hour, its start and
stop limits on a vertical 20°–70° scale, the exhaust temperature, the impulsion
and return pipes, and the impulsion−return difference. The badge shows OFF for
state 0 and ON for any other state. Each reading opens its chart when clicked.

## Power (last hour)

The power card shows the latest total house usage in large digits, the last
hour's minimum, average and peak, how the current reading compares with that
average, and the energy used over the hour in kWh. Below are 60 bars, one per
minute, with a dashed line at the hour's average; the rightmost bar is the
latest reading. Colours follow the load: teal up to 600 W, blending to amber at
1.2 kW and orange-red at 2 kW and above. Data refreshes every 30 seconds, and
clicking a bar opens the per-device breakdown for that minute.

The Devices panel lists each device drawing more than 5 W (readings from the
last 5 minutes) plus "Unknown" (total minus known devices, drawn hatched),
sorted by power. Rows show only the device icon (the name appears on hover).
Each bar shows its watts and its share of the total; the bar
colour runs green to yellow to orange along the track, so bigger loads reach
warmer colours. Bars grow and shrink smoothly when values change, rows slide
when the order changes, and devices fade in or out as they turn on or off.
Clicking a device opens its chart. The same panel is used for the per-minute
breakdown, without the total row.

## Energy per device

The row of device energy cards (tags marked `PowerConsumptionStats`) shows,
for each device, today's kWh in large digits with today's cost underneath, then
a small table with kWh and € columns for the week and month so the numbers line
up. At the bottom, a thin bar shows the device's share of the whole home's
consumption this month (energy tag 11), going teal → amber → orange-red as the
share grows (amber around 25%, orange-red from 40%). Costs use `priceKwh` from
the config. Clicking a card opens the full table, including yesterday, last week
and last month. The cards are hidden on mobile.

## Weather

The weather card (hidden on mobile) shows, in the header, the current
temperature and the next 24 hours' low and high. Below it, 24 hourly columns
start with a highlighted NOW column; each has the hour, an icon and the
temperature, joined by a curve whose dots, line and soft glow follow the
dashboard's blue → grey → orange scale (blue at 8° or below, orange from 24°).
Night hours (21:00–06:00) have a faint blue background, and clear or partly
cloudy nights show a moon. The bottom row has ten daily tiles with the high on
top, the low below and a vertical range bar on a scale shared across all ten
days; today's tile is outlined. Hovering an hour or day shows rain chance and
wind. The forecast (IPMA) refreshes hourly. The card fills the height of
`.weatherContainer` in `MainView.vue` (20vh by default) and everything inside
scales with it, so changing that one height resizes the whole card.

## Gate

The gate card draws the gate between two lit pillars. Closed: blue lights, both
leaves shut with a lock badge, and a CLOSED pill. Open: amber lights, leaves
swung open, and an OPEN pill. Changes animate over about 1.6 s: the leaves swing
in 3D while colours and the floor glow fade across, and when closing the lock
reappears only once the leaves meet. The first reading after page load is shown
without animation. The sensor is checked every 10 seconds; clicking the card
opens the gate history.

## PC

The PC card, placed after the gate, has one panel per chip: CPU (tags 40 load,
41 temperature) and GPU (tags 42 load, 43 temperature). Each panel shows a chip
icon and name, then a circular temperature gauge (a 270° ring covering
20–100 °C, with the °C reading in the middle), then, past a divider, the load as
a large % above a 20-segment bar that lights left to right (teal → amber →
orange-red, with the last lit segment glowing). Load turns amber around 70% and
orange-red from 90%; temperature turns warm from about 60° and hot from 85°.
Clicking the load or temperature opens its chart. Readings refresh every 10
seconds; readings older than 5 minutes are treated as the PC being off, showing
empty gauges and "offline". On mobile the gate and car share a row and the PC
card gets its own full-width row below.

## Setup

The backend URL comes from `BACKEND_SERVER_URL` in the repo-root `.env` at build time
(or from the environment, e.g. in CI). Without it the app uses port 4000 of the host it
is served from. An optional `public/config.json` with `serverURL` and `priceKwh` (the
electricity price used for the energy costs) overrides it at runtime, so a built app can
be pointed at another server without rebuilding.

```bash
npm install
npm run dev      # dev server on http://localhost:3007, reachable from the LAN
npm start        # dev server on http://localhost:3000
npm run build    # production build in dist/ (bundle report in dist/stats.html)
npm run preview  # serve the production build locally
```

The build uses relative paths, so `dist/` can be served from any folder (the
Docker image serves it at `/frontend/`).
