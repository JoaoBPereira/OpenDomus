<!-- eslint-disable @typescript-eslint/no-explicit-any -->
<script setup lang="ts">
import { ref, computed } from 'vue';
import { Notify } from 'quasar';
import { useRouter } from 'vue-router';
import { restartServer as svcRestartServer, getDevices, getTags } from '../../services';
import { VueFlow, type Edge, type Node, type NodeMouseEvent } from '@vue-flow/core';
import { Background } from '@vue-flow/background';
import { Controls } from '@vue-flow/controls';
import '@vue-flow/core/dist/style.css';
import '@vue-flow/core/dist/theme-default.css';
import '@vue-flow/controls/dist/style.css';
import MapNode from '@/components/MapNode.vue';

const restartLoading = ref(false);
const mapLoading = ref(false);
const devices = ref<any[]>([]);
const tags = ref<any[]>([]);
const router = useRouter();

function handleNodeClick({ node }: NodeMouseEvent) {
  const { kind, item } = node.data;
  if (kind === 'device') {
    void router.push({ path: '/devices', query: { editDeviceId: String(item.id) } });
  } else if (kind === 'tag') {
    void router.push({ path: '/tags', query: { editTagId: String(item.id) } });
  }
}

async function restartServer() {
  restartLoading.value = true;
  try {
    await svcRestartServer();
    Notify.create({ type: 'positive', message: 'Server restart requested successfully' });
  } catch (err: any) {
    Notify.create({
      type: 'negative',
      message: 'Failed to restart server: ' + (err?.message || String(err)),
    });
  } finally {
    restartLoading.value = false;
  }
}

async function loadMap() {
  mapLoading.value = true;
  try {
    const [devData, tagData] = await Promise.all([getDevices(), getTags()]);
    devices.value = devData;
    tags.value = tagData;
  } catch (err: any) {
    Notify.create({
      type: 'negative',
      message: 'Failed to load map: ' + (err?.message || String(err)),
    });
  } finally {
    mapLoading.value = false;
  }
}

// Root on top, devices in one row, each device's tags in a column under it
const graph = computed(() => {
  const nodes: Node[] = [
    { id: 'root', type: 'map', position: { x: 0, y: 0 }, data: { kind: 'root' } },
  ];
  const edges: Edge[] = [];
  const left = -((devices.value.length - 1) * 260) / 2;
  devices.value.forEach((d, i) => {
    const id = `device-${d.id}`;
    const x = left + i * 260;
    nodes.push({ id, type: 'map', position: { x, y: 120 }, data: { kind: 'device', item: d } });
    edges.push({ id: `root->${id}`, source: 'root', target: id });
    tags.value
      .filter((t) => t.deviceId === d.id)
      .forEach((t, j) => {
        const tagId = `tag-${t.id}`;
        nodes.push({
          id: tagId,
          type: 'map',
          position: { x, y: 300 + j * 160 },
          data: { kind: 'tag', item: t },
        });
        edges.push({ id: `${id}->${tagId}`, source: id, target: tagId });
      });
  });
  return { nodes, edges };
});

void loadMap();
</script>

<template>
  <q-page class="column" style="height: calc(100vh - 50px)">
    <div class="row items-center q-pa-sm q-gutter-sm">
      <q-btn
        color="negative"
        icon="restart_alt"
        label="Restart Server"
        :loading="restartLoading"
        @click="restartServer"
      />
      <q-btn
        color="primary"
        icon="refresh"
        label="Refresh Map"
        :loading="mapLoading"
        @click="loadMap"
      />
    </div>

    <div class="col relative-position" style="border: 1px solid #ddd; border-radius: 4px">
      <q-inner-loading :showing="mapLoading" />
      <VueFlow
        v-if="!mapLoading"
        :nodes="graph.nodes"
        :edges="graph.edges"
        :nodes-connectable="false"
        fit-view-on-init
        :max-zoom="1.5"
        @node-click="handleNodeClick"
      >
        <template #node-map="nodeProps"><MapNode v-bind="nodeProps" /></template>
        <Background />
        <Controls :show-interactive="false" />
      </VueFlow>
    </div>
  </q-page>
</template>
