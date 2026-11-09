<script setup lang="ts">
  import { computed, onBeforeUnmount, onMounted, ref } from 'vue';

  import WeatherIcon from './WeatherIcon.vue';
  import { isNight, weatherColor } from './weatherData';
  import { getWeather, type WeatherResponse } from '../../services/api';

  const HOURS = 24;
  const DAYS = 10;

  const weatherData = ref<WeatherResponse | null>(null);
  const isLoading = ref(true);
  const errorMessage = ref('');

  let intervalId: number | undefined;

  const fetchWeather = async () => {
    try {
      weatherData.value = await getWeather();
      errorMessage.value = '';
    } catch {
      errorMessage.value = 'Failed to load weather forecast.';
    } finally {
      isLoading.value = false;
    }
  };

  // ── Hourly ────────────────────────────────────────────────────────────
  // Starts at the latest forecast hour that isn't in the future, shown as "NOW".
  const hours = computed(() => {
    const entries = [...(weatherData.value?.hourly24?.entries ?? [])].sort((a, b) => Date.parse(a.time) - Date.parse(b.time));
    if (!entries.length) return [];

    const firstFuture = entries.findIndex((entry) => Date.parse(entry.time) > Date.now());
    const start = firstFuture === -1 ? Math.max(0, entries.length - HOURS) : Math.max(0, firstFuture - 1);
    const shown = entries.slice(start, start + HOURS);

    const temps = shown.map((entry) => entry.temperature);
    const low = Math.min(...temps);
    const span = Math.max(Math.max(...temps) - low, 4);

    return shown.map((entry, index) => {
      const time = new Date(entry.time);
      const temp = Math.round(entry.temperature);
      const label = index === 0 ? 'Now' : time.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' });
      return {
        key: entry.time,
        label,
        temp,
        typeId: entry.weatherTypeId,
        night: isNight(time),
        color: weatherColor(entry.temperature),
        // Column centre across, and height in the chart (leaving room above for the label).
        x: ((index + 0.5) / shown.length) * 100,
        y: 8 + ((entry.temperature - low) / span) * 52,
        description: `${label} · ${temp}° · rain ${entry.precipitationProbability ?? '--'}% · wind ${Math.round(entry.windSpeed)} km/h`,
      };
    });
  });

  const columnWidth = computed(() => `${100 / Math.max(hours.value.length, 1)}%`);

  // Consecutive night hours become one shaded band.
  const nightBands = computed(() => {
    const bands: { left: string; width: string }[] = [];
    const count = hours.value.length;
    let runStart = -1;
    hours.value.forEach((hour, index) => {
      if (hour.night && runStart === -1) runStart = index;
      const runEnds = runStart !== -1 && (!hour.night || index === count - 1);
      if (!runEnds) return;
      const end = hour.night ? index + 1 : index;
      bands.push({ left: `${(runStart / count) * 100}%`, width: `${((end - runStart) / count) * 100}%` });
      runStart = -1;
    });
    return bands;
  });

  const linePoints = computed(() => hours.value.map((hour) => `${hour.x},${100 - hour.y}`).join(' '));
  const areaPoints = computed(() => {
    const list = hours.value;
    if (!list.length) return '';
    return `${linePoints.value} ${list[list.length - 1].x},100 ${list[0].x},100`;
  });

  const summary = computed(() => {
    if (!hours.value.length) return undefined;
    const temps = hours.value.map((hour) => hour.temp);
    const low = Math.min(...temps);
    const high = Math.max(...temps);
    return {
      now: hours.value[0].temp,
      low,
      high,
      nowColor: hours.value[0].color,
      lowColor: weatherColor(low),
      highColor: weatherColor(high),
    };
  });

  // ── Daily ─────────────────────────────────────────────────────────────
  // Range bars share one scale across all days, so warmer days sit higher.
  const days = computed(() => {
    const list = (weatherData.value?.daily10 ?? []).slice(0, DAYS);
    if (!list.length) return [];

    const low = Math.min(...list.map((day) => day.temperatureMin));
    const span = Math.max(Math.max(...list.map((day) => day.temperatureMax)) - low, 1);
    const today = new Date().toLocaleDateString('en-CA');

    return list.map((day) => {
      const dateKey = day.date.slice(0, 10);
      const minColor = weatherColor(day.temperatureMin);
      const maxColor = weatherColor(day.temperatureMax);
      return {
        key: dateKey,
        name: new Date(`${dateKey}T12:00:00`).toLocaleDateString('en-US', { weekday: 'short' }),
        isToday: dateKey === today,
        typeId: day.weatherTypeId,
        min: Math.round(day.temperatureMin),
        max: Math.round(day.temperatureMax),
        minColor,
        maxColor,
        rangeStyle: {
          bottom: `${((day.temperatureMin - low) / span) * 100}%`,
          height: `${Math.max(((day.temperatureMax - day.temperatureMin) / span) * 100, 4)}%`,
          background: `linear-gradient(to top, ${minColor}, ${maxColor})`,
        },
        description: `${dateKey}: ${Math.round(day.temperatureMin)}° to ${Math.round(day.temperatureMax)}°, rain ${day.precipitationProbability}%, wind ${day.windDirection}`,
      };
    });
  });

  onMounted(() => {
    fetchWeather();
    intervalId = window.setInterval(fetchWeather, 60 * 60 * 1000);
  });

  onBeforeUnmount(() => {
    if (intervalId) {
      clearInterval(intervalId);
    }
  });
</script>

<template>
  <section class="weatherCard" aria-label="Weather">
    <div class="content">
      <header class="header">
        <h2>Weather</h2>
        <span v-if="summary" class="summary">
          now <strong :style="{ color: summary.nowColor }">{{ summary.now }}&deg;</strong> &middot; next 24h <strong :style="{ color: summary.lowColor }">{{ summary.low }}&deg;</strong> /
          <strong :style="{ color: summary.highColor }">{{ summary.high }}&deg;</strong>
        </span>
      </header>

      <div v-if="isLoading" class="state">Loading weather...</div>
      <div v-else-if="errorMessage" class="state error">{{ errorMessage }}</div>

      <template v-else>
        <div class="hourly">
          <span v-for="(band, index) in nightBands" :key="index" class="nightBand" :style="band" />
          <span class="nowBand" :style="{ width: columnWidth }" />

          <div class="hourColumns">
            <div v-for="hour in hours" :key="hour.key" class="hour" :title="hour.description">
              <span class="hourTime" :class="{ now: hour.label === 'Now' }">{{ hour.label }}</span>
              <WeatherIcon class="hourIcon" :type-id="hour.typeId" :night="hour.night" />
            </div>
          </div>

          <div class="chart">
            <svg viewBox="0 0 100 100" preserveAspectRatio="none" aria-hidden="true">
              <defs>
                <linearGradient id="weatherTempLine" gradientUnits="userSpaceOnUse" x1="0" y1="0" x2="100" y2="0">
                  <stop v-for="hour in hours" :key="hour.key" :offset="hour.x / 100" :stop-color="hour.color" />
                </linearGradient>
              </defs>
              <polygon class="area" :points="areaPoints" />
              <polyline class="line" :points="linePoints" />
            </svg>

            <span v-for="hour in hours" :key="hour.key" class="point" :style="{ left: `${hour.x}%`, bottom: `${hour.y}%`, color: hour.color }">
              <span class="pointLabel">{{ hour.temp }}&deg;</span>
            </span>
          </div>
        </div>

        <div class="days">
          <div v-for="day in days" :key="day.key" class="day" :class="{ today: day.isToday }" :title="day.description">
            <div class="dayHeader">
              <span class="dayName">{{ day.name }}</span>
              <WeatherIcon class="dayIcon" :type-id="day.typeId" />
            </div>
            <div class="dayTemps">
              <span class="dayTemp" :style="{ color: day.maxColor }">{{ day.max }}&deg;</span>
              <span class="range">
                <span class="rangeFill" :style="day.rangeStyle" />
              </span>
              <span class="dayTemp" :style="{ color: day.minColor }">{{ day.min }}&deg;</span>
            </div>
          </div>
        </div>
      </template>
    </div>
  </section>
</template>

<style lang="scss" scoped>
  $card-bg: #101013;
  $panel-bg: #16171c;
  $track: #1f2027;
  $text: #f4f4f8;
  $muted: #8c93a6;

  // ── Card ──────────────────────────────────────────────────────────────
  // Fills its container's height. Everything inside is sized in cqh (a share of
  // the card's height), so the whole card scales when the container changes.
  .weatherCard {
    height: 100%;
    margin: 0 0.5vw;
    border: 0.08vw solid #25252b;
    border-radius: 0.8vw;
    background: $card-bg;
    color: $text;
    font-family: 'Seven Segment', sans-serif;
    container-type: size;
    box-sizing: border-box;
    -webkit-user-select: none;
    user-select: none;
  }

  .content {
    display: flex;
    flex-direction: column;
    gap: 1.8cqh;
    height: 100%;
    padding: 3cqh 1.2vw;
    box-sizing: border-box;
  }

  .header {
    display: flex;
    align-items: baseline;
    justify-content: space-between;

    h2 {
      margin: 0;
      font-size: 4cqh;
      letter-spacing: 0.25em;
      text-transform: uppercase;
    }
  }

  .summary {
    color: $muted;
    font-size: 4cqh;

    strong {
      font-size: 5.5cqh;
      font-weight: normal;
    }
  }

  .state {
    color: $muted;
    font-size: 5cqh;
  }

  .error {
    color: #fda4af;
  }

  // ── Hourly strip ──────────────────────────────────────────────────────
  .hourly {
    position: relative;
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
  }

  .nightBand,
  .nowBand {
    position: absolute;
    top: 0;
    bottom: 0;
  }

  .nightBand {
    background: rgba(110, 120, 200, 0.08);
  }

  .nowBand {
    left: 0;
    border-radius: 0.5vw;
    background: rgba(255, 255, 255, 0.07);
  }

  .hourColumns {
    position: relative;
    display: flex;
  }

  .hour {
    display: flex;
    flex: 1;
    flex-direction: column;
    align-items: center;
    gap: 1cqh;
    min-width: 0;
    padding-top: 1.5cqh;
  }

  .hourTime {
    color: $muted;
    font-size: 5cqh;
    white-space: nowrap;

    &.now {
      color: $text;
      letter-spacing: 0.1em;
      text-transform: uppercase;
    }
  }

  .hourIcon {
    width: 8cqh;
    height: 8cqh;
  }

  // ── Temperature line ──────────────────────────────────────────────────
  // Takes the rest of the strip's height.
  .chart {
    position: relative;
    flex: 1;
    min-height: 0;

    svg {
      position: absolute;
      inset: 0;
      width: 100%;
      height: 100%;
    }
  }

  .line {
    fill: none;
    stroke: url(#weatherTempLine);
    stroke-width: 2;
    stroke-linejoin: round;
    vector-effect: non-scaling-stroke;
  }

  // The soft glow underneath the line, in the same colours.
  .area {
    fill: url(#weatherTempLine);
    opacity: 0.16;
  }

  // Dot and label take the hour's temperature colour.
  .point {
    position: absolute;
    width: 2cqh;
    height: 2cqh;
    border-radius: 50%;
    background: currentColor;
    transform: translate(-50%, 50%);
  }

  .pointLabel {
    position: absolute;
    bottom: calc(100% + 0.1cqh);
    left: 50%;
    font-size: 6cqh;
    white-space: nowrap;
    transform: translateX(-50%);
  }

  // ── Daily cards ───────────────────────────────────────────────────────
  // The daily tiles get more height than the hourly strip, leaving room for the range bars.
  .days {
    display: flex;
    flex: 1.5;
    gap: 0.6vw;
    min-height: 0;
  }

  .day {
    display: flex;
    flex: 1;
    flex-direction: row;
    align-items: center;
    gap: 1cqh;
    min-width: 0;
    padding: 1.8cqh 0.4vw;
    border: 0.08vw solid #24252c;
    border-radius: 0.6vw;
    background: $panel-bg;
    box-sizing: border-box;

    &.today {
      border-color: #5a5d6a;
    }
  }

  .dayHeader {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 3cqh;
  }

  .dayTemps {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1cqh;
    height: 100%;
  }

  .dayName {
    font-size: 7cqh;
    letter-spacing: 0.2em;
    text-transform: uppercase;
  }

  .dayIcon {
    width: 12cqh;
    height: 12cqh;
  }

  .dayTemp {
    font-size: 7cqh;
  }

  // Takes the rest of the tile's height.
  .range {
    position: relative;
    flex: 1;
    width: 1vw;
    min-height: 0;
    border-radius: 0.25vw;
    background: $track;
  }

  .rangeFill {
    position: absolute;
    left: 0;
    right: 0;
    border-radius: inherit;
  }
</style>
