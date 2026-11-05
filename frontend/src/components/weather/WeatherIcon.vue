<script setup lang="ts">
  import { computed } from 'vue';

  // IPMA weather type ids, grouped into the icons we draw.
  const props = defineProps<{
    typeId: number;
    night?: boolean;
  }>();

  const kind = computed(() => {
    const id = props.typeId;
    if (id === 1) return 'clear';
    if ([2, 3, 25].includes(id)) return 'partly';
    if ([19, 20, 23].includes(id)) return 'storm';
    if ([18, 21, 22, 28, 29, 30].includes(id)) return 'snow';
    if ([16, 17, 26].includes(id)) return 'fog';
    if (id >= 6 && id <= 15) return 'rain';
    return 'cloudy';
  });

  const CLOUD = 'M7.5 18H17a4 4 0 0 0 .4-7.98A5.5 5.5 0 0 0 6.9 11.2 3.4 3.4 0 0 0 7.5 18z';
</script>

<template>
  <svg class="weatherIcon" viewBox="0 0 24 24" aria-hidden="true">
    <template v-if="kind === 'clear' && !night">
      <circle class="sun" cx="12" cy="12" r="4" />
      <path class="sun" d="M12 2.5v2M12 19.5v2M2.5 12h2M19.5 12h2M5.3 5.3l1.4 1.4M17.3 17.3l1.4 1.4M18.7 5.3l-1.4 1.4M6.7 17.3l-1.4 1.4" />
    </template>

    <path v-else-if="kind === 'clear'" class="moon" d="M19.5 14.5A8 8 0 1 1 9.5 4.5a6.2 6.2 0 0 0 10 10z" />

    <template v-else-if="kind === 'partly'">
      <template v-if="!night">
        <circle class="sun" cx="16.5" cy="7.5" r="2.6" />
        <path class="sun" d="M16.5 2.6v1.1M21.4 7.5h-1.1M20 4l-.8.8M13 4l.8.8M20 11l-.8-.8" />
      </template>
      <path v-else class="moon" d="M21 9.5A4.5 4.5 0 1 1 15.5 3.5a3.6 3.6 0 0 0 5.5 6z" />
      <path class="cloud" :d="CLOUD" transform="translate(-2 2)" />
    </template>

    <template v-else-if="kind === 'rain'">
      <path class="cloud" :d="CLOUD" transform="translate(0 -3)" />
      <path class="rain" d="M9 18l-1 2.5M12.5 18l-1 2.5M16 18l-1 2.5" />
    </template>

    <template v-else-if="kind === 'storm'">
      <path class="cloud" :d="CLOUD" transform="translate(0 -3)" />
      <path class="bolt" d="M12.5 16l-2 3.2h3l-2 3.3" />
    </template>

    <template v-else-if="kind === 'snow'">
      <path class="cloud" :d="CLOUD" transform="translate(0 -3)" />
      <path class="snow" d="M9 19h.01M12.5 20.5h.01M16 19h.01" />
    </template>

    <template v-else-if="kind === 'fog'">
      <path class="cloud" :d="CLOUD" transform="translate(0 -3)" />
      <path class="fog" d="M5 18.5h14M7 21h10" />
    </template>

    <path v-else class="cloud" :d="CLOUD" />
  </svg>
</template>

<style lang="scss" scoped>
  .weatherIcon {
    fill: none;
    stroke-width: 1.4;
    stroke-linecap: round;
    stroke-linejoin: round;
  }

  .sun,
  .bolt {
    stroke: #f2c94c;
  }

  .moon {
    stroke: #aab4f5;
  }

  .cloud {
    stroke: #c9cdd8;
  }

  .rain {
    stroke: #8fa6f0;
  }

  .snow {
    stroke: #e3e9ff;
    stroke-width: 2.4;
  }

  .fog {
    stroke: #8c93a6;
  }
</style>
