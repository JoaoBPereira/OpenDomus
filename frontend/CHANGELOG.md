# Changelog

All notable changes to the frontend dashboard are listed here.

## Unreleased

### Added

- **PC card** (`components/pc/PcStats.vue`), placed after the gate. One panel per chip, CPU and GPU:
  - Circular temperature gauge (20–100 °C) with the reading in the middle.
  - Load as a large % above a 20-segment bar that lights left to right (teal → amber → orange-red); the last lit segment glows.
  - Colours: load turns amber around 70% and orange-red from 90%; temperature warms from 60° and is hot from 85°.
  - Clicking the temperature or load opens its chart. Readings refresh every 10 seconds; readings older than 5 minutes show the card as "offline".
  - New tags in `constants/tagIds.ts`: CPU load 40, CPU temperature 41, GPU load 42, GPU temperature 43.

### Changed

- **Power (last hour)** (`components/power/Power1H.vue`), redesigned:
  - Current usage in large digits, coloured by load with a soft glow.
  - Last hour's minimum, average and peak, "now vs average", and energy used in kWh.
  - 60 per-minute bars coloured by load (teal → amber at 1.2 kW → orange-red at 2 kW), a dashed average line, a highlighted "now" bar and time labels.
  - Bars now show per-minute averages instead of the last 119 raw readings. Clicking a bar still opens that minute's per-device breakdown.
- **Devices** (`components/power/PowerBarChartKHH.vue`), redesigned:
  - One row per device: icon, a green → yellow → orange bar with the watts at its end, and the share of the total.
  - "Unknown" (total minus known devices) is drawn hatched.
  - Rows are sorted largest first. Bars grow and shrink smoothly, rows slide when the order changes, and devices fade in and out.
  - The device name was removed from rows; it's still in the tooltip.
  - Replaces the ECharts bar chart.
- **Gate** (`components/gate/Gate.vue`), redesigned:
  - The gate is drawn in code instead of using the open/closed PNG pictures.
  - Closed: blue lamps, shut leaves and a lock. Open: amber lamps and swung-open leaves.
  - Changes animate over about 1.6 s with a 3D swing; the first reading after page load isn't animated.
  - The title sits in the top-left corner like the other cards, and the card is wider (26% of the screen width, 46% on phones).
- **Weather** (`components/weather/Weather.vue`), redesigned:
  - Header with the current temperature and the next 24 hours' low and high.
  - 24 hourly columns starting with a highlighted NOW column, joined by a temperature curve coloured blue → grey → orange with a soft glow.
  - Night hours (21:00–06:00) have a faint blue background, and clear nights show a moon.
  - Ten daily tiles with high, low and a vertical range bar on a scale shared across all days.
  - New thin-line icons (`WeatherIcon.vue`) in colour: amber sun, pale-blue moon, grey clouds, blue rain. Replaces `WeatherItem.vue`.
  - Wind and rain chance moved to hover tooltips.
  - Sized by height: the card fills `.weatherContainer` (20vh) and its contents scale with it.
- **Dashboard layout** (`views/MainView.vue`): on phones, the gate and car share a row and the PC card gets its own full-width row below.
- **Energy per device** (`components/power/PowerConsumptionDaily.vue`), redesigned:
  - Today's kWh in large digits with today's cost underneath.
  - A small table with kWh and € columns for the week and month, with the numbers lined up.
  - A thin bar showing the device's share of the home's consumption this month (energy tag 11), teal → amber → orange-red as the share grows.
  - Clicking a card still opens the full table. Unit conversion moved to a shared helper (`components/power/powerEnergy.ts`).
- **Temperature section** tweaks:
  - First-floor hallway reduced to 25% of the floor height, with its temperature and change side by side.
  - Floor plans are shorter on phones.
  - Room temperatures use a single text size, and the warm (orange) end of the colour scale is brighter.
  - Outside card narrowed slightly, with tighter spacing and margins in the temperature row.
- **Boiler** layout tweaks: column widths and the position of the flow−return difference.
- **Power section** spacing: the gap above the Devices panel now applies on phones only.

### Fixed

- The Devices "Unknown" share showed the known devices' share instead of its own.
- The per-minute breakdown pop-up no longer counts the house total as one of the devices, so its percentages add up to 100%.

### Notes

- `images/closed_gate.png` and `images/open_gate.png` are no longer used and can be deleted.
- The frontend `README.md` describes the new Power, Devices, Gate and PC cards.
