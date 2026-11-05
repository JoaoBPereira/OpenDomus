<script setup lang="ts">
  import { computed, inject, onBeforeUnmount, onMounted, ref } from 'vue';

  import { getLastTagValueBoolean, getLatestTagValueFloat } from '../../services/api';
  import { openChartKey } from '../../composables/useOpenChart';

  const props = defineProps<{
    label: string;
    batteryTagId: number;
    plugTagId: number;
  }>();

  const openChart = inject(openChartKey)!;

  const EMPTY = '\u2014';
  const SEGMENTS = 10;

  const battery = ref<number>();
  const pluggedIn = ref(false);

  let intervalId: number | undefined;

  const fetchData = () => {
    getLatestTagValueFloat(props.batteryTagId).then((json) => {
      battery.value = Number(json.value);
    });

    getLastTagValueBoolean(props.plugTagId, 1).then((json) => {
      pluggedIn.value = Boolean(json[0]?.value);
    });
  };

  // Any charge lights the first segment, matching the old display.
  const litSegments = computed(() => {
    if (battery.value === undefined) return 0;
    return Math.min(Math.ceil(battery.value / (100 / SEGMENTS)), SEGMENTS);
  });

  const batteryText = computed(() => (battery.value === undefined ? EMPTY : Math.round(battery.value).toString()));
  const plugText = computed(() => (pluggedIn.value ? 'Plugged in' : 'Unplugged'));
  const description = computed(() => `${props.label} battery ${batteryText.value}%. ${plugText.value}. Show chart`);

  function onCardClick() {
    openChart(props.batteryTagId.toString(), props.label, 0, 100);
  }

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
  <button type="button" class="carCard" :class="{ pluggedIn }" :aria-label="description" @click="onCardClick">
    <h2>{{ label }}</h2>

    <span class="display">
      <span class="content">
        <span class="reading">
          <span class="battery">
            <span v-for="index in SEGMENTS" :key="index" class="segment" :class="{ lit: index <= litSegments, last: index === litSegments }" />
          </span>

          <span class="percentage"> {{ batteryText }}<sup v-if="battery !== undefined">%</sup> </span>

          <span class="charger">
            <span class="screen" />
            <svg viewBox="0 0 24 24" aria-hidden="true">
              <path d="M13 2 4 14h7l-1 8 9-12h-7z" />
            </svg>
          </span>
        </span>

        <span class="status">
          <svg viewBox="0 0 24 24" aria-hidden="true">
            <path d="M9 2v6M15 2v6M6 8h12v3a6 6 0 0 1-12 0zM12 17v5" />
          </svg>
          {{ plugText }}
        </span>
      </span>
    </span>
  </button>
</template>

<style lang="scss" scoped>
  $card-bg: #101013;
  $text: #f4f4f8;
  $muted: #8c93a6;
  $track: #1f2027;
  $charge: #9ccc85;
  $teal: #4fc3a1;
  $teal-light: #7ddcb4;
  $charger-bg: #22242a;
  $charger-off: #34363e;
  $outline: #e8e9ee;

  // ── Card ──────────────────────────────────────────────────────────────
  // Same padding and title style as the Gate and PC cards.
  .carCard {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    width: 26vw;
    min-height: 0;
    margin: 0;
    padding: 1.2vh 1.2vw 2.4vh;
    border: 0.08vw solid #25252b;
    border-radius: 0.8vw;
    background: $card-bg;
    color: $text;
    font-family: 'Seven Segment', sans-serif;
    container-type: inline-size;
    box-sizing: border-box;
    cursor: pointer;

    @include mobile {
      width: 46vw;
      padding: 1vh 3vw 2vh;
      border-width: 0.1vh;
      border-radius: 1vh;
    }

    &:focus-visible {
      outline: 0.15vw solid #ffffff;
      outline-offset: 0.2vw;
    }

    h2 {
      align-self: flex-start;
      flex-shrink: 0;
      margin: 0;
      font-size: 1.4vh;
      letter-spacing: 0.25em;
      text-transform: uppercase;
    }
  }

  .display {
    flex: 1;
    width: 100%;
    min-height: 0;
    aspect-ratio: 100 / 34;
    container-type: size;
  }

  .content {
    // Fit the complete display to the available width and height.
    --unit: min(1cqw, 3cqh);

    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: calc(4 * var(--unit));
    width: 100%;
    height: 100%;

    > * {
      flex-shrink: 0;
    }
  }

  // ── Battery | percentage | charger ────────────────────────────────────
  .reading {
    display: flex;
    align-items: center;
    gap: calc(5 * var(--unit));
  }

  .battery {
    position: relative;
    display: flex;
    gap: calc(0.6 * var(--unit));
    width: calc(42 * var(--unit));
    height: calc(6.5 * var(--unit));
    padding: calc(0.8 * var(--unit));
    border: calc(0.35 * var(--unit)) solid $outline;
    border-radius: var(--unit);
    box-shadow: 0 0 calc(8 * var(--unit)) rgba($charge, 0.08);
    box-sizing: border-box;

    // The terminal nub on the right.
    &::after {
      content: '';
      position: absolute;
      top: 50%;
      right: calc(-1.4 * var(--unit));
      width: var(--unit);
      height: 40%;
      border-radius: 0 calc(0.4 * var(--unit)) calc(0.4 * var(--unit)) 0;
      background: $outline;
      transform: translateY(-50%);
    }
  }

  .segment {
    flex: 1;
    border-radius: calc(0.4 * var(--unit));
    background: $track;
    transition:
      background 0.4s ease,
      box-shadow 0.4s ease;

    &.lit {
      background: $charge;
    }

    &.last {
      box-shadow: 0 0 calc(1.5 * var(--unit)) $charge;
    }
  }

  .percentage {
    color: $charge;
    font-size: calc(14 * var(--unit));
    line-height: 1;
    letter-spacing: 0.05em;
    text-shadow: 0 0 calc(2 * var(--unit)) rgba($charge, 0.5);
    white-space: nowrap;

    sup {
      margin-left: calc(0.5 * var(--unit));
      color: $text;
      font-size: 0.3em;
      text-shadow: none;
      vertical-align: top;
    }
  }

  // ── Charger ───────────────────────────────────────────────────────────
  .charger {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: calc(2.5 * var(--unit));
    width: calc(9 * var(--unit));
    height: calc(17 * var(--unit));
    padding-top: calc(3 * var(--unit));
    border: calc(0.35 * var(--unit)) solid #c9ccd3;
    border-radius: calc(1.5 * var(--unit));
    background: $charger-bg;
    box-sizing: border-box;

    svg {
      width: calc(4 * var(--unit));
      fill: none;
      stroke: $charger-off;
      stroke-width: 2;
      stroke-linejoin: round;
      transition:
        stroke 0.4s ease,
        filter 0.4s ease;
    }
  }

  .screen {
    width: 70%;
    height: calc(3 * var(--unit));
    border-radius: calc(0.5 * var(--unit));
    background: $charger-off;
    transition:
      background 0.4s ease,
      box-shadow 0.4s ease;
  }

  // ── Plug status pill ──────────────────────────────────────────────────
  .status {
    display: inline-flex;
    align-items: center;
    gap: calc(1.5 * var(--unit));
    padding: var(--unit) calc(3.5 * var(--unit));
    border: calc(0.25 * var(--unit)) solid $muted;
    border-radius: 999px;
    background: rgba($muted, 0.08);
    color: $muted;
    font-size: calc(2.8 * var(--unit));
    letter-spacing: 0.25em;
    text-transform: uppercase;
    transition:
      color 0.4s ease,
      border-color 0.4s ease,
      background 0.4s ease,
      box-shadow 0.4s ease;

    @include mobile {
      font-size: calc(4.5 * var(--unit));
    }

    svg {
      width: 1.2em;
      fill: none;
      stroke: currentColor;
      stroke-width: 2;
      stroke-linecap: round;
      stroke-linejoin: round;
    }
  }

  // ── Plugged in ────────────────────────────────────────────────────────
  .pluggedIn {
    .screen {
      background: $teal;
      box-shadow: 0 0 calc(1.5 * var(--unit)) rgba($teal, 0.6);
    }

    .charger svg {
      stroke: $teal;
      filter: drop-shadow(0 0 calc(0.6 * var(--unit)) rgba($teal, 0.6));
    }

    .status {
      border-color: $teal;
      background: rgba($teal, 0.15);
      color: $teal-light;
      box-shadow: 0 0 calc(2 * var(--unit)) rgba($teal, 0.35);
    }
  }
</style>
