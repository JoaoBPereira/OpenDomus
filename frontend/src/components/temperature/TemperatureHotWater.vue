<script setup lang="ts">
  import { computed, inject } from 'vue';
  import { openChartKey } from '../../composables/useOpenChart';
  import { TAG_TEMP_HOT_WATER } from '../../constants/tagIds';
  import { formatDelta, temperatureTone, type TemperatureReading } from './temperatureData';

  const props = defineProps<{
    reading?: TemperatureReading;
  }>();

  const openChart = inject(openChartKey)!;

  const EMPTY = '\u2014';
  const MIN_TEMP = 20;
  const MAX_TEMP = 55;

  const value = computed(() => (props.reading?.error ? undefined : props.reading?.value));
  const temperature = computed(() => (value.value === undefined ? EMPTY : value.value.toFixed(1)));
  const delta = computed(() => formatDelta(props.reading?.delta));
  const trend = computed(() => {
    const change = props.reading?.delta ?? 0;
    return { warming: change > 0, cooling: change < 0 };
  });
  const style = computed(() => (value.value === undefined ? {} : { backgroundColor: temperatureTone(value.value, MIN_TEMP, MAX_TEMP) }));

  const status = computed(() => {
    const reading = props.reading;
    if (!reading) return 'Loading';
    if (reading.error) return 'Unavailable';
    if (reading.stale) return 'Stale reading';
    if (reading.historyError) return 'History unavailable';
    return '';
  });

  const description = computed(() => {
    const reading = props.reading;
    return [
      'Hot water',
      value.value !== undefined ? `${temperature.value} degrees Celsius` : status.value,
      `Last hour: ${delta.value}`,
      reading?.timestamp ? `Updated ${reading.timestamp.toLocaleString()}` : '',
      reading?.error ?? reading?.historyError ?? '',
      reading?.stale ? 'Reading is more than 24 hours old' : '',
    ]
      .filter(Boolean)
      .join('. ');
  });

  function onClick() {
    openChart(TAG_TEMP_HOT_WATER, 'Main Water', 20, 70);
  }
</script>

<template>
  <button type="button" class="tank" :class="{ stale: reading?.stale }" :style="style" :title="description" :aria-label="description" @click="onClick">
    <span class="tankLabel">Hot Water</span>
    <span class="temperature">{{ temperature }}<sup v-if="value !== undefined">&deg;</sup></span>
    <span class="delta" :class="trend">{{ delta }}</span>
    <span v-if="status" class="status">{{ status }}</span>
  </button>
</template>

<style lang="scss" scoped>
  $tank-bg: #28282c;
  $text: #f4f4f8;
  $muted: #a8b2c6;
  $warm: #e2ac69;
  $cool: #a0b8ec;
  $outline: #8f8a80;
  $warning: #edc777;

  // Rounded dome on top, slightly rounded base, like a hot-water cylinder.
  // Sized by width (capped by row height) so the shape never stretches.
  .tank {
    position: relative;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 1vh;
    width: min(100%, 17.7vh);
    aspect-ratio: 0.68;
    margin: 0;
    padding: 0;
    border: 0.12vw solid $outline;
    border-radius: 50% 50% 4% 4% / 34% 34% 3% 3%;
    background: $tank-bg;
    color: $text;
    font-family: 'Seven Segment', sans-serif;
    container-type: inline-size;
    cursor: pointer;

    @include mobile {
      width: min(100%, 13.6vh);
      gap: 0.5vh;
      border-width: 0.2vh;
    }

    &:hover {
      box-shadow: inset 0 0 2vw #ffffff12;
    }

    &:focus-visible {
      outline: 0.15vw solid #ffffff;
      outline-offset: 0.3vw;

      @include mobile {
        outline-width: 0.25vh;
      }
    }
  }

  // Sits in the dome, where the tank is still wide enough for the text.
  .tankLabel {
    position: absolute;
    top: 16%;
    left: 0;
    right: 0;
    font-size: 9cqw;
    letter-spacing: 0.1em;
    text-align: center;
    text-transform: uppercase;
    white-space: nowrap;
  }

  .temperature {
    font-size: 26cqw;
    line-height: 1;
    letter-spacing: 0.05em;
    white-space: nowrap;

    sup {
      vertical-align: top;
      font-size: 0.4em;
    }
  }

  .delta {
    color: $muted;
    font-size: 12cqw;
    white-space: nowrap;

    &.warming {
      color: $warm;
    }
    &.cooling {
      color: $cool;
    }
  }

  .status {
    color: $muted;
    font-size: 9cqw;
    text-transform: uppercase;
  }

  .stale {
    .temperature,
    .status {
      color: $warning;
    }
  }
</style>
