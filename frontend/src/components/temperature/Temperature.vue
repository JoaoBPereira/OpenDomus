<script setup lang="ts">
  import { onBeforeUnmount, onMounted, ref } from 'vue';

  import Boiler from '../boiler/Boiler.vue';
  import TemperatureFloor from './TemperatureFloor.vue';
  import TemperatureHotWater from './TemperatureHotWater.vue';
  import TemperatureOutside from './TemperatureOutside.vue';
  import { firstRooms, groundRooms, loadTemperature, type TemperatureReadings } from './temperatureData';
  import { TAG_TEMP_HOT_WATER, TAG_TEMP_OUTSIDE } from '../../constants/tagIds';

  const readings = ref<TemperatureReadings>({});
  const allRooms = [...groundRooms, ...firstRooms];
  const roomSensorIds = allRooms.flatMap((room) => (room.tagId ? [room.tagId] : []));
  const sensorIds = [...new Set([...roomSensorIds, TAG_TEMP_OUTSIDE, TAG_TEMP_HOT_WATER])];

  let intervalId: number | undefined;
  let fetching = false;
  let disposed = false;

  const fetchData = async () => {
    if (fetching) return;
    fetching = true;
    try {
      const now = Date.now();
      await Promise.all(
        sensorIds.map(async (tagId) => {
          const reading = await loadTemperature(tagId, now);
          if (!disposed) readings.value[tagId] = reading;
        }),
      );
    } finally {
      fetching = false;
    }
  };

  onMounted(() => {
    fetchData();
    intervalId = window.setInterval(fetchData, 60000);
  });

  onBeforeUnmount(() => {
    disposed = true;
    if (intervalId) {
      clearInterval(intervalId);
    }
  });
</script>

<template>
  <div class="temperatureMain">
    <div class="temperatureFloor">
      <TemperatureFloor :floor="0" :rooms="groundRooms" :readings="readings" />
    </div>
    <div class="temperatureFloor">
      <TemperatureFloor :floor="1" :rooms="firstRooms" :readings="readings" />
    </div>
    <div class="temperatureOutside">
      <TemperatureOutside :reading="readings[TAG_TEMP_OUTSIDE]" />
    </div>
    <div class="temperatureBoiler">
      <Boiler />
    </div>
    <div class="temperatureHotWater">
      <TemperatureHotWater :reading="readings[TAG_TEMP_HOT_WATER]" />
    </div>
  </div>
</template>

<style lang="scss" scoped>
  // Desktop: one row. Mobile: floors stacked, then Outside and hot water, then the boiler.
  .temperatureMain {
    display: flex;
    gap: 0.5vw;
    height: 23.5vh;
    margin-left: 0.5vw;
    width: 99vw;
    -webkit-user-select: none;
    -moz-user-select: none;
    -ms-user-select: none;
    user-select: none;

    @include mobile {
      flex-wrap: wrap;
      gap: 0.5vh 0.5vw;
      height: auto;
      margin-left: 2vw;
      width: 96vw;
    }
  }

  .temperatureFloor {
    flex: 3;
    min-width: 0;

    @include mobile {
      flex: 0 0 100%;
    }
  }

  .temperatureOutside {
    flex: 1;
    min-width: 0;
  }

  .temperatureBoiler {
    flex: 4;
    min-width: 0;

    @include mobile {
      order: 3;
      flex: 0 0 100%;
      height: 30vh;
    }
  }

  .temperatureHotWater {
    flex: 1;
    display: flex;
    justify-content: center;
    align-items: center;
    min-width: 0;
  }
</style>
