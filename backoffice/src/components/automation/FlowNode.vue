<script setup lang="ts">
import { computed, inject } from 'vue';
import { Handle, Position, type NodeProps } from '@vue-flow/core';
import type { AutomationNode } from '@/components/models';
import { nodeInfo, summary } from './graph';

const props = defineProps<NodeProps<AutomationNode>>();
const names = inject<{ tags: Map<number, string>; devices: Map<number, string> }>(
  'automationNames',
);

const info = computed(() => nodeInfo(props.data.type));
const text = computed(() => summary(props.data, names));
</script>

<template>
  <div class="flow-node" :class="[info.category, { selected: props.selected }]">
    <Handle v-if="info.category !== 'trigger'" type="target" :position="Position.Left" />
    <div class="row items-center no-wrap">
      <q-icon :name="info.icon" size="18px" class="q-mr-xs" />
      <span class="text-weight-medium">{{ info.label }}</span>
    </div>
    <div v-if="text" class="text-caption ellipsis">{{ text }}</div>
    <Handle v-if="info.category !== 'action'" type="source" :position="Position.Right" />
  </div>
</template>

<style scoped>
.flow-node {
  min-width: 140px;
  max-width: 220px;
  padding: 8px 12px;
  border-radius: 6px;
  border: 1px solid #9e9e9e;
  background: white;
}
.flow-node.trigger {
  border-left: 4px solid #1976d2;
}
.flow-node.gate {
  border-left: 4px solid #9c27b0;
}
.flow-node.action {
  border-left: 4px solid #21ba45;
}
.flow-node.selected {
  box-shadow: 0 0 0 2px #1976d2;
}
</style>
