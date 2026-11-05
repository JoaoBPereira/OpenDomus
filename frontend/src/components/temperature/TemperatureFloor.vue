<script setup lang="ts">
  import { computed, inject } from 'vue';
  import { openChartKey } from '../../composables/useOpenChart';
  import { formatDelta, temperatureTone, type Room, type TemperatureReading, type TemperatureReadings } from './temperatureData';

  const props = defineProps<{
    floor: number;
    rooms: Room[];
    readings: TemperatureReadings;
  }>();

  const openChart = inject(openChartKey)!;

  const EMPTY = '\u2014';
  const MIN_TEMP = 16;
  const MAX_TEMP = 26;

  const roomViews = computed(() => props.rooms.map((room) => buildRoomView(room)));

  function buildRoomView(room: Room) {
    const reading = room.tagId ? props.readings[room.tagId] : undefined;
    const status = statusText(room, reading);
    const delta = reading?.delta;

    return {
      room,
      reading,
      status,
      hasValue: reading?.value !== undefined,
      temperature: reading?.value === undefined ? EMPTY : reading.value.toFixed(1),
      delta: formatDelta(delta),
      trend: { warming: (delta ?? 0) > 0, cooling: (delta ?? 0) < 0 },
      style: backgroundTone(reading),
      description: describe(room, reading, status, formatDelta(delta)),
    };
  }

  function statusText(room: Room, reading?: TemperatureReading) {
    if (!room.tagId) return 'No sensor';
    if (!reading) return 'Loading';
    if (reading.error) return 'Unavailable';
    if (reading.stale) return 'Stale reading';
    if (reading.historyError) return 'History unavailable';
    return '';
  }

  function backgroundTone(reading?: TemperatureReading) {
    if (reading?.value === undefined || reading.error) return {};
    return { backgroundColor: temperatureTone(reading.value, MIN_TEMP, MAX_TEMP) };
  }

  function describe(room: Room, reading: TemperatureReading | undefined, status: string, delta: string) {
    return [
      room.name,
      reading?.value !== undefined ? `${reading.value.toFixed(1)} degrees Celsius` : status,
      `Last hour: ${delta}`,
      reading?.timestamp ? `Updated ${reading.timestamp.toLocaleString()}` : '',
      room.shared ? 'Shared with Living Room' : '',
      reading?.error ?? reading?.historyError ?? '',
      reading?.stale ? 'Reading is more than 24 hours old' : '',
    ]
      .filter(Boolean)
      .join('. ');
  }

  function onRoomClick(room: Room) {
    if (room.tagId) openChart(room.tagId, room.name);
  }
</script>

<template>
  <section class="floor" :aria-label="`Floor ${floor} temperatures`">
    <div class="floorPlan" :class="floor === 0 ? 'groundPlan' : 'firstPlan'">
      <component
        :is="view.room.tagId ? 'button' : 'div'"
        v-for="view in roomViews"
        :key="view.room.area"
        :type="view.room.tagId ? 'button' : undefined"
        class="room"
        :class="[view.room.area, { noSensor: !view.room.tagId, stale: view.reading?.stale, unavailable: view.reading?.error }]"
        :style="view.style"
        :title="view.description"
        :aria-label="view.description"
        @click="onRoomClick(view.room)"
      >
        <span class="roomName">{{ view.room.name }}</span>

        <span v-if="view.room.tagId" class="roomReading">
          <span class="temperature">{{ view.temperature }}<sup v-if="view.hasValue">&deg;</sup></span>
          <span class="delta" :class="view.trend">
            {{ view.delta }}
          </span>
        </span>

        <span v-if="view.status" class="roomStatus">{{ view.status }}</span>
      </component>
    </div>
  </section>
</template>

<style lang="scss" scoped>
  $stripe-bg: #101013;
  $room-bg: #28282c;
  $text: #f4f4f8;
  $muted: #b6bdcb;
  $warm: #e2ac69;
  $cool: #a0b8ec;
  $warning: #edc777;

  // ── Floor ─────────────────────────────────────────────────────────────
  .floor {
    height: 100%;
    min-width: 0;
    color: $text;
    font-family: 'Seven Segment', sans-serif;
    container-type: inline-size;
  }

  // ── Floor plan grids ──────────────────────────────────────────────────
  // Fills the dashboard row on desktop; keeps a fixed shape when stacked on mobile.
  .floorPlan {
    display: grid;
    height: 100%;
    border: 0.08vw solid #a9a9ad;
    border-radius: 0.15vw;
    overflow: hidden;
    box-sizing: border-box;

    @include mobile {
      height: 40cqw;
      border-width: 0.1vh;
      border-radius: 0.2vh;
    }
  }

  .groundPlan {
    grid-template-columns: 1fr 2.9fr 1fr;
    grid-template-rows: 1fr 1fr;
  }

  .firstPlan {
    grid-template-columns: 3fr 2fr 2fr 0.8fr 3fr;
    grid-template-rows: 3fr 1fr;

    .temperature {
      font-size: 7cqw;
    }
  }

  // Ground floor
  .office {
    grid-area: 1 / 1 / 3 / 2;
  }
  .living {
    grid-area: 1 / 2 / 3 / 4;
  }
  .kitchen {
    grid-area: 1 / 3 / 2 / 4;
  }

  // First floor
  .main {
    grid-area: 1 / 1 / 2 / 2;
  }
  .eduardo {
    grid-area: 1 / 2 / 2 / 3;
  }
  .filipe {
    grid-area: 1 / 3 / 2 / 4;
  }
  .wc {
    grid-area: 1 / 4 / 2 / 5;
  }
  .cave {
    grid-area: 1 / 5 / 3 / 6;
  }
  .hallway {
    grid-area: 2 / 1 / 5 / 5;
  }

  // ── Room tile ─────────────────────────────────────────────────────────
  .room {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 0;
    margin: 0;
    padding: 2.5vh 0.3vw;
    border: 0.08vw solid #929294;
    border-radius: 0;
    background: $room-bg;
    color: inherit;
    font: inherit;
    text-align: center;

    @include mobile {
      border-width: 0.1vh;
    }
  }

  button.room {
    cursor: pointer;

    &:hover {
      box-shadow: inset 0 0 2vw #ffffff12;
    }

    &:focus-visible {
      z-index: 2;
      outline: 0.15vw solid #ffffff;
      outline-offset: -0.3vw;

      @include mobile {
        outline-width: 0.25vh;
        outline-offset: -0.5vh;
      }
    }
  }

  .roomName,
  .roomStatus {
    position: absolute;
    left: 0.5vw;
    right: 0.35vw;
    text-align: left;
    text-transform: uppercase;
    overflow-wrap: anywhere;

    @include mobile {
      left: 1.5vw;
      right: 1vw;
    }
  }

  .roomName {
    top: 1vh;
    font-size: 2.7cqw;
    letter-spacing: 0.12em;
  }

  .roomStatus {
    bottom: 0.8vh;
    color: $muted;
    font-size: 2.4cqw;
    letter-spacing: 0.07em;
  }

  // ── Reading ───────────────────────────────────────────────────────────
  .roomReading {
    display: flex;
    flex-direction: column;
    gap: 0.8vh;
  }

  .temperature {
    font-size: 7cqw;
    line-height: 1;
    letter-spacing: 0.03em;
    white-space: nowrap;

    sup {
      vertical-align: top;
      font-size: 0.45em;
    }
  }

  .delta {
    color: #a8b2c6;
    font-size: 3.1cqw;
    white-space: nowrap;

    &.warming {
      color: $warm;
    }
    &.cooling {
      color: $cool;
    }
  }

  // Smaller rooms use slightly smaller temperatures.
  .kitchen .temperature {
    font-size: 7cqw;
  }

  // Short tile: temperature and change sit side by side with less padding.
  .hallway {
    padding-block: 0.5vh;

    .roomReading {
      flex-direction: row;
      align-items: baseline;
      gap: 0.6vw;

      @include mobile {
        gap: 1.5vw;
      }
    }
    .temperature {
      font-size: 7cqw;
    }
  }

  // ── States ────────────────────────────────────────────────────────────
  .noSensor {
    background: repeating-linear-gradient(45deg, $stripe-bg, $stripe-bg 0.5vh, #25252d 0.5vh, #25252d 0.6vh);

    .roomName {
      left: 50%;
      right: auto;
      transform: translateX(-50%);
      white-space: nowrap;
    }

    .roomStatus {
      top: 50%;
      bottom: auto;
      left: 50%;
      right: auto;
      transform: translate(-50%, -50%);
      writing-mode: vertical-rl;
      white-space: nowrap;
    }
  }

  .stale {
    .temperature,
    .roomStatus {
      color: $warning;
    }
  }

  .unavailable .temperature {
    color: $muted;
  }
</style>
