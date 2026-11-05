<script setup lang="ts">
import { computed } from 'vue';

export interface EssentialLinkProps {
  title: string;
  caption?: string;
  link?: string;
  icon?: string;
}

const props = withDefaults(defineProps<EssentialLinkProps>(), {
  caption: '',
  link: '#',
  icon: '',
});

const { title, caption, link, icon } = props;

const isExternal = computed(() => {
  const l = props.link ?? '';
  return l.startsWith('http') || l.startsWith('mailto:') || l.startsWith('tel:');
});
</script>

<template>
  <q-item
    clickable
    :tag="isExternal ? 'a' : 'router-link'"
    :href="isExternal ? link : undefined"
    :to="!isExternal ? link : undefined"
    :target="isExternal ? '_blank' : undefined"
  >
    <q-item-section v-if="icon" avatar>
      <q-icon :name="icon" />
    </q-item-section>

    <q-item-section>
      <q-item-label>{{ title }}</q-item-label>
      <q-item-label caption>{{ caption }}</q-item-label>
    </q-item-section>
  </q-item>
</template>
