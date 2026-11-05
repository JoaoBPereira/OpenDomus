<script setup lang="ts">
  import { computed, onMounted, ref } from 'vue';

  import Modal from '../../components/ui/Modal.vue';
  import { getKwh, startOfDay, startOfMonth } from './powerEnergy';

  const props = defineProps<{
    name: string;
    tagId: number;
    priceKwh: number;
    scale: string;
    // The whole home's consumption this month; omitted for the home total itself.
    homeMonthKwh?: number;
  }>();

  const baseUrl = import.meta.env.BASE_URL;

  const TEAL = [79, 195, 161];
  const AMBER = [224, 163, 58];
  const RED = [240, 100, 60];
  // Share of the home's month: teal until 10%, amber at 25%, orange-red from 40%.
  const SHARE_STOPS = [10, 25, 40];

  const thisDayKWH = ref(0);
  const thisWeekKWH = ref(0);
  const thisMonthKWH = ref(0);
  const lastDayKWH = ref(0);
  const lastWeekKWH = ref(0);
  const lastMonthKWH = ref(0);
  const showDialog = ref(false);

  const fetchData = async () => {
    const today = startOfDay();

    const yesterday = new Date(today);
    yesterday.setDate(yesterday.getDate() - 1);

    const monday = new Date(today);
    monday.setDate(monday.getDate() - ((monday.getDay() + 6) % 7));

    const lastMonday = new Date(monday);
    lastMonday.setDate(lastMonday.getDate() - 7);

    const thisMonth = startOfMonth();
    const lastMonth = new Date(thisMonth);
    lastMonth.setMonth(lastMonth.getMonth() - 1);

    thisDayKWH.value = await getKwh(props.tagId, props.scale, today);
    lastDayKWH.value = await getKwh(props.tagId, props.scale, yesterday, today);
    thisWeekKWH.value = await getKwh(props.tagId, props.scale, monday);
    lastWeekKWH.value = await getKwh(props.tagId, props.scale, lastMonday, monday);
    thisMonthKWH.value = await getKwh(props.tagId, props.scale, thisMonth);
    lastMonthKWH.value = await getKwh(props.tagId, props.scale, lastMonth, thisMonth);
  };

  // ── Share of the home ─────────────────────────────────────────────────
  function mix(from: number[], to: number[], amount: number) {
    return from.map((channel, index) => Math.round(channel + (to[index] - channel) * amount));
  }

  function shareColor(share: number) {
    const [warm, amber, hot] = SHARE_STOPS;
    let color = TEAL;
    if (share >= hot) color = RED;
    else if (share >= amber) color = mix(AMBER, RED, (share - amber) / (hot - amber));
    else if (share > warm) color = mix(TEAL, AMBER, (share - warm) / (amber - warm));
    return `rgb(${color.join(', ')})`;
  }

  const share = computed(() => {
    if (!props.homeMonthKwh) return undefined;
    return Math.min(Math.max((thisMonthKWH.value / props.homeMonthKwh) * 100, 0), 100);
  });

  const shareStyle = computed(() => {
    if (share.value === undefined) return undefined;
    const color = shareColor(share.value);
    return { width: `${share.value}%`, background: color, color };
  });

  const formatCost = (kwh: number) => (kwh * props.priceKwh).toFixed(2);

  function closeDialog() {
    showDialog.value = false;
  }

  onMounted(() => {
    fetchData();
  });
</script>

<template>
  <div class="powerConsumptionDailyMain">
    <button type="button" class="card" :aria-label="`${name}: ${thisDayKWH.toFixed(2)} kWh today. Show details`" @click="showDialog = true">
      <span class="today">
        <img class="icon" :src="`${baseUrl}icons/${tagId}.svg`" :alt="name" />
        <span class="todayReading">
          <span class="label">Today</span>
          <span class="todayKwh">
            {{ thisDayKWH.toFixed(2) }}
            <small>kWh</small>
          </span>
          <span class="todayCost">
            {{ formatCost(thisDayKWH) }}
            <small>&euro;</small>
          </span>
        </span>
      </span>

      <span class="periods">
        <span />
        <span class="columnHead">kWh</span>
        <span class="columnHead">&euro;</span>

        <span class="label">Week</span>
        <span class="kwh">{{ thisWeekKWH.toFixed(2) }}</span>
        <span class="cost">{{ formatCost(thisWeekKWH) }}</span>

        <span class="label">Month</span>
        <span class="kwh">{{ thisMonthKWH.toFixed(2) }}</span>
        <span class="cost">{{ formatCost(thisMonthKWH) }}</span>
      </span>

      <span v-if="share !== undefined" class="share">
        <span class="shareTrack">
          <span class="shareFill" :style="shareStyle" />
        </span>
        <span class="shareCaption">
          <span>of home &middot; month</span>
          <span :style="{ color: shareStyle?.color }">{{ Math.round(share) }}%</span>
        </span>
      </span>
    </button>

    <Modal :open="showDialog" :title="name" @close="closeDialog">
      <table class="powerConsumptionDailyTable">
        <thead>
          <tr>
            <th>Period</th>
            <th>kWh</th>
            <th>€</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td>Today</td>
            <td>{{ thisDayKWH.toFixed(2) }}</td>
            <td>{{ formatCost(thisDayKWH) }}</td>
          </tr>
          <tr>
            <td>Yesterday</td>
            <td>{{ lastDayKWH.toFixed(2) }}</td>
            <td>{{ formatCost(lastDayKWH) }}</td>
          </tr>
          <tr>
            <td>This Week</td>
            <td>{{ thisWeekKWH.toFixed(2) }}</td>
            <td>{{ formatCost(thisWeekKWH) }}</td>
          </tr>
          <tr>
            <td>Last Week</td>
            <td>{{ lastWeekKWH.toFixed(2) }}</td>
            <td>{{ formatCost(lastWeekKWH) }}</td>
          </tr>
          <tr>
            <td>This Month</td>
            <td>{{ thisMonthKWH.toFixed(2) }}</td>
            <td>{{ formatCost(thisMonthKWH) }}</td>
          </tr>
          <tr>
            <td>Last Month</td>
            <td>{{ lastMonthKWH.toFixed(2) }}</td>
            <td>{{ formatCost(lastMonthKWH) }}</td>
          </tr>
        </tbody>
      </table>
    </Modal>
  </div>
</template>

<style lang="scss" scoped>
  $card-bg: #101013;
  $track: #1f2027;
  $text: #f4f4f8;
  $muted: #8c93a6;
  $cost: #e3b341;

  .powerConsumptionDailyMain {
    height: 100%;

    @include mobile {
      display: none;
    }
  }

  // ── Card ──────────────────────────────────────────────────────────────
  // Same card style as the Power, Devices, Gate and PC cards.
  // Spacing on the card itself uses vh: cq units only work for its children.
  .card {
    display: flex;
    flex-direction: column;
    gap: 0.6vh;
    width: 100%;
    height: 100%;
    margin: 0;
    padding: 0.6vh 0.6vw;
    border: 0.08vw solid #25252b;
    border-radius: 0.8vw;
    background: $card-bg;
    color: $text;
    font-family: 'Seven Segment', sans-serif;
    text-align: left;
    container-type: size;
    box-sizing: border-box;
    cursor: pointer;
    // Scale the contents to fit both the width and the height left by the rows
    // above (about 94 units tall, minus the two 1.6vh gaps). Resolved in the
    // children, so cq units refer to this card.
    --unit: min(1cqw, calc((100cqh - 3.2vh) / 100));

    &:focus-visible {
      outline: 0.15vw solid #ffffff;
      outline-offset: 0.2vw;
    }
  }

  .label {
    color: $muted;
    font-size: calc(13.5 * var(--unit));
    letter-spacing: 0.12em;
    text-transform: uppercase;
  }

  small {
    margin-left: 0.2em;
    font-size: 0.45em;
  }

  // ── Today ─────────────────────────────────────────────────────────────
  .today {
    display: flex;
    align-items: flex-start;
    gap: calc(6 * var(--unit));
  }

  .icon {
    width: calc(45 * var(--unit));
    margin-top: calc(3 * var(--unit));
    opacity: 0.85;
  }

  .todayReading {
    display: flex;
    flex-direction: column;
    line-height: 1;
  }

  .todayKwh {
    font-size: calc(24 * var(--unit));
    letter-spacing: 0.05em;
    white-space: nowrap;

    small {
      color: $muted;
    }
  }

  .todayCost {
    color: $cost;
    font-size: calc(22 * var(--unit));
    white-space: nowrap;
  }

  // ── Week and month table ──────────────────────────────────────────────
  // Numbers are right-aligned in their columns so the digits line up.
  .periods {
    display: grid;
    grid-template-columns: auto 1fr auto;
    align-items: baseline;
    gap: calc(2.5 * var(--unit)) calc(8 * var(--unit));
    font-size: calc(18.75 * var(--unit));
  }

  .columnHead {
    color: $muted;
    font-size: calc(15.5 * var(--unit));
    text-align: right;
  }

  .kwh,
  .cost {
    text-align: right;
    white-space: nowrap;
  }

  .cost {
    color: $cost;
  }

  // ── Share of the home ─────────────────────────────────────────────────
  .share {
    display: flex;
    flex-direction: column;
    gap: calc(2 * var(--unit));
    margin-top: auto;
  }

  .shareTrack {
    height: calc(2 * var(--unit));
    overflow: visible;
    border-radius: var(--unit);
    background: $track;
  }

  // The glow uses the fill colour, set as its text colour.
  .shareFill {
    display: block;
    min-width: calc(10 * var(--unit));
    height: 100%;
    border-radius: inherit;
    box-shadow: 0 0 calc(1.5 * var(--unit)) currentColor;
    transition:
      width 0.8s ease,
      background 0.8s ease;
  }

  .shareCaption {
    display: flex;
    justify-content: space-between;
    color: $muted;
    font-size: calc(12.5 * var(--unit));
  }

  // ── Details pop-up ────────────────────────────────────────────────────
  .powerConsumptionDailyTable {
    width: 100%;
    border-collapse: collapse;
    font-size: 1rem;
    color: aliceblue;

    th,
    td {
      padding: 0.5rem 1rem;
      text-align: right;
      border-bottom: 1px solid rgba(255, 255, 255, 0.2);
    }

    th:first-child,
    td:first-child {
      text-align: left;
    }

    thead th {
      font-size: 0.85rem;
      opacity: 0.7;
      border-bottom: 2px solid rgba(255, 255, 255, 0.4);
    }
  }
</style>
