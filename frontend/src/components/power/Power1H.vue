<script setup lang="ts">
  import { computed, onBeforeUnmount, onMounted, ref } from 'vue';

  import { getLatestTagValueFloat, getPowerConsumptionNearTimestamp, getTagValueFloatTimeSeries, type TagValueTimeSeries } from '../../services/api';
  import { TAG_POWER_TOTAL, TAG_POWER_ID_NAMES } from '../../constants/tagIds';

  import Modal from '../ui/Modal.vue';
  import { PowerData } from '../../models';

  import PowerBarChartKHH from './PowerBarChartKHH.vue';

  const EMPTY = '\u2014';
  const MINUTE = 60 * 1000;
  const BAR_COUNT = 60;
  const MIN_SCALE_W = 1000;

  const TEAL = [79, 195, 161];
  const AMBER = [224, 163, 58];
  const RED = [240, 100, 60];

  interface MinuteBar {
    time: number;
    power?: number;
  }

  const series = ref<TagValueTimeSeries[]>([]);
  const current = ref<{ power: number; time: number }>();
  const windowEnd = ref(Date.now());

  const showDialog = ref(false);
  const dialogTitle = ref<string | undefined>();
  const dataPower = ref<PowerData[]>([]);

  let intervalId: number | undefined;

  // ── Load colours ──────────────────────────────────────────────────────
  // Teal up to 600 W, blending to amber at 1.2 kW, then to orange-red at 2 kW and above.
  function loadColor(power: number) {
    if (power <= 600) return rgb(TEAL);
    if (power <= 1200) return rgb(mix(TEAL, AMBER, (power - 600) / 600));
    return rgb(mix(AMBER, RED, Math.min((power - 1200) / 800, 1)));
  }

  function mix(from: number[], to: number[], amount: number) {
    return from.map((channel, index) => Math.round(channel + (to[index] - channel) * amount));
  }

  function rgb([red, green, blue]: number[]) {
    return `rgb(${red}, ${green}, ${blue})`;
  }

  // ── Last-hour data ────────────────────────────────────────────────────
  const minuteStart = (time: number) => Math.floor(time / MINUTE) * MINUTE;

  // One bar per minute; the rightmost "now" bar shows the latest reading.
  const bars = computed<MinuteBar[]>(() => {
    const byMinute = new Map(series.value.map((row) => [minuteStart(Date.parse(row.bucket)), Number(row.avg_value)]));
    const lastMinute = minuteStart(windowEnd.value);

    return Array.from({ length: BAR_COUNT }, (_, index) => {
      const time = lastMinute - (BAR_COUNT - 1 - index) * MINUTE;
      const isNow = index === BAR_COUNT - 1;
      return { time, power: isNow && current.value ? current.value.power : byMinute.get(time) };
    });
  });

  const stats = computed(() => {
    const rows = series.value;
    if (!rows.length) return undefined;

    const averages = rows.map((row) => Number(row.avg_value));
    const average = averages.reduce((sum, value) => sum + value, 0) / averages.length;
    return {
      min: Math.min(...rows.map((row) => Number(row.min_value))),
      average,
      peak: Math.max(...rows.map((row) => Number(row.max_value))),
      // Each bucket is one minute, so its average watts × 1/60 h is that minute's energy.
      energyKwh: averages.reduce((sum, value) => sum + value, 0) / 60 / 1000,
    };
  });

  const scaleMax = computed(() => Math.max(stats.value?.peak ?? 0, current.value?.power ?? 0, MIN_SCALE_W));

  const currentText = computed(() => (current.value ? current.value.power.toFixed(2) : EMPTY));
  const currentStyle = computed(() => (current.value ? { color: loadColor(current.value.power) } : {}));

  const nowVsAverage = computed(() => {
    if (!current.value || !stats.value || stats.value.average === 0) return EMPTY;
    const percent = Math.round(((current.value.power - stats.value.average) / stats.value.average) * 100);
    return `${percent > 0 ? '+' : ''}${percent}%`;
  });

  const averageLineStyle = computed(() => (stats.value ? { bottom: `${(stats.value.average / scaleMax.value) * 100}%` } : undefined));

  function barStyle(bar: MinuteBar) {
    if (bar.power === undefined) return { height: '0' };
    const color = loadColor(bar.power);
    return { height: `${Math.max((bar.power / scaleMax.value) * 100, 1)}%`, background: color, color };
  }

  function formatWatts(value?: number) {
    return value === undefined ? EMPTY : `${Math.round(value)}`;
  }

  const fetchData = async () => {
    const now = Date.now();
    const start = minuteStart(now) - (BAR_COUNT - 1) * MINUTE;

    const [latest, rows] = await Promise.allSettled([
      getLatestTagValueFloat(TAG_POWER_TOTAL),
      getTagValueFloatTimeSeries(TAG_POWER_TOTAL, new Date(start).toISOString(), new Date(now).toISOString(), 1),
    ]);

    windowEnd.value = now;
    if (latest.status === 'fulfilled') {
      current.value = { power: Number(latest.value.value), time: Date.parse(latest.value.timestamp ?? latest.value.time) };
    }
    if (rows.status === 'fulfilled') {
      // The API returns 0 instead of an empty list when there is no data.
      series.value = Array.isArray(rows.value) ? rows.value : [];
    }
  };

  // ── Per-device breakdown for a clicked minute ─────────────────────────
  const getGradient = (percent: number): string => {
    const g = percent < 50 ? 255 : Math.floor(255 - ((percent * 2 - 100) * 255) / 100);
    const r = percent > 50 ? 255 : Math.floor((percent * 2 * 255) / 100);
    return `rgb(${r}, ${g}, 0)`;
  };

  const clickOnBar = (bar: MinuteBar) => {
    if (bar.power === undefined) return;
    getPowerConsumptionNearTimestamp(new Date(bar.time).toISOString())
      .then((json) => {
        const powerData: PowerData[] = [];

        const houseCurrentWatts = json.find((element) => element.tag.id === TAG_POWER_TOTAL);
        if (!houseCurrentWatts || houseCurrentWatts.value === null) return;
        const totalPower = houseCurrentWatts.value;
        dialogTitle.value = `Power Consumption - ${new Date(bar.time).toLocaleTimeString([], {
          hour: '2-digit',
          minute: '2-digit',
          hour12: false,
        })}`;

        json.forEach((element) => {
          if (element.value !== null && element.value > 0) {
            powerData.push({
              time: new Date(element.time).getTime(),
              power: Math.round(element.value),
              name: TAG_POWER_ID_NAMES[element.tag.id] ?? element.tag.name,
              percentage: (element.value / totalPower) * 100,
              color: getGradient((element.value / totalPower) * 100),
              tagId: element.tag.id,
            });
          }
        });

        // Unknown is the difference between the total and the sum of known devices.
        const sumOfKnown = powerData.filter((item) => item.tagId !== TAG_POWER_TOTAL).reduce((sum, item) => sum + item.power, 0);
        const unknownPower = totalPower - sumOfKnown;

        if (unknownPower > 0) {
          powerData.push({
            time: new Date(houseCurrentWatts.time).getTime(),
            power: Math.round(unknownPower),
            name: 'unknown',
            percentage: (unknownPower / totalPower) * 100,
            color: getGradient((unknownPower / totalPower) * 100),
            tagId: 0,
          });
        }

        dataPower.value = powerData;
        showDialog.value = true;
      })
      .catch((err) => {
        console.error('Error fetching power consumption data:', err);
      });
  };

  const closeDialog = () => {
    showDialog.value = false;
  };

  onMounted(() => {
    fetchData();
    intervalId = window.setInterval(fetchData, 30000);
  });

  onBeforeUnmount(() => {
    if (intervalId) {
      clearInterval(intervalId);
    }
  });
</script>

<template>
  <div class="Power1HMain">
    <header class="header">
      <h2>Power</h2>
      <span class="energy">
        last hour &middot; <strong>{{ stats ? stats.energyKwh.toFixed(2) : EMPTY }}</strong> kWh
      </span>
    </header>

    <div class="current" :style="currentStyle">
      <span class="currentValue">{{ currentText }}</span>
      <span class="unit">W</span>
    </div>

    <div class="stats">
      <span class="stat">
        <span class="statLabel">Min</span>
        <span :style="{ color: stats ? loadColor(stats.min) : undefined }">{{ formatWatts(stats?.min) }} W</span>
      </span>
      <span class="stat">
        <span class="statLabel">Avg</span>
        <span :style="{ color: stats ? loadColor(stats.average) : undefined }">{{ formatWatts(stats?.average) }} W</span>
      </span>
      <span class="stat">
        <span class="statLabel">Peak</span>
        <span :style="{ color: stats ? loadColor(stats.peak) : undefined }">{{ formatWatts(stats?.peak) }} W</span>
      </span>
      <span class="stat">
        <span class="statLabel">Now vs avg</span>
        <span class="nowVsAvg">{{ nowVsAverage }}</span>
      </span>
    </div>

    <div class="chart">
      <span v-if="averageLineStyle" class="averageLine" :style="averageLineStyle" />
      <button
        v-for="(bar, index) in bars"
        :key="bar.time"
        type="button"
        class="bar"
        :class="{ now: index === BAR_COUNT - 1 }"
        :style="barStyle(bar)"
        :aria-label="`${new Date(bar.time).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}: ${formatWatts(bar.power)} W`"
        @click="clickOnBar(bar)"
      />
    </div>

    <div class="timeLabels">
      <span>-60 min</span>
      <span>-45</span>
      <span>-30</span>
      <span>-15</span>
      <span class="nowLabel">now</span>
    </div>

    <Modal :open="showDialog" :title="dialogTitle" @close="closeDialog">
      <PowerBarChartKHH :data="dataPower" />
    </Modal>
  </div>
</template>

<style lang="scss" scoped>
  $card-bg: #101013;
  $text: #f4f4f8;
  $muted: #8c93a6;
  $unit: #b9b4e6;

  // ── Card ──────────────────────────────────────────────────────────────
  .Power1HMain {
    display: flex;
    flex-direction: column;
    gap: 0.6vh;
    height: 100%;
    margin: 0 0.5vw;
    padding: 1.2vh 1.2vw 0.8vh;
    border: 0.08vw solid #25252b;
    border-radius: 0.8vw;
    background: $card-bg;
    color: $text;
    font-family: 'Seven Segment', sans-serif;
    box-sizing: border-box;
    -webkit-user-select: none;
    user-select: none;

    @include mobile {
      margin: 0 2vw;
      padding: 1vh 3vw 0.6vh;
      border-width: 0.1vh;
      border-radius: 1vh;
    }
  }

  // ── Header ────────────────────────────────────────────────────────────
  .header {
    display: flex;
    align-items: baseline;
    justify-content: space-between;

    h2 {
      margin: 0;
      font-size: 1.4vh;
      letter-spacing: 0.25em;
      text-transform: uppercase;
    }
  }

  .energy {
    color: $muted;
    font-size: 1.3vh;

    strong {
      color: $text;
      font-size: 1.6vh;
      font-weight: normal;
    }
  }

  // ── Current usage ─────────────────────────────────────────────────────
  // The glow uses the load colour set on this element.
  .current {
    display: flex;
    align-items: flex-end;
    justify-content: center;
    gap: 0.4vw;
    margin-top: -1vh;
    color: $text;
    line-height: 1;

    @include mobile {
      margin-top: 0;
    }
  }

  .currentValue {
    font-size: 5.5vh;
    letter-spacing: 0.06em;
    text-shadow: 0 0 1.5vh currentColor;
  }

  .unit {
    color: $unit;
    font-size: 2vh;
  }

  // ── Stats ─────────────────────────────────────────────────────────────
  .stats {
    display: flex;
    justify-content: center;
    gap: 1.6vw;
    font-size: 1.3vh;
    white-space: nowrap;

    @include mobile {
      gap: 3vw;
    }
  }

  .statLabel {
    margin-right: 0.4vw;
    color: $muted;
    font-size: 0.85em;
    letter-spacing: 0.12em;
    text-transform: uppercase;

    @include mobile {
      margin-right: 1vw;
    }
  }

  .nowVsAvg {
    color: $text;
  }

  // ── Chart ─────────────────────────────────────────────────────────────
  .chart {
    position: relative;
    display: flex;
    flex: 1;
    align-items: flex-end;
    gap: 0.15vw;
    min-height: 0;

    @include mobile {
      gap: 0.4vw;
    }
  }

  .averageLine {
    position: absolute;
    left: 0;
    right: 0;
    border-top: 0.1vh dashed rgba(200, 200, 210, 0.35);
    pointer-events: none;
  }

  .bar {
    flex: 1;
    min-width: 0;
    margin: 0;
    padding: 0;
    border: 0;
    border-radius: 0.2vw 0.2vw 0 0;
    opacity: 0.8;
    cursor: pointer;

    &:hover {
      opacity: 1;
    }

    &:focus-visible {
      outline: 0.15vw solid #ffffff;
      outline-offset: 0.1vw;
    }

    // The bar's text colour is its load colour, used for the glow.
    &.now {
      opacity: 1;
      box-shadow: 0 0 1vh currentColor;
      filter: brightness(1.2);
    }
  }

  .timeLabels {
    display: flex;
    justify-content: space-between;
    color: $muted;
    font-size: 1.1vh;
  }

  .nowLabel {
    color: $text;
  }
</style>
