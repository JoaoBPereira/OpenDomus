<script setup lang="ts">
  import { computed, inject } from 'vue';
  import { openChartKey } from '../../composables/useOpenChart';
  import { TAG_TEMP_OUTSIDE } from '../../constants/tagIds';
  import { formatDelta, temperatureTone, type TemperatureReading } from './temperatureData';

  const props = defineProps<{
    reading?: TemperatureReading;
  }>();

  const openChart = inject(openChartKey)!;

  const EMPTY = '\u2014';
  const MIN_TEMP = 0;
  const MAX_TEMP = 40;

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
      'Outside',
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
    openChart(TAG_TEMP_OUTSIDE, 'Outside', 0, 50);
  }
</script>

<template>
  <button type="button" class="outsideCard" :class="{ stale: reading?.stale }" :style="style" :title="description" :aria-label="description" @click="onClick">
    <span class="outsideLabel">Outside</span>

    <span class="outsideReading">
      <span class="temperature">{{ temperature }}<sup v-if="value !== undefined">&deg;</sup></span>
      <span class="delta" :class="trend">{{ delta }}</span>
    </span>

    <span v-if="status" class="outsideStatus">{{ status }}</span>
  </button>
</template>
<style lang="scss" scoped>
  $card-bg: #101013;
  $text: #f4f4f8;
  $muted: #a8b2c6;
  $warm: #e2ac69;
  $cool: #a0b8ec;
  $warning: #edc777;

  .outsideCard {
    display: flex;
    flex-direction: column;
    width: 100%;
    height: 100%;
    margin: 0;
    padding: 1.2vh 1.2vw;
    border: 0.08vw solid #25252b;
    border-radius: 0.8vw;
    background: $card-bg;
    color: $text;
    font-family: 'Seven Segment', sans-serif;
    container-type: inline-size;
    cursor: pointer;

    @include mobile {
      min-height: 12vh;
      border-width: 0.1vh;
      border-radius: 1vh;
    }

    &:hover {
      box-shadow: inset 0 0 2vw #ffffff12;
    }

    &:focus-visible {
      outline: 0.15vw solid #ffffff;
      outline-offset: -0.3vw;

      @include mobile {
        outline-width: 0.25vh;
        outline-offset: -0.5vh;
      }
    }
  }

  .outsideLabel,
  .outsideStatus {
    align-self: flex-start;
    text-transform: uppercase;
  }

  .outsideLabel {
    font-size: 10cqw;
    letter-spacing: 0.2em;

    @include mobile {
      font-size: 7cqw;
    }
  }

  .outsideStatus {
    color: $muted;
    font-size: 9cqw;
    letter-spacing: 0.07em;

    @include mobile {
      font-size: 6cqw;
    }
  }

  // Fills the space between the label and status, keeping the reading centered.
  .outsideReading {
    display: flex;
    flex: 1;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 0.8vh;
  }

  // Capped by height so short screens don't overlap the label.
  .temperature {
    font-size: min(26cqw, 9vh);
    line-height: 1;
    letter-spacing: 0.03em;
    white-space: nowrap;

    sup {
      vertical-align: top;
      font-size: 0.45em;
    }
  }

  .delta {
    color: $muted;
    font-size: min(11cqw, 2.8vh);
    white-space: nowrap;

    &.warming {
      color: $warm;
    }
    &.cooling {
      color: $cool;
    }
  }

  .stale {
    .temperature,
    .outsideStatus {
      color: $warning;
    }
  }
</style>
