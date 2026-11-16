<script setup lang="ts">
  import { computed, provide, ref } from 'vue';

  import { openChartKey, openChartTimeSeriesKey } from '../composables/useOpenChart';
  import PowerConsumptionStats from '../components/power/PowerConsumptionStats.vue';
  import { TAG_CAR_CAPTUR_BATTERY, TAG_CAR_CAPTUR_LOCATION, TAG_CAR_CAPTUR_PLUG } from '../constants/tagIds';

  import Car from '../components/car/Car.vue';
  import Gate from '../components/gate/Gate.vue';
  import PcStats from '../components/pc/PcStats.vue';
  import PowerConsumption from '../components/power/PowerConsumption.vue';
  import Temperature from '../components/temperature/Temperature.vue';
  import TopBar from '../components/topbar/TopBar.vue';
  import Weather from '../components/weather/Weather.vue';
  import Modal from '../components/ui/Modal.vue';
  import LocationMap from '../components/car/LocationMap.vue';
  import { getTagValueFloat, getTagValueFloatTimeSeries } from '../services/api';

  interface ChartPoint {
    value: number;
    date: Date;
  }

  const axisLabel = {
    color: '#ffffff',
    formatter: (value: number) => new Date(value).toLocaleTimeString('pt-PT', { timeZone: 'Europe/Lisbon' }),
  };

  const showDialog = ref(false);
  const chartType = ref(0);
  const dialogTitle = ref<string | undefined>();
  const lineData = ref<ChartPoint[]>([]);
  const scatterDataAvg = ref<ChartPoint[]>([]);
  const scatterDataMax = ref<ChartPoint[]>([]);
  const scatterDataMin = ref<ChartPoint[]>([]);
  const chartLowerLimit = ref(0);
  const chartUpperLimit = ref(30);

  // The chart modal turns into an accordion (chart | map) when it is opened with a location.
  const locationTagId = ref<number>();
  const expanded = ref<'chart' | 'map'>('chart');

  // The day picked in the accordion (yyyy-mm-dd); empty shows the last 24 hours.
  const selectedDay = ref('');
  // Today as yyyy-mm-dd in local time (the Swedish locale formats dates that way).
  const today = () => new Date().toLocaleDateString('sv-SE');
  const chartTagId = ref('');
  const chartRange = ref<[string, string]>(['', '']);

  // Loads the line chart for the picked day (local midnight to midnight) or the last 24 hours.
  async function loadLine() {
    const to = selectedDay.value ? new Date(`${selectedDay.value}T00:00:00`) : new Date();
    const from = new Date(to);
    (selectedDay.value ? to : from).setDate(to.getDate() + (selectedDay.value ? 1 : -1));
    chartRange.value = [from.toISOString(), to.toISOString()];

    const json = await getTagValueFloat(chartTagId.value, ...chartRange.value);
    // The server answers "0" instead of an empty list.
    lineData.value = (Array.isArray(json) ? json : []).map((element) => ({ value: element.value, date: new Date(element.time) }));
  }


  const lineOption = computed(() => ({
    backgroundColor: 'transparent',
    grid: { left: 60, right: 20, top: 20, bottom: 30 },
    tooltip: {
      trigger: 'axis',
      position: [0, 0],
    },
    xAxis: {
      type: 'time',
      axisLabel,
      axisLine: { lineStyle: { color: '#888888' } },
      axisTick: { show: false },
    },
    yAxis: {
      type: 'value',
      min: chartLowerLimit.value,
      max: chartUpperLimit.value,
      axisLabel: {
        color: '#ffffff',
        formatter: (value: number) => Math.trunc(value).toString(),
      },
      splitLine: { lineStyle: { color: '#222222' } },
    },
    series: [
      {
        type: 'line',
        showSymbol: false,
        data: lineData.value.map((v: ChartPoint) => [v.date, v.value]),
        lineStyle: { color: '#4ade80' },
      },
    ],
  }));

  const scatterOption = computed(() => ({
    backgroundColor: 'transparent',
    grid: { left: 60, right: 20, top: 30, bottom: 30 },
    tooltip: {
      trigger: 'axis',
      position: [0, 0],
    },
    legend: { textStyle: { color: '#ffffff' } },
    xAxis: {
      type: 'time',
      axisLabel,
      axisLine: { lineStyle: { color: '#888888' } },
      axisTick: { show: false },
    },
    yAxis: {
      type: 'value',
      min: chartLowerLimit.value,
      max: chartUpperLimit.value,
      axisLabel: {
        color: '#ffffff',
        formatter: (value: number) => Math.trunc(value).toString(),
      },
      splitLine: { lineStyle: { color: '#222222' } },
    },
    series: [
      {
        name: 'Avg',
        type: 'scatter',
        data: scatterDataAvg.value.map((v: ChartPoint) => [v.date, v.value]),
        itemStyle: { color: '#60a5fa' },
      },
      {
        name: 'Min',
        type: 'scatter',
        data: scatterDataMin.value.map((v: ChartPoint) => [v.date, v.value]),
        itemStyle: { color: '#facc15' },
      },
      {
        name: 'Max',
        type: 'scatter',
        data: scatterDataMax.value.map((v: ChartPoint) => [v.date, v.value]),
        itemStyle: { color: '#f97316' },
      },
    ],
  }));

  async function openChart(tagId: string, title: string, lowerLimit = 10, upperLimit = 30, withLocationTagId?: number) {
    locationTagId.value = withLocationTagId;
    expanded.value = 'chart';
    selectedDay.value = today();
    chartTagId.value = tagId;

    chartLowerLimit.value = lowerLimit;
    chartUpperLimit.value = upperLimit;

    await loadLine();
    const tempValues = lineData.value;

    dialogTitle.value = title;
    if (tempValues.length > 0) {
      dialogTitle.value = `${title} - Last Update: ${tempValues[tempValues.length - 1].date.toLocaleString('pt-PT', { timeZone: 'Europe/Lisbon' })}`;
    }

    chartType.value = 0;
    showDialog.value = true;
  }

  async function openChartTimeSeries(tagId: string, title: string, lowerLimit = 10, upperLimit = 30) {
    const tempValuesAvg: ChartPoint[] = [];
    const tempValuesMax: ChartPoint[] = [];
    const tempValuesMin: ChartPoint[] = [];

    const now = new Date();
    const hoursPrevious24 = new Date();
    hoursPrevious24.setDate(hoursPrevious24.getDate() - 1);

    const json = await getTagValueFloatTimeSeries(tagId, hoursPrevious24.toISOString(), now.toISOString(), 5);
    json.forEach((element) => {
      tempValuesAvg.push({ value: element.avg_value, date: new Date(element.bucket) });
      tempValuesMax.push({ value: element.max_value, date: new Date(element.bucket) });
      tempValuesMin.push({ value: element.min_value, date: new Date(element.bucket) });
    });

    if (lowerLimit === 0 && upperLimit === 0) {
      const allValues = [...tempValuesAvg, ...tempValuesMax, ...tempValuesMin];
      const valuesOnly = allValues.map((v) => v.value);
      const calculatedLowerLimit = Math.min(...valuesOnly) * 0.9;
      const calculatedUpperLimit = Math.max(...valuesOnly) * 1.1;
      chartLowerLimit.value = calculatedLowerLimit;
      chartUpperLimit.value = calculatedUpperLimit;
    } else {
      chartLowerLimit.value = lowerLimit;
      chartUpperLimit.value = upperLimit;
    }

    scatterDataAvg.value = tempValuesAvg;
    scatterDataMax.value = tempValuesMax;
    scatterDataMin.value = tempValuesMin;

    dialogTitle.value = title;
    if (tempValuesAvg.length > 0) {
      dialogTitle.value = `${title} - Last Update: ${tempValuesAvg[tempValuesAvg.length - 1].date.toLocaleString('pt-PT', { timeZone: 'Europe/Lisbon' })}`;
    }

    chartType.value = 1;
    showDialog.value = true;
  }

  function closeDialog() {
    showDialog.value = false;
  }

  provide(openChartKey, openChart);
  provide(openChartTimeSeriesKey, openChartTimeSeries);
</script>

<template>
  <div class="mainContainer">
    <div class="topBarContainer">
      <TopBar />
    </div>
    <div class="weatherContainer">
      <Weather />
    </div>
    <div class="powerConsumptionContainer">
      <PowerConsumption />
    </div>
    <div class="temperatureContainer">
      <Temperature />
    </div>

    <div class="outsideContainer">
      <div class="outsideGateContainer">
        <Gate />
      </div>
      <div class="outsidePcContainer">
        <PcStats />
      </div>
      <div class="outsideCapturContainer">
        <Car label="Captur" :battery-tag-id="TAG_CAR_CAPTUR_BATTERY" :plug-tag-id="TAG_CAR_CAPTUR_PLUG" :location-tag-id="TAG_CAR_CAPTUR_LOCATION" />
      </div>
    </div>

    <div class="powerConsumptionStatsContainer">
      <PowerConsumptionStats />
    </div>

    <Modal :open="showDialog" :title="dialogTitle" @close="closeDialog">
      <div v-if="chartType === 0 && locationTagId !== undefined" class="accordion">
        <label class="dayPicker">
          Day
          <input v-model="selectedDay" type="date" @change="loadLine" />
          <span v-if="!selectedDay">(last 24h)</span>
        </label>
        <button type="button" :aria-expanded="expanded === 'chart'" @click="expanded = 'chart'">Battery</button>
        <VChart v-if="expanded === 'chart'" class="accordionPanel" :option="lineOption" autoresize />
        <button type="button" :aria-expanded="expanded === 'map'" @click="expanded = 'map'">Location</button>
        <!-- The key remounts the map when the period changes. -->
        <LocationMap
          v-if="expanded === 'map'"
          :key="chartRange[0]"
          class="accordionPanel"
          :tag-id="locationTagId"
          :from="chartRange[0]"
          :to="chartRange[1]"
          :show-latest="!selectedDay || selectedDay === today()"
          :battery="lineData"
        />
      </div>
      <div v-else style="width: 100%; height: 100%">
        <VChart v-if="chartType === 0":option="lineOption" autoresize style="width: 100%; height: 100%" />
        <VChart v-else :option="scatterOption" autoresize style="width: 100%; height: 100%" />
      </div>
    </Modal>
  </div>
</template>

<style lang="scss">
  body {
    background-color: #000000;
  }

  // On desktop the page is exactly one screen tall: the energy cards at the
  // bottom take whatever height the rows above leave. On mobile it scrolls.
  .mainContainer {
    display: flex;
    flex-direction: column;
    height: 100vh;

    @include mobile {
      height: auto;
    }
  }
  .topBarContainer {
    flex: 0 0 auto;
    margin-top: 0.5vh;
  }

  // Change the height here to resize the whole weather card.
  .weatherContainer {
    flex: 0 0 auto;
    height: 18vh;
    margin-top: 0.5vh;

    @include mobile {
      display: none;
    }
  }
  .powerConsumptionContainer {
    flex: 0 0 auto;
    margin-top: 0.5vh;
  }

  .temperatureContainer {
    flex: 0 0 auto;
    margin-top: 0.5vh;
  }

  // On mobile the gate and car share a row and the PC card gets its own row below.
  .outsideContainer {
    display: flex;
    margin-top: 0.5vh;
    height: 16vh;

    @include mobile {
      flex-wrap: wrap;
      row-gap: 1vh;
      margin-top: 0.5vh;
    }

    .outsideGateContainer {
      flex: 1;
      display: flex;
      justify-content: center;
    }

    .outsidePcContainer {
      flex: 1;
      display: flex;
      justify-content: center;

      @include mobile {
        order: 3;
        flex: 0 0 100%;
      }
    }

    .outsideCapturContainer {
      flex: 1;
      display: flex;
      justify-content: center;
    }
  }

  // The open panel takes the height the two headers leave.
  .accordion {
    display: flex;
    flex-direction: column;
    gap: 6px;
    height: 100%;

    .dayPicker {
      flex: 0 0 auto;
      display: flex;
      align-items: center;
      gap: 8px;

      input {
        padding: 4px 8px;
        border: 1px solid #444444;
        border-radius: 4px;
        background: #1a1a1a;
        color: #ffffff;
        font: inherit;
        color-scheme: dark;
      }
    }

    > button {
      flex: 0 0 auto;
      padding: 8px 12px;
      border: 1px solid #444444;
      border-radius: 4px;
      background: #1a1a1a;
      color: #ffffff;
      font: inherit;
      text-align: left;
      cursor: pointer;

      &::before {
        content: '+ ';
      }

      &[aria-expanded='true'] {
        background: #2a2a2a;

        &::before {
          content: '- ';
        }
      }
    }

    .accordionPanel {
      flex: 1 1 0;
      min-height: 0;
      width: 100%;
      margin: 0;
      border: 0;
    }
  }

  .powerConsumptionStatsContainer {
    flex: 1 1 0;
    min-height: 0;
    margin-top: 0.5vh;
    margin-bottom: 0.5vh;
    @include mobile {
      display: none;
    }
  }
</style>
