<script setup lang="ts">
  import { computed, inject, onBeforeUnmount, onMounted, ref } from 'vue';

  import { getLatestFloat, type FloatTimeValue } from '../../utils';
  import { getTagValueFloat } from '../../services/api';
  import Modal from '../ui/Modal.vue';
  import { openChartKey } from '../../composables/useOpenChart';
  import { formatDelta, loadTemperature, temperatureTone, type TemperatureReading } from '../temperature/temperatureData';
  import { TAG_BOILER_DELTA, TAG_BOILER_EXHAUST, TAG_BOILER_HIGH_LIMIT, TAG_BOILER_IMPULSION, TAG_BOILER_RETURN, TAG_BOILER_STATE, TAG_BOILER_WATER_TEMP } from '../../constants/tagIds';

  const openChart = inject(openChartKey)!;

  const EMPTY = '\u2014';
  const SCALE_MIN = 20;
  const SCALE_MAX = 70;

  const tempWater = ref<TemperatureReading>();
  const state = ref<FloatTimeValue>();
  const highLimitTemperature = ref<FloatTimeValue>();
  const deltaTemperature = ref<FloatTimeValue>();
  const impulsionTemperature = ref<FloatTimeValue>();
  const returnTemperature = ref<FloatTimeValue>();
  const exhaustTemperature = ref<FloatTimeValue>();

  let intervalId: number | undefined;

  const fetchData = async () => {
    tempWater.value = await loadTemperature(TAG_BOILER_WATER_TEMP, Date.now());
    state.value = await getLatestFloat(TAG_BOILER_STATE);
    deltaTemperature.value = await getLatestFloat(TAG_BOILER_DELTA);
    highLimitTemperature.value = await getLatestFloat(TAG_BOILER_HIGH_LIMIT);
    impulsionTemperature.value = await getLatestFloat(TAG_BOILER_IMPULSION);
    returnTemperature.value = await getLatestFloat(TAG_BOILER_RETURN);
    exhaustTemperature.value = await getLatestFloat(TAG_BOILER_EXHAUST);
  };

  onMounted(() => {
    fetchData();
    intervalId = window.setInterval(fetchData, 60000);
  });

  onBeforeUnmount(() => {
    if (intervalId) {
      clearInterval(intervalId);
    }
  });

  const water = computed(() => (tempWater.value?.error ? undefined : tempWater.value?.value));

  const waterDelta = computed(() => formatDelta(tempWater.value?.delta));

  const waterTrend = computed(() => {
    const change = tempWater.value?.delta ?? 0;
    return { warming: change > 0, cooling: change < 0 };
  });

  const stop = computed(() => highLimitTemperature.value?.value);

  const start = computed(() => {
    if (stop.value === undefined || deltaTemperature.value === undefined) return undefined;
    return stop.value - deltaTemperature.value.value;
  });

  // Boiler state code → dot class (coloured in the styles) and pill label.
  function describeState(code?: number) {
    switch (code) {
      case 0:
      case 1:
        return { status: 'off', label: 'Off/Inactive' };
      case 2:
      case 3:
      case 4:
      case 10:
        return { status: 'igniting', label: 'Igniting/Starting up' };
      case 5:
      case 6:
        return { status: 'preparing', label: 'Auto Tuning/Preparing' };
      case 7:
        return { status: 'running', label: 'Running/On' };
      case 8:
      case -2:
      case 11:
      case 9:
        return { status: 'error', label: 'Shutting down/Error' };
      case -3:
      case -4:
        return { status: 'alarm', label: 'Alarm/Warning' };
      case 20:
        return { status: 'standby', label: 'Standby' };
      default:
        return { status: 'unknown', label: 'Unknown' };
    }
  }

  const boilerState = computed(() => describeState(state.value?.value));

  // ── State history dialog ──────────────────────────────────────────────
  const TZ = 'Europe/Lisbon';
  const dayKey = (d: Date) => d.toLocaleDateString('en-CA', { timeZone: TZ });
  const formatTime = (d: Date) => d.toLocaleTimeString('en-US', { timeZone: TZ, hour: 'numeric', minute: '2-digit', second: '2-digit' });

  const historyOpen = ref(false);
  const historyTitle = ref('Boiler State');
  const historyRange = ref(24);
  const historyData = ref<Array<{ value: number; date: Date }>>([]);

  const fetchHistory = async () => {
    const now = new Date();
    const from = new Date();
    from.setHours(from.getHours() - historyRange.value);

    const json = await getTagValueFloat(TAG_BOILER_STATE, from.toISOString(), now.toISOString());
    const values = json.map((element) => ({ value: element.value, date: new Date(element.time) })).sort((a, b) => a.date.getTime() - b.date.getTime());
    // Keep only the readings where the state actually changed.
    historyData.value = values.filter((item, index) => index === 0 || item.value !== values[index - 1].value);

    historyTitle.value = values.length > 0 ? `Boiler State - Last Update: ${values[values.length - 1].date.toLocaleString('pt-PT', { timeZone: TZ })}` : 'Boiler State';
  };

  const openHistory = () => {
    historyOpen.value = true;
    fetchHistory();
  };

  const historyGroups = computed(() => {
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    const todayKey = dayKey(new Date());
    const yesterdayKey = dayKey(yesterday);

    const groups: Array<{ key: string; label: string; items: Array<{ value: number; date: Date; status: string; label: string }> }> = [];
    [...historyData.value]
      .sort((a, b) => b.date.getTime() - a.date.getTime())
      .forEach((item) => {
        const key = dayKey(item.date);
        let group = groups[groups.length - 1];
        if (!group || group.key !== key) {
          const dateLabel = item.date.toLocaleDateString('en-US', { timeZone: TZ, month: 'long', day: 'numeric', year: 'numeric' });
          const prefix = key === todayKey ? 'Today' : key === yesterdayKey ? 'Yesterday' : item.date.toLocaleDateString('en-US', { timeZone: TZ, weekday: 'long' });
          group = { key, label: `${prefix} · ${dateLabel}`, items: [] };
          groups.push(group);
        }
        group.items.push({ ...item, ...describeState(item.value) });
      });
    return groups;
  });

  const flowDelta = computed(() => {
    if (impulsionTemperature.value === undefined || returnTemperature.value === undefined) return EMPTY;
    return formatDelta(Math.round((impulsionTemperature.value.value - returnTemperature.value.value) * 10) / 10);
  });

  const statusText = computed(() => {
    return `state ${state.value?.value}`;
  });

  const tankStyle = computed(() => (water.value === undefined ? {} : { backgroundColor: temperatureTone(water.value, SCALE_MIN, SCALE_MAX) }));

  // Position on the 20°–70° scale, as a percentage from the bottom.
  function scalePercent(value: number) {
    const percent = ((value - SCALE_MIN) / (SCALE_MAX - SCALE_MIN)) * 100;
    return Math.min(Math.max(percent, 0), 100);
  }

  const bandStyle = computed(() => {
    if (start.value === undefined || stop.value === undefined) return undefined;
    const bottom = scalePercent(start.value);
    return { bottom: `${bottom}%`, height: `${scalePercent(stop.value) - bottom}%` };
  });

  const markerStyle = computed(() => (water.value === undefined ? undefined : { bottom: `${scalePercent(water.value)}%` }));
  const stopStyle = computed(() => (stop.value === undefined ? undefined : { bottom: `${scalePercent(stop.value)}%` }));
  const startStyle = computed(() => (start.value === undefined ? undefined : { top: `${100 - scalePercent(start.value)}%` }));

  function formatTemperature(value?: number) {
    return value === undefined ? EMPTY : `${value.toFixed(1)}\u00b0`;
  }

  function formatLimit(value?: number) {
    return value === undefined ? EMPTY : `${Math.round(value)}\u00b0`;
  }
</script>

<template>
  <div class="boiler">
    <div class="exhaust">
      <span class="chimney" />
      <button type="button" class="reading" @click="openChart(TAG_BOILER_EXHAUST, 'Boiler Exhaust', 0, 300)">
        <span class="label">Exhaust</span>
        <span class="value">{{ formatTemperature(exhaustTemperature?.value) }}</span>
      </button>
    </div>

    <div class="scale">
      <span class="scaleEnd scaleMax">{{ SCALE_MAX }}&deg;</span>

      <div class="track">
        <span v-if="bandStyle" class="band" :style="bandStyle" />
        <span v-if="markerStyle" class="marker" :style="markerStyle" />

        <button v-if="stopStyle" type="button" class="limit stop" :style="stopStyle" @click="openChart(TAG_BOILER_HIGH_LIMIT, 'Boiler High Limit', 40, 80)">
          <span class="label">Stop</span>
          <span class="value">{{ formatLimit(stop) }}</span>
        </button>
        <span v-if="startStyle" class="limit start" :style="startStyle">
          <span class="label">Start</span>
          <span class="value">{{ formatLimit(start) }}</span>
        </span>
      </div>

      <span class="scaleEnd scaleMin">{{ SCALE_MIN }}&deg;</span>
    </div>

    <div class="tank" :style="tankStyle">
      <svg class="flame" viewBox="0 0 24 24" aria-hidden="true">
        <path d="M12 2c1 4-3 5.5-3 9.5a3 3 0 0 0 6 0c0-1.5-.6-2.6-1-3.2 2.2 1.2 4 3.6 4 6.7a6 6 0 0 1-12 0c0-5.3 6-7.5 6-13z" />
      </svg>

      <button type="button" class="waterReading" @click="openChart(TAG_BOILER_WATER_TEMP, 'Main Water', 20, 80)">
        <span class="label">Water</span>
        <span class="waterValue">{{ water === undefined ? EMPTY : water.toFixed(1) }}<sup v-if="water !== undefined">&deg;</sup></span>
        <span class="waterDelta" :class="waterTrend">{{ waterDelta }}</span>
      </button>

      <button type="button" class="statePill" @click="openHistory">
        <span class="stateDot" :class="boilerState.status" />
        {{ boilerState.label }}
      </button>

      <span class="status">{{ `state ${state?.value?.valueOf()}` }}</span>
    </div>

    <div class="pipes">
      <button type="button" class="pipe impulsion" @click="openChart(TAG_BOILER_IMPULSION, 'Boiler Impulsion', 20, 80)">
        <span class="label">Impulsion &rarr;</span>
        <span class="value">{{ formatTemperature(impulsionTemperature?.value) }}</span>
        <span class="line" />
      </button>

      <div class="flowDelta">
        <span class="label">&Delta;T flow&minus;return</span>
        <span class="value">{{ flowDelta }}</span>
      </div>

      <button type="button" class="pipe return" @click="openChart(TAG_BOILER_RETURN, 'Boiler Return', 20, 80)">
        <span class="line" />
        <span class="label">&larr; Return</span>
        <span class="value">{{ formatTemperature(returnTemperature?.value) }}</span>
      </button>
    </div>

    <!-- Teleported: the card's size containment would otherwise trap the fixed overlay. -->
    <Teleport to="body">
      <Modal :open="historyOpen" :title="historyTitle" @close="historyOpen = false">
        <div class="historyBody">
          <div style="padding-left: 1vw">
            <label class="historySelectLabel">
              Time Range
              <select v-model.number="historyRange" class="historySelect" @change="fetchHistory">
                <option :value="24">24 Hours</option>
                <option :value="48">48 Hours</option>
                <option :value="168">7 Days</option>
                <option :value="720">1 Month</option>
              </select>
            </label>
          </div>
          <div class="timeline">
            <div v-for="group in historyGroups" :key="group.key" class="timeline-group">
              <div class="timeline-day">{{ group.label }}</div>
              <div v-for="item in group.items" :key="item.date.toISOString()" class="timeline-item">
                <div class="timeline-time">{{ formatTime(item.date) }}</div>
                <div class="timeline-marker">
                  <div class="timeline-icon"><span class="stateDot" :class="item.status" /></div>
                </div>
                <div class="timeline-content">
                  <div class="timeline-title">Boiler &rarr; {{ item.label }}</div>
                  <div class="timeline-subtitle">State {{ item.value }}</div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </Modal>
    </Teleport>
  </div>
</template>

<style lang="scss" scoped>
  $card-bg: #101013;
  $text: #f4f4f8;
  $muted: #a8b2c6;
  $warm: #e2ac69;
  $cool: #8fa6f0;
  $light-green: #8ee6a4;
  $green: #3fbf6a;
  $red: #e74c3c;
  $orange: #f39c12;
  $pipe-warm: #b98a4e;
  $pipe-cool: #5b6fa8;
  $blue: #5b8def;
  $outline: #c9c9cf;

  // ── Card ──────────────────────────────────────────────────────────────
  .boiler {
    display: grid;
    grid-template-columns: 33% 33% 0fr;
    grid-template-rows: 18% 1fr;
    width: 100%;
    height: 100%;
    padding: 1.2vh 1.2vw;
    border: 0.08vw solid #25252b;
    border-radius: 0.8vw;
    background: $card-bg;
    color: $text;
    font-family: 'Seven Segment', sans-serif;
    container-type: size;
    box-sizing: border-box;

    @include mobile {
      padding: 1.2vh 2vw;
      border-width: 0.1vh;
      border-radius: 1vh;
    }
  }

  button {
    margin: 0;
    padding: 0;
    border: 0;
    background: none;
    color: inherit;
    font: inherit;
    text-align: inherit;
    cursor: pointer;

    &:focus-visible {
      outline: 0.15vw solid #ffffff;
      outline-offset: 0.2vw;

      @include mobile {
        outline-width: 0.25vh;
      }
    }
  }

  .label {
    font-size: 4.4cqmin;
    letter-spacing: 0.2em;
    text-transform: uppercase;
  }

  .value {
    font-size: 7.5cqmin;
    letter-spacing: 0.05em;
    white-space: nowrap;
  }

  // ── Exhaust ───────────────────────────────────────────────────────────
  .exhaust {
    grid-area: 1 / 2 / 2 / 4;
    display: flex;
    gap: 1vw;
    padding-left: 12%;

    .chimney {
      width: 9%;
      border: 0.08vw solid #3a3a42;
      border-bottom: 0;
      background: #1c1c21;
    }

    .reading {
      display: flex;
      flex-direction: column;
      color: $muted;
    }
  }

  // ── Scale ─────────────────────────────────────────────────────────────
  .scale {
    grid-area: 2 / 1 / 3 / 2;
    position: relative;
  }

  .scaleEnd {
    position: absolute;
    right: 24%;
    color: $muted;
    font-size: 4cqmin;
  }

  .scaleMax {
    top: 0;
  }

  .scaleMin {
    bottom: 0;
  }

  .track {
    position: absolute;
    top: 8%;
    bottom: 8%;
    right: 10%;
    width: 0.6vw;
    border-radius: 1vw;
    background: linear-gradient(#5c4630, #2a3150);

    @include mobile {
      width: 1.5vw;
    }
  }

  // Dashed guides run from the band towards the tank.
  .band {
    position: absolute;
    left: -50%;
    right: -50%;
    border: 0.08vw solid $warm;
    border-radius: 0.2vw;
    background: rgba(226, 172, 105, 0.35);

    &::after {
      content: '';
      position: absolute;
      top: -0.08vw;
      bottom: -0.08vw;
      left: 100%;
      width: 180%;
      border-top: 0.08vw dashed rgba(226, 172, 105, 0.6);
      border-bottom: 0.08vw dashed rgba(226, 172, 105, 0.6);
    }
  }

  .marker {
    position: absolute;
    left: 50%;
    width: 4.5cqmin;
    height: 4.5cqmin;
    border: 0.15vw solid #101013;
    border-radius: 50%;
    background: #9bb1ff;
    transform: translate(-50%, 50%);
  }

  .limit {
    position: absolute;
    right: calc(100% + 1vw);
    display: flex;
    align-items: baseline;
    gap: 0.4vw;
    white-space: nowrap;

    .label {
      letter-spacing: 0.1em;
    }

    .value {
      color: $warm;
    }
  }

  // ── Tank ──────────────────────────────────────────────────────────────
  .tank {
    grid-area: 2 / 2 / 3 / 3;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 1.5vh 0.5vw;
    border: 0.12vw solid $outline;
    border-radius: 0.8vw;
    background: #2a2f45;
    text-align: center;

    @include mobile {
      border-width: 0.2vh;
      border-radius: 1vh;
    }
  }

  .flame {
    width: 6cqmin;
    height: 6cqmin;
    fill: none;
    stroke: $muted;
    stroke-width: 1.6;
  }

  .waterReading {
    display: flex;
    flex: 1;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 1vh;
    text-align: center;
  }

  // Also limited by width, since the tank column narrows in the one-row layout.
  .waterValue {
    font-size: min(19cqmin, 8cqw);
    line-height: 1;
    letter-spacing: 0.05em;
    white-space: nowrap;

    sup {
      vertical-align: top;
      font-size: 0.4em;
    }
  }

  .waterDelta {
    color: $muted;
    font-size: 5.5cqmin;
    white-space: nowrap;

    &.warming {
      color: $warm;
    }
    &.cooling {
      color: $cool;
    }
  }

  .statePill {
    display: flex;
    align-items: center;
    gap: 0.6vw;
    padding: 0.5vh 1vw;
    border: 0.1vw solid #5a5f73;
    border-radius: 2vw;
    background: #15161c;
    font-size: 4.6cqmin;
    letter-spacing: 0.15em;
    text-transform: uppercase;

    @include mobile {
      gap: 1.5vw;
      padding: 0.5vh 2.5vw;
      border-width: 0.15vh;
      border-radius: 3vh;
    }
  }

  .stateDot {
    flex-shrink: 0;
    width: 4cqmin;
    height: 4cqmin;
    border-radius: 50%;
    background: $muted;
  }

  .stateDot.off {
    background: $muted;
    box-shadow: 0 0 1vw $muted;
  }

  .stateDot.igniting {
    background: $warm;
    box-shadow: 0 0 1vw $warm;
  }

  .stateDot.preparing {
    background: $light-green;
    box-shadow: 0 0 1vw $light-green;
  }

  .stateDot.running {
    background: $green;
    box-shadow: 0 0 1vw $green;
  }

  .stateDot.error {
    background: $red;
    box-shadow: 0 0 1vw $red;
  }

  .stateDot.alarm {
    background: $orange;
    box-shadow: 0 0 1vw $orange;
  }

  .stateDot.standby {
    background: $blue;
    box-shadow: 0 0 1vw $blue;
  }

  .stateDot.unknown {
    background: $pipe-cool;
    box-shadow: 0 0 1vw $pipe-cool;
  }

  .status {
    margin-top: 1vh;
    color: $muted;
    font-size: 4.2cqmin;
    white-space: nowrap;
  }

  // ── Pipes ─────────────────────────────────────────────────────────────
  .pipes {
    grid-area: 2 / 3 / 2 / 4;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    padding: 4% 0 6%;
  }

  .pipe {
    display: flex;
    flex-direction: column;
    gap: 0.6vh;

    .label,
    .value {
      padding-left: 1.5vw;
    }

    .line {
      height: 1.2cqmin;
      border-radius: 0 1vw 1vw 0;
    }
  }

  .impulsion {
    color: $warm;

    .line {
      background: $pipe-warm;
    }
  }

  .return {
    color: $cool;

    .line {
      background: $pipe-cool;
    }
  }

  .flowDelta {
    display: flex;
    flex-direction: column;
    align-self: flex-end;
    margin-left: 1vw;
    color: $muted;

    .value {
      color: $text;
    }
  }

  // ── State history dialog ──────────────────────────────────────────────
  .historySelectLabel {
    display: inline-flex;
    flex-direction: column;
    gap: 6px;
    color: #ffffff;
    font-size: 0.85rem;
  }

  .historySelect {
    background: #111111;
    color: #ffffff;
    border: 1px solid #444444;
    border-radius: 4px;
    padding: 6px 8px;
  }

  .historyBody {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .timeline {
    flex: 1;
    min-height: 0;
    padding: 12px 1vw;
    color: #ffffff;
    overflow-y: auto;
  }

  .timeline-group {
    margin-bottom: 16px;
  }

  .timeline-day {
    font-size: 0.95rem;
    color: #dddddd;
    margin-bottom: 12px;
  }

  .timeline-item {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    position: relative;
    min-height: 56px;
  }

  .timeline-time {
    width: 90px;
    text-align: right;
    font-size: 0.8rem;
    color: #aaaaaa;
    padding-top: 10px;
  }

  .timeline-marker {
    position: relative;
    display: flex;
    justify-content: center;
    width: 36px;
    align-self: stretch;
  }

  .timeline-item:not(:last-child) .timeline-marker::after {
    content: '';
    position: absolute;
    top: 36px;
    bottom: 0;
    left: 50%;
    width: 1px;
    background: #444444;
  }

  .timeline-icon {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #2a2a30;

    .stateDot {
      width: 14px;
      height: 14px;
      box-shadow: none;
    }
  }

  .timeline-content {
    padding-top: 4px;
  }

  .timeline-title {
    font-size: 0.95rem;
  }

  .timeline-subtitle {
    font-size: 0.75rem;
    color: #aaaaaa;
    margin-top: 2px;
  }
</style>
