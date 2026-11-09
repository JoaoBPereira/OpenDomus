<script setup lang="ts">
  import { computed, inject, onBeforeUnmount, onMounted, ref } from 'vue';

  import { getLatestTagValueFloat } from '../../services/api';
  import { openChartKey } from '../../composables/useOpenChart';
  import { TAG_PC_CPU_LOAD, TAG_PC_CPU_TEMP, TAG_PC_GPU_LOAD, TAG_PC_GPU_TEMP } from '../../constants/tagIds';

  const openChart = inject(openChartKey)!;

  const EMPTY = '\u2014';
  // Readings older than this are treated as the PC being off.
  const MAX_READING_AGE = 5 * 60 * 1000;
  const TEMP_MIN = 20;
  const TEMP_MAX = 100;

  // TEMP ring: 35 °C → 90 °C, 50/50 split
  const TEAL = [70, 210, 160];
  const AMBER = [245, 196, 70];
  const RED = [255, 100, 70];

  // LOAD ring (F · Sky): 0 → 100 %, 55/45 split
  const INDIGO = [70, 60, 190];
  const BLUE = [90, 140, 255];
  const SKY = [170, 225, 255];

  // Colour stops: [teal until, amber at, orange-red from].
  const LOAD_STOPS = [50, 70, 90];
  const TEMP_STOPS = [60, 72, 85];

  const chips = [
    { name: 'CPU', loadTag: TAG_PC_CPU_LOAD, tempTag: TAG_PC_CPU_TEMP },
    { name: 'GPU', loadTag: TAG_PC_GPU_LOAD, tempTag: TAG_PC_GPU_TEMP },
  ];

  const readings = ref<Record<string, number | undefined>>({});

  let intervalId: number | undefined;

  const fetchData = async () => {
    const tags = chips.flatMap((chip) => [chip.loadTag, chip.tempTag]);
    const results = await Promise.allSettled(tags.map((tag) => getLatestTagValueFloat(tag)));

    const next: Record<string, number | undefined> = {};
    results.forEach((result, index) => {
      if (result.status !== 'fulfilled') return;
      const time = Date.parse(result.value.timestamp ?? result.value.time);
      if (Date.now() - time < MAX_READING_AGE) next[tags[index]] = Number(result.value.value);
    });
    readings.value = next;
  };

  // ── Colours ───────────────────────────────────────────────────────────
  function mix(from: number[], to: number[], amount: number) {
    return from.map((channel, index) => Math.round(channel + (to[index] - channel) * amount));
  }

  function rampColor(value: number, [warm, amber, hot]: number[]) {
    let color = TEAL;
    if (value >= hot) color = RED;
    else if (value >= amber) color = mix(AMBER, RED, (value - amber) / (hot - amber));
    else if (value > warm) color = mix(TEAL, AMBER, (value - warm) / (amber - warm));
    return `rgb(${color.join(', ')})`;
  }

  function rampColorLoad(value: number, [warm, amber, hot]: number[]) {
    let color = INDIGO;
    if (value >= hot) color = SKY;
    else if (value >= amber) color = mix(BLUE, SKY, (value - amber) / (hot - amber));
    else if (value > warm) color = mix(INDIGO, BLUE, (value - warm) / (amber - warm));
    return `rgb(${color.join(', ')})`;
  }

  // ── Rows ──────────────────────────────────────────────────────────────
  const clamp = (value: number) => Math.min(Math.max(value, 0), 1);

  // Each ring is a 270° arc (75 of 100 units), open at the bottom.
  function ringStyle(fraction: number, color: string) {
    return { strokeDasharray: `${75 * fraction} 100`, stroke: color, color };
  }

  const rows = computed(() =>
    chips.map((chip) => {
      const load = readings.value[chip.loadTag];
      const temp = readings.value[chip.tempTag];

      return {
        ...chip,
        load,
        temp,
        loadText: load === undefined ? EMPTY : Math.round(load).toString(),
        tempText: temp === undefined ? EMPTY : Math.round(temp).toString(),
        loadColor: load === undefined ? undefined : rampColorLoad(load, LOAD_STOPS),
        tempColor: temp === undefined ? undefined : rampColor(temp, TEMP_STOPS),
        loadRing: load === undefined ? undefined : ringStyle(clamp(load / 100), rampColorLoad(load, LOAD_STOPS)),
        tempRing: temp === undefined ? undefined : ringStyle(clamp((temp - TEMP_MIN) / (TEMP_MAX - TEMP_MIN)), rampColor(temp, TEMP_STOPS)),
      };
    }),
  );

  const offline = computed(() => Object.keys(readings.value).length === 0);

  onMounted(() => {
    fetchData();
    intervalId = window.setInterval(fetchData, 10000);
  });

  onBeforeUnmount(() => {
    if (intervalId) {
      clearInterval(intervalId);
    }
  });
</script>

<template>
  <section class="pcCard" aria-label="PC stats">
    <header class="header">
      <h2>PC</h2>
      <span v-if="offline" class="offline">offline</span>
    </header>

    <div class="display">
      <div class="content">
        <div v-for="row in rows" :key="row.name" class="chipRow">
          <div class="chip">
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <rect x="6" y="6" width="12" height="12" rx="1.5" />
              <rect x="9.5" y="9.5" width="5" height="5" rx="0.5" />
              <path d="M9 2v3M12 2v3M15 2v3M9 19v3M12 19v3M15 19v3M2 9h3M2 12h3M2 15h3M19 9h3M19 12h3M19 15h3" />
            </svg>
            <span class="chipName">{{ row.name }}</span>
          </div>

          <button
            type="button"
            class="gauge"
            :aria-label="`${row.name} temperature ${row.tempText} degrees Celsius. Show chart`"
            @click="openChart(row.tempTag, `${row.name} Temperature`, TEMP_MIN, TEMP_MAX)"
          >
            <svg viewBox="0 0 100 100" aria-hidden="true">
              <circle class="ring track" cx="50" cy="50" r="42" pathLength="100" fill="none" />
              <circle v-if="row.temp !== undefined" class="ring value" cx="50" cy="50" r="42" pathLength="100" fill="none" :style="row.tempRing" />
            </svg>
            <span class="gaugeReading">
              <span class="label">Temp</span>
              <span class="valueLine">
                <span class="value" :style="{ color: row.tempColor }">{{ row.tempText }}</span>
                <span class="unit">&deg;C</span>
              </span>
            </span>
          </button>

          <button type="button" class="gauge" :aria-label="`${row.name} load ${row.loadText}%. Show chart`" @click="openChart(row.loadTag, `${row.name} Load`, 0, 100)">
            <svg viewBox="0 0 100 100" aria-hidden="true">
              <circle class="ring track" cx="50" cy="50" r="42" pathLength="100" fill="none" />
              <circle v-if="row.load !== undefined" class="ring value" cx="50" cy="50" r="42" pathLength="100" fill="none" :style="row.loadRing" />
            </svg>
            <span class="gaugeReading">
              <span class="label">Load</span>
              <span class="valueLine">
                <span class="value" :style="{ color: row.loadColor }">{{ row.loadText }}</span>
                <span class="unit">%</span>
              </span>
            </span>
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<style lang="scss" scoped>
  $card-bg: #101013;
  $track: #1f2027;
  $text: #f4f4f8;
  $muted: #8c93a6;

  // ── Card ──────────────────────────────────────────────────────────────
  .pcCard {
    display: flex;
    flex-direction: column;
    gap: 0.6vh;
    width: 26vw;
    min-height: 0;
    padding: 0.6vh 1.2vw;
    border: 0.08vw solid #25252b;
    border-radius: 0.8vw;
    background: $card-bg;
    color: $text;
    font-family: 'Seven Segment', sans-serif;
    container-type: inline-size;
    box-sizing: border-box;
    -webkit-user-select: none;
    user-select: none;

    @include mobile {
      width: 96vw;
      padding: 0.6vh 3vw;
      border-width: 0.1vh;
      border-radius: 1vh;
    }
  }

  .header {
    display: flex;
    flex-shrink: 0;
    align-items: baseline;
    justify-content: space-between;

    h2 {
      margin: 0;
      font-size: 1.4vh;
      letter-spacing: 0.25em;
      text-transform: uppercase;
    }
  }

  .offline {
    color: $muted;
    font-size: 1.3vh;
  }

  .display {
    flex: 1;
    width: 100%;
    min-height: 0;
    aspect-ratio: 100 / 56;
    container-type: size;
  }

  .content {
    // Scale the readings to fit vertically without narrowing the panels.
    --unit: min(1cqw, calc(100cqh / 56));

    display: flex;
    flex-direction: column;
    gap: var(--unit);
    width: 100%;
    height: 100%;
  }

  // ── Chip row: name | temperature | load ──────────────────────────────
  // Each chip sits on its own slightly lighter panel, with the three items evenly spaced.
  .chipRow {
    display: grid;
    grid-template-columns: auto calc(24.2 * var(--unit)) calc(24.2 * var(--unit));
    align-items: center;
    justify-content: space-evenly;
    flex: 1;
    width: 100%;
    min-height: 0;
    padding: var(--unit) calc(2 * var(--unit));
    border: calc(0.2 * var(--unit)) solid #24252c;
    border-radius: calc(1.5 * var(--unit));
    background: #16171c;
    box-sizing: border-box;
  }

  .chip {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: calc(0.8 * var(--unit));
    color: $muted;

    svg {
      width: calc(8.75 * var(--unit));
      fill: none;
      stroke: currentColor;
      stroke-width: 1.4;
      stroke-linecap: round;
    }
  }

  .chipName {
    color: $text;
    font-size: calc(4 * var(--unit));
    letter-spacing: 0.2em;
  }

  button {
    margin: 0;
    padding: 0;
    border: 0;
    background: none;
    color: inherit;
    font-family: inherit;
    text-align: left;
    cursor: pointer;

    &:focus-visible {
      outline: 0.15vw solid #ffffff;
      outline-offset: 0.3vw;
    }
  }

  // ── Readings ──────────────────────────────────────────────────────────
  .label {
    color: $text;
    font-size: calc(2.4 * var(--unit));
    letter-spacing: 0.15em;
    text-transform: uppercase;
  }

  .valueLine {
    display: flex;
    align-items: baseline;
    gap: calc(0.8 * var(--unit));
    line-height: 1;
  }

  .value {
    color: $text;
    font-size: calc(8 * var(--unit));
    letter-spacing: 0.05em;
    text-shadow: 0 0 var(--unit) currentColor;
    transition: color 0.8s ease;
  }

  .unit {
    color: $muted;
    font-size: calc(3 * var(--unit));
  }

  // ── Gauges ────────────────────────────────────────────────────────────
  // Temperature and load: a 270° ring open at the bottom, with the reading in the middle.
  .gauge {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 0;
    width: 100%;
    aspect-ratio: 1;

    svg {
      position: absolute;
      inset: 0;
      width: 100%;
      height: 100%;
      // Lets the ring glow spread past the drawing area.
      overflow: visible;
    }
  }

  .ring {
    fill: none;
    stroke-width: 7;
    stroke-linecap: round;
    transform: rotate(135deg);
    transform-origin: 50% 50%;

    &.track {
      stroke: $track;
      stroke-dasharray: 75 100;
    }

    // The glow uses the ring's colour, set as its text colour.
    &.value {
      filter: drop-shadow(0 0 var(--unit) currentColor);
      transition:
        stroke-dasharray 0.8s ease,
        stroke 0.8s ease;
    }
  }

  .gaugeReading {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: calc(0.4 * var(--unit));

    .label {
      font-size: calc(2.9 * var(--unit));
    }

    .value {
      font-size: calc(6 * var(--unit));
    }

    .unit {
      align-self: flex-start;
      font-size: calc(2.4 * var(--unit));
    }
  }
</style>
