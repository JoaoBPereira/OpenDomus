<script setup lang="ts">
  import { computed, inject, onBeforeUnmount, onMounted, ref } from 'vue';
  import { openChartTimeSeriesKey } from '../../composables/useOpenChart';
  import { PowerData } from '../../models';
  import { getLatestTagValueFloat, getTags, Tag } from '../../services/api';
  import { TAG_POWER_TOTAL } from '../../constants/tagIds';

  // With `data` (the per-minute breakdown modal) it shows that data; otherwise it loads the live devices.
  const props = defineProps<{
    data?: PowerData[];
  }>();

  const openChart = inject(openChartTimeSeriesKey)!;

  const baseUrl = import.meta.env.BASE_URL;

  const UNKNOWN_TAG_ID = 0;
  const MIN_DEVICE_WATTS = 5;
  const MAX_READING_AGE = 5 * 60 * 1000;
  // The largest bar fills this share of the track, leaving room for its value pill.
  const LARGEST_BAR_SHARE = 0.87;

  const powerTags: Tag[] = [];
  const livePower = ref<PowerData[]>([]);
  const hiddenIcons = ref(new Set<number>());

  let intervalId: number | undefined;

  // ── Live data ─────────────────────────────────────────────────────────
  const fetchInitialData = async () => {
    const tags = await getTags();
    powerTags.length = 0;
    powerTags.push(...tags.filter((tag) => tag.displayIn.includes('PowerBarChartKHH')));
    await fetchData();
  };

  const fetchData = async () => {
    const total = await getLatestTagValueFloat(TAG_POWER_TOTAL);
    const devices: PowerData[] = [];

    for (const tag of powerTags) {
      const reading = await getLatestTagValueFloat(tag.id);
      const time = new Date(reading.timestamp ?? reading.time).getTime();
      if (reading.value > MIN_DEVICE_WATTS && Date.now() - time < MAX_READING_AGE) {
        devices.push({ time, power: Math.round(reading.value), name: tag.name, percentage: 0, color: '', tagId: tag.id });
      }
    }

    // Unknown is whatever the total measures beyond the known devices.
    const known = devices.reduce((sum, device) => sum + device.power, 0);
    const unknown = Math.round(total.value - known);
    if (unknown > 0) {
      devices.push({ time: Date.parse(total.timestamp ?? total.time), power: unknown, name: 'Unknown', percentage: 0, color: '', tagId: UNKNOWN_TAG_ID });
    }

    livePower.value = devices;
  };

  // ── Rows ──────────────────────────────────────────────────────────────
  const rows = computed(() => {
    const items = (props.data?.length ? props.data : livePower.value).filter((item) => item.tagId !== TAG_POWER_TOTAL && item.power > 0);
    const total = items.reduce((sum, item) => sum + item.power, 0);
    const largest = Math.max(...items.map((item) => item.power), 1);

    return [...items]
      .sort((a, b) => b.power - a.power)
      .map((item) => {
        const fill = (item.power / largest) * LARGEST_BAR_SHARE * 100;
        return {
          item,
          name: item.tagId === UNKNOWN_TAG_ID ? 'Unknown' : item.name,
          isUnknown: item.tagId === UNKNOWN_TAG_ID,
          share: total ? Math.round((item.power / total) * 100) : 0,
          // The gradient spans the whole track, so longer bars reach warmer colours.
          fillStyle: { width: `${fill}%`, backgroundSize: `${(100 / fill) * 100}% 100%` },
        };
      });
  });

  function onRowClick(item: PowerData) {
    if (item.tagId === UNKNOWN_TAG_ID) return;
    openChart(item.tagId.toString(), item.name, 0, 0);
  }

  function hideIcon(tagId: number) {
    hiddenIcons.value = new Set(hiddenIcons.value).add(tagId);
  }

  onMounted(() => {
    if (props.data) return;
    fetchInitialData();
    intervalId = window.setInterval(() => {
      void fetchData();
    }, 30000);
  });

  onBeforeUnmount(() => {
    if (intervalId) {
      clearInterval(intervalId);
    }
  });
</script>

<template>
  <div class="BarChartKHH">
    <h2>Devices</h2>

    <TransitionGroup tag="div" name="row" class="rows">
      <button
        v-for="row in rows"
        :key="row.item.tagId"
        type="button"
        class="row"
        :class="{ unknown: row.isUnknown }"
        :aria-label="`${row.name}: ${row.item.power} W, ${row.share}%`"
        :title="row.name"
        @click="onRowClick(row.item)"
      >
        <span class="icon">
          <img v-if="!hiddenIcons.has(row.item.tagId)" :src="`${baseUrl}icons/${row.item.tagId}.svg`" alt="" @error="hideIcon(row.item.tagId)" />
        </span>
        <span class="track">
          <span class="fill" :style="row.fillStyle">
            <span class="value">{{ row.item.power }}<small>W</small></span>
          </span>
        </span>
        <span class="share">{{ row.share }}%</span>
      </button>
    </TransitionGroup>
  </div>
</template>

<style lang="scss" scoped>
  $card-bg: #101013;
  $track-bg: #1d1e23;
  $text: #f4f4f8;
  $share: #8fa6f0;
  $green: #74d39a;
  $yellow: #e3c51a;
  $orange: #d9873e;

  // ── Card ──────────────────────────────────────────────────────────────
  .BarChartKHH {
    display: flex;
    flex-direction: column;
    gap: 0.8vh;
    height: 100%;
    margin: 0 0.5vw 0 0;
    padding: 1.2vh 1.2vw;
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
      padding: 1vh 3vw;
      border-width: 0.1vh;
      border-radius: 1vh;
    }

    h2 {
      margin: 0;
      font-size: 1.4vh;
      letter-spacing: 0.25em;
      text-transform: uppercase;
    }
  }

  // ── Rows ──────────────────────────────────────────────────────────────
  .rows {
    position: relative;
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 0.6vh;
    min-height: 0;
  }

  // Rows slide to their new place when the order changes, and fade in or out
  // when a device starts or stops drawing power.
  .row-move,
  .row-enter-active,
  .row-leave-active {
    transition:
      transform 0.6s ease,
      opacity 0.4s ease;
  }

  .row-enter-from,
  .row-leave-to {
    opacity: 0;
  }

  // Taken out of the layout so the remaining rows can slide into its place.
  .row-leave-active {
    position: absolute;
    width: 100%;
  }

  .row {
    display: grid;
    grid-template-columns: 2.5vh 1fr 1vw;
    align-items: center;
    gap: 0.6vw;
    flex: 1;
    max-height: 4vh;
    min-height: 0;
    margin: 0;
    padding: 0;
    border: 0;
    background: none;
    color: inherit;
    font: inherit;
    text-align: left;
    cursor: pointer;

    @include mobile {
      grid-template-columns: 2.5vh 1fr 8vw;
      gap: 1.5vw;
    }

    &:hover .fill {
      filter: brightness(1.1);
    }

    &:focus-visible {
      outline: 0.15vw solid #ffffff;
      outline-offset: 0.2vw;
    }

    &.unknown {
      cursor: default;
    }
  }

  .icon {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;

    img {
      height: 100%;
      max-height: 2.5vh;
      opacity: 0.6;
    }
  }

  // ── Bar ───────────────────────────────────────────────────────────────
  .track {
    height: 100%;
    max-height: 3vh;
    border-radius: 0.3vw;
    background: $track-bg;

    @include mobile {
      border-radius: 0.6vh;
    }
  }

  .fill {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: flex-end;
    min-width: max-content;
    height: 100%;
    border-radius: inherit;
    background-image: linear-gradient(90deg, $green, $yellow 65%, $orange);
    background-repeat: no-repeat;
    // Background size changes with the width, keeping the gradient fixed to the track.
    transition:
      width 0.8s ease,
      background-size 0.8s ease;
    animation: grow 0.8s ease;
  }

  // New bars grow in from the left; only "from" is set, so it ends at the bar's real width.
  @keyframes grow {
    from {
      width: 0;
    }
  }

  // Unknown is drawn hatched, since it isn't a measured device.
  .unknown .fill {
    background-image: repeating-linear-gradient(-45deg, rgba(16, 16, 19, 0.55) 0 0.35vw, transparent 0.35vw 0.7vw), linear-gradient(90deg, $green, $yellow 65%, $orange);

    @include mobile {
      background-image: repeating-linear-gradient(-45deg, rgba(16, 16, 19, 0.55) 0 0.8vh, transparent 0.8vh 1.6vh), linear-gradient(90deg, $green, $yellow 65%, $orange);
    }
  }

  .value {
    margin-right: 0.3vw;
    padding: 0.1vh 0.4vw;
    border: 0.08vw solid rgba(0, 0, 0, 0.35);
    border-radius: 0.3vw;
    background: rgba(255, 255, 255, 0.55);
    color: #111114;
    font-size: 1.9vh;
    line-height: 1.1;
    white-space: nowrap;

    @include mobile {
      margin-right: 1vw;
      padding: 0.1vh 1.2vw;
      border-radius: 0.6vh;
    }

    small {
      margin-left: 0.15em;
      font-size: 0.55em;
    }
  }

  .share {
    color: $share;
    font-size: 1.2vh;
    text-align: right;
  }
</style>
