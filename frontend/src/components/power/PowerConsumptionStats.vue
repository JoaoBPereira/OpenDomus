<script setup lang="ts">
  import { onMounted, ref } from 'vue';
  import { getTags, Tag } from '../../services/api';
  import { TAG_ENERGY_TOTAL } from '../../constants/tagIds';
  import PowerConsumptionDaily from './PowerConsumptionDaily.vue';
  import { getKwh, startOfMonth } from './powerEnergy';

  const config = Config;

  const dataPowerTags = ref<Tag[]>([]);
  const homeMonthKwh = ref<number>();

  // Tag configs arrive as JSON strings.
  function parseConfig(tag: Tag) {
    if (typeof tag.config !== 'string') return tag;
    try {
      tag.config = JSON.parse(tag.config);
    } catch (error) {
      console.error('Failed to parse tag config for tag:', tag, error);
    }
    return tag;
  }

  const fetchInitialData = async () => {
    const tags = await getTags();
    const filteredTags = tags
      .filter((tag) => tag.displayIn.includes('PowerConsumptionStats'))
      .map(parseConfig)
      .sort((a, b) => a.id - b.id);
    dataPowerTags.value = filteredTags;

    const homeTag = tags.find((tag) => tag.id === TAG_ENERGY_TOTAL);
    const scale = (homeTag?.config as any)?.Emeter?.scale;
    if (scale) homeMonthKwh.value = await getKwh(TAG_ENERGY_TOTAL, scale, startOfMonth());
  };

  onMounted(() => {
    fetchInitialData();
  });
</script>

<template>
  <div style="display: flex; width: 98vw; height: 100%; gap: 1vw; margin-left: 1vw; margin-right: 1vw">
    <div style="flex: 1; min-width: 0" v-for="tag in dataPowerTags" :key="tag.id">
      <PowerConsumptionDaily :name="tag.name" :tag-id="tag.id" :price-kwh="config.priceKwh" :scale="tag.config.Emeter.scale" :home-month-kwh="tag.id === TAG_ENERGY_TOTAL ? undefined : homeMonthKwh" />
    </div>
  </div>
</template>

<style lang="scss" scoped></style>
