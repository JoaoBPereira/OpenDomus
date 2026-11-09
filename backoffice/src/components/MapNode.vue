<!-- eslint-disable @typescript-eslint/no-explicit-any -->
<script setup lang="ts">
import { Handle, Position, type NodeProps } from '@vue-flow/core';

// data: { kind: 'root' | 'device' | 'tag', item?: device or tag from the API }
const props = defineProps<NodeProps<{ kind: 'root' | 'device' | 'tag'; item?: any }>>();
</script>

<template>
  <div v-if="props.data.kind === 'root'" class="map-node root">
    <div class="row items-center no-wrap">
      <q-icon name="hub" size="18px" class="q-mr-xs" />
      <span class="text-weight-medium">Devices</span>
    </div>
    <Handle type="source" :position="Position.Bottom" />
  </div>
  <div
    v-else
    class="map-node"
    :class="[props.data.kind, props.data.item.enabled ? 'enabled' : 'disabled']"
  >
    <Handle type="target" :position="Position.Top" />
    <div class="row items-center no-wrap">
      <q-icon
        :name="props.data.kind === 'device' ? 'devices' : 'sell'"
        size="18px"
        class="q-mr-xs"
      />
      <span class="text-weight-medium ellipsis">{{ props.data.item.name }}</span>
    </div>
    <template v-if="props.data.kind === 'device'">
      <div class="text-caption">
        Id: {{ props.data.item.id }} · {{ props.data.item.type || 'N/A' }}
      </div>
      <div class="text-caption ellipsis">URL: {{ props.data.item.url || 'N/A' }}</div>
      <div class="text-caption">Period: {{ props.data.item.periodicity || 'N/A' }}</div>
    </template>
    <template v-else>
      <div class="text-caption">
        Id: {{ props.data.item.id }} · {{ props.data.item.type || 'N/A' }}
      </div>
      <div class="text-caption">
        {{ props.data.item.valueType || 'N/A' }} · {{ props.data.item.valueStorage || 'N/A' }}
      </div>
      <div class="text-caption">
        Expires: {{ props.data.item.expiresIn ? props.data.item.expiresIn + 'h' : 'N/A' }}
      </div>
    </template>
    <div class="text-caption" :class="props.data.item.enabled ? 'text-positive' : 'text-negative'">
      {{ props.data.item.enabled ? '✓ Enabled' : '✗ Disabled' }}
    </div>
    <Handle v-if="props.data.kind === 'device'" type="source" :position="Position.Bottom" />
  </div>
</template>

<!-- Same look as the automation editor's FlowNode -->
<style scoped>
.map-node {
  width: 220px;
  padding: 8px 12px;
  border-radius: 6px;
  border: 1px solid #9e9e9e;
  background: white;
  cursor: pointer;
}
.map-node.root {
  width: auto;
  border-left: 4px solid #1976d2;
  cursor: default;
}
.map-node.enabled {
  border-left: 4px solid #21ba45;
}
.map-node.disabled {
  border-left: 4px solid #c10015;
}
</style>
