<script setup lang="ts">
  import Power1H from './Power1H.vue';
  import PowerBarChartKHH from './PowerBarChartKHH.vue';
</script>

<template>
  <div class="powerConsumptionMain">
    <div class="power1H">
      <Power1H />
    </div>
    <div class="powerConsumptionDevices">
      <PowerBarChartKHH />
    </div>
  </div>
</template>

<style lang="scss" scoped>
  .powerConsumptionMain {
    display: flex;
    height: 23.5vh;
    @include mobile {
      display: block;
      height: 100%;
    }
  }

  .power1H {
    flex: 1;
    width: 50vw;

    @include mobile {
      flex: initial;
      width: 100vw;
    }
  }

  .powerConsumptionDevices {
    flex: 1;
    width: 50vw;

    @include mobile {
      flex: initial;
      margin-top: 0.5vh;
      width: 100vw;
      height: 20.5vh;
    }
  }
</style>
