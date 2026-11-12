<script setup lang="ts">
  import { computed } from 'vue';
  import { sunTimes } from './sunTimes';

  const props = defineProps<{ now: Date }>();

  // Lisbon unless config.json sets sunLat / sunLon.
  const lat = Number.isFinite(Config.sunLat) ? Config.sunLat! : 38.72;
  const lon = Number.isFinite(Config.sunLon) ? Config.sunLon! : -9.14;

  // Arc geometry in SVG units: a half circle centred on the horizon line.
  const CX = 50;
  const CY = 46;
  const R = 38;
  const EMPTY = '--:--';

  const times = computed(() => sunTimes(props.now, lat, lon));
  const format = (date?: Date) => date?.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' }) ?? EMPTY;

  // The next two sun events, soonest first: sunset then tomorrow's sunrise at 13:00, tomorrow's sunrise then sunset at 23:00.
  const upcoming = computed<{ kind: 'rise' | 'set'; at?: Date }[]>(() => {
    const tomorrow = new Date(props.now);
    tomorrow.setDate(tomorrow.getDate() + 1);
    const events = [times.value, sunTimes(tomorrow, lat, lon)]
      .flatMap((day) => (day ? [{ kind: 'rise' as const, at: day.sunrise }, { kind: 'set' as const, at: day.sunset }] : []))
      .filter((event) => event.at > props.now)
      .slice(0, 2);
    // Polar day or night: no events, show both rows empty.
    return events.length === 2 ? events : [{ kind: 'rise' }, { kind: 'set' }];
  });
  const label = computed(() => upcoming.value.map((event) => `${event.kind === 'rise' ? 'Sunrise' : 'Sunset'} ${format(event.at)}`).join(', '));

  // 0 at sunrise, 1 at sunset; outside that range it is night.
  const progress = computed(() => {
    if (!times.value) return undefined;
    const { sunrise, sunset } = times.value;
    return (props.now.getTime() - sunrise.getTime()) / (sunset.getTime() - sunrise.getTime());
  });
  const isDay = computed(() => progress.value !== undefined && progress.value >= 0 && progress.value <= 1);

  const sun = computed(() => {
    // At night the sun waits on the horizon: left before sunrise, right after sunset.
    const t = isDay.value ? progress.value! : (progress.value ?? 0) > 1 ? 1 : 0;
    const angle = Math.PI * (1 - t);
    return { x: CX + R * Math.cos(angle), y: CY - R * Math.sin(angle) };
  });
  const travelled = computed(() => `M ${CX - R} ${CY} A ${R} ${R} 0 0 1 ${sun.value.x} ${sun.value.y}`);

  // Daylight left today: all of it before sunrise, counting down during the day, 0 after sunset.
  const daylight = computed(() => {
    if (!times.value) return EMPTY;
    const { sunrise, sunset } = times.value;
    const from = Math.max(props.now.getTime(), sunrise.getTime());
    const minutes = Math.max(0, Math.ceil((sunset.getTime() - from) / 60000));
    return `${Math.floor(minutes / 60)}h ${String(minutes % 60).padStart(2, '0')}m`;
  });
</script>

<template>
  <div class="sun" :aria-label="`${label}, daylight left ${daylight}`">
    <svg class="arc" viewBox="0 0 100 52" aria-hidden="true">
      <path class="track" :d="`M ${CX - R} ${CY} A ${R} ${R} 0 0 1 ${CX + R} ${CY}`" />
      <path v-if="isDay" class="travelled" :d="travelled" />
      <line class="horizon" x1="4" :y1="CY" x2="96" :y2="CY" />
      <circle class="dot" :class="{ night: !isDay }" :cx="sun.x" :cy="sun.y" :r="isDay ? 6 : 4" />
    </svg>

    <div class="times">
      <span v-for="event in upcoming" :key="event.kind" class="row">
        <svg class="icon" :class="event.kind" viewBox="0 0 24 24" aria-hidden="true">
          <path :d="`M3 20h18M7 20a5 5 0 0 1 10 0M12 3v7${event.kind === 'rise' ? 'M9 6l3-3 3 3' : 'M9 7l3 3 3-3'}`" />
        </svg>
        <span class="lcd">{{ format(event.at) }}</span>
      </span>
    </div>

    <div class="daylight">
      <span class="label">Daylight left</span>
      <span class="length">{{ daylight }}</span>
    </div>
  </div>
</template>

<style lang="scss" scoped>
  $amber: #e0a33a;
  $red: #f0643c;
  $muted: #8c93a6;
  $text: #f4f4f8;
  $mono: 'JetBrains Mono', 'Fira Code', Consolas, 'Courier New', monospace;

  .sun {
    display: flex;
    align-items: center;
    gap: 1.2cqw;
  }

  .arc {
    width: 5cqw;
    overflow: visible;
    fill: none;
    stroke-linecap: round;
  }

  .track {
    stroke: #3a3b42;
    stroke-width: 2.5;
    stroke-dasharray: 0.1 6;
  }

  .travelled {
    stroke: $amber;
    stroke-width: 3;
  }

  .horizon {
    stroke: #3a3b42;
    stroke-width: 2;
  }

  .dot {
    fill: $amber;
    filter: drop-shadow(0 0 4px $amber);

    &.night {
      fill: #55575f;
      filter: none;
    }
  }

  .times {
    display: flex;
    flex-direction: column;
    gap: 0.3cqw;
  }

  .row {
    display: flex;
    align-items: center;
    gap: 0.6cqw;
  }

  .icon {
    width: 1.1cqw;
    height: 1.1cqw;
    fill: none;
    stroke-width: 1.8;
    stroke-linecap: round;
    stroke-linejoin: round;

    &.rise {
      stroke: $amber;
    }

    &.set {
      stroke: $red;
    }
  }

  .lcd {
    font-family: 'Seven Segment', sans-serif;
    font-size: 1.3cqw;
    line-height: 1;
    letter-spacing: 0.05em;
  }

  .daylight {
    display: flex;
    flex-direction: column;
    gap: 0.4cqw;
  }

  .label {
    color: $muted;
    font-family: $mono;
    font-size: 0.6cqw;
    letter-spacing: 0.25em;
    text-transform: uppercase;
  }

  .length {
    color: $text;
    font-family: $mono;
    font-size: 1cqw;
    white-space: nowrap;
  }
</style>
