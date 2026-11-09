<script setup lang="ts">
  import { computed } from 'vue';

  const props = withDefaults(
    defineProps<{
      value?: number;
      timestamp?: Date;
      minTemp?: number;
      maxTemp?: number;
      unit?: 'C' | 'F';
    }>(),
    {
      value: 0,
      timestamp: () => new Date(),
      minTemp: 0,
      maxTemp: 40,
      unit: 'C',
    },
  );

  const heightPercent = computed(() => {
    const clamped = Math.min(Math.max(props.value, props.minTemp), props.maxTemp);
    return ((clamped - props.minTemp) / (props.maxTemp - props.minTemp)) * 100;
  });

  const label = computed(() => `${props.value}`);

  const isOlderThan24Hours = computed(() => {
    if (!props.timestamp) return false;
    const now = new Date();
    const diff = now.getTime() - props.timestamp.getTime();
    return diff > 24 * 60 * 60 * 1000; // 24 hours in milliseconds
  });
</script>

<template>
  <div class="thermometer-wrapper">
    <div :class="isOlderThan24Hours ? 'thermometer-yellow' : 'thermometer'">
      <div class="temperature" :style="{ height: heightPercent + '%' }" :data-value="label" />
      <div class="graduations" />
    </div>
  </div>
</template>

<style scoped>
  .thermometer-wrapper {
    height: 100%;
    width: 100%;
    cursor: pointer;
  }

  .thermometer {
    left: 50%;
    transform: translateX(-50%);
    width: 25%;
    background: #38383f;
    height: 94%;
    position: relative;
    border: 0.25vw solid #2a2a2e;
    border-radius: 1vw;
    z-index: 1;
    container-type: inline-size;
  }

  .thermometer-yellow {
    left: 50%;
    transform: translateX(-50%);
    width: 25%;
    background: #fffb00;
    height: 94%;
    position: relative;
    border: 0.25vw solid #2a2a2e;
    border-radius: 1vw;
    z-index: 1;
    container-type: inline-size;
  }

  .thermometer::before,
  .thermometer::after {
    position: absolute;
    content: '';
    border-radius: 50%;
  }

  .thermometer::after {
    transform: translateX(-50%);
    width: 150%;
    height: 20%;
    background-color: #3dcadf;
    bottom: -0.35vw;
    border: 0.25vw solid #2a2a2e;
    z-index: -3;
    left: 50%;
  }

  .graduations {
    height: 59%;
    top: 20%;
    width: 50%;
  }

  .graduations,
  .graduations::before {
    position: absolute;
    border-top: 0.25vh solid rgba(0, 0, 0, 0.5);
    border-bottom: 0.25vh solid rgba(0, 0, 0, 0.5);
  }

  .graduations::before {
    content: '';
    height: 34%;
    width: 100%;
    top: 32%;
    left: 0%;
  }

  .temperature {
    bottom: 0;
    background: linear-gradient(#ff0000, #3dcadf9f) no-repeat bottom;
    width: 100%;
    border-radius: 1vw;
    background-size: 100% 240px;
    transition: height 0.2s ease-in-out;
  }

  .temperature,
  .temperature::before,
  .temperature::after {
    position: absolute;
  }

  .temperature::before {
    content: attr(data-value);
    background: rgba(255, 255, 255, 0.7);
    color: black;
    z-index: 2;
    padding: 0.1vw 0.25vh;
    border-radius: 0.1vw;
    font-size: 120cqw;
    font-family: 'Seven Segment', sans-serif;
    font-weight: bold;
    top: 50%;
    left: 50%;
    transform: translate(-50%, -50%);
    white-space: nowrap;
  }
</style>
