<!-- eslint-disable @typescript-eslint/no-explicit-any -->
<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { Notify } from 'quasar';
import type { Automation } from '@/components/models';
import { getAutomations, getLogs, updateAutomation } from '../../services';
import { nodeInfo, summary, timeTriggers, valueTriggers } from '@/components/automation/graph';
import { useAutomationNames } from '@/components/automation/names';

// Lists the automations that have a trigger of the given type (one menu entry per type)
const props = defineProps<{ trigger: 'timer' | 'tagValueFloat' }>();

const router = useRouter();
const automations = ref<Automation[]>([]);
const loading = ref(false);
const { names, load: loadNames } = useAutomationNames();

const title = computed(() =>
  props.trigger === 'timer' ? 'Automations Time' : 'Automations Values',
);
const rows = computed(() =>
  automations.value.filter((automation) =>
    automation.graph.nodes.some((node) =>
      (props.trigger === 'timer' ? timeTriggers : valueTriggers).includes(node.type),
    ),
  ),
);

const columns: any[] = [
  { name: 'id', label: 'ID', align: 'right', field: 'id' },
  { name: 'name', label: 'Name', align: 'left', field: 'name' },
  { name: 'triggers', label: 'Triggers', align: 'left', field: 'id' },
  { name: 'actions', label: 'Actions', align: 'left', field: 'id' },
  { name: 'enabled', label: 'Enabled', align: 'left', field: 'enabled' },
  { name: 'lastRun', label: 'View Last Run', align: 'left', field: 'id' },
];

// the logs of the last run of the automation shown in the dialog
const lastRun = ref<{ name: string; logs: any[] } | null>(null);

// every run logs "<name>: ..." with run=<id>: find the newest run id, then all the lines of that run
async function viewLastRun(automation: Automation) {
  try {
    const prefix = automation.name + ': ';
    const recent = await getLogs(prefix, 0, 100);
    const run = recent.rows
      .filter((row: any) => row.message?.includes(prefix))
      .map((row: any) => /\brun=(\w+)/.exec(row.message)?.[1])
      .find(Boolean);
    let logs: any[] = [];
    if (run) {
      const res = await getLogs('run=' + run, 0, 1000);
      // run ids are short and can repeat across automations, so keep this automation's lines
      logs = res.rows
        .filter((row: any) => row.message?.includes(prefix) && new RegExp('\\brun=' + run + '\\b').test(row.message))
        .reverse();
    }
    lastRun.value = { name: automation.name, logs };
  } catch (err: any) {
    Notify.create({
      type: 'negative',
      message: 'Failed to load the last run: ' + (err?.message || String(err)),
    });
  }
}

function triggers(automation: Automation) {
  return automation.graph.nodes
    .filter((node) => nodeInfo(node.type).category === 'trigger')
    .map((node) => summary(node, names))
    .join(' · ');
}

function actions(automation: Automation) {
  return automation.graph.nodes
    .filter((node) => nodeInfo(node.type).category === 'action')
    .map((node) => nodeInfo(node.type).label)
    .join(', ');
}

async function load() {
  loading.value = true;
  try {
    await loadNames();
    automations.value = await getAutomations();
  } catch (err: any) {
    Notify.create({
      type: 'negative',
      message: 'Failed to load automations: ' + (err?.message || String(err)),
    });
  } finally {
    loading.value = false;
  }
}

async function setEnabled(automation: Automation, enabled: boolean) {
  try {
    Object.assign(automation, await updateAutomation(automation.id, { ...automation, enabled }));
  } catch (err: any) {
    Notify.create({
      type: 'negative',
      message:
        'Failed to update automation: ' +
        (err?.response?.data?.message || err?.message || String(err)),
    });
  }
}

onMounted(() => {
  void load();
});
</script>

<template>
  <q-page class="q-pa-md">
    <q-table
      :title="title"
      :rows="rows"
      :columns="columns"
      row-key="id"
      :loading="loading"
      flat
      bordered
      separator="vertical"
      :rows-per-page-options="[50, 100, 0]"
    >
      <template v-slot:top-right>
        <q-btn
          color="primary"
          icon="add"
          label="New"
          @click="router.push({ path: '/automations/new', query: { trigger: props.trigger } })"
        />
      </template>
      <template v-slot:body="slot">
        <q-tr
          :props="slot"
          style="cursor: pointer"
          @click="router.push(`/automations/${slot.row.id}`)"
        >
          <q-td auto-width>{{ slot.row.id }}</q-td>
          <q-td>{{ slot.row.name }}</q-td>
          <q-td>{{ triggers(slot.row) }}</q-td>
          <q-td>{{ actions(slot.row) }}</q-td>
          <q-td auto-width @click.stop>
            <q-toggle
              :model-value="slot.row.enabled"
              @update:model-value="(value) => setEnabled(slot.row, value)"
            />
          </q-td>
          <q-td auto-width @click.stop>
            <q-btn flat dense icon="history" label="View" @click="viewLastRun(slot.row)" />
          </q-td>
        </q-tr>
      </template>
    </q-table>

    <q-dialog :model-value="lastRun !== null" @update:model-value="lastRun = null">
      <q-card v-if="lastRun" style="min-width: 60vw; max-width: 90vw">
        <q-card-section class="text-h6">Last run: {{ lastRun.name }}</q-card-section>
        <q-card-section v-if="!lastRun.logs.length">No run found in the logs.</q-card-section>
        <q-markup-table v-else flat dense wrap-cells>
          <tbody>
            <tr v-for="(log, index) in lastRun.logs" :key="index">
              <td class="text-no-wrap">{{ log.date }}</td>
              <td>{{ log.level }}</td>
              <td>{{ log.message }}</td>
            </tr>
          </tbody>
        </q-markup-table>
        <q-card-actions align="right">
          <q-btn flat label="Close" v-close-popup />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </q-page>
</template>
