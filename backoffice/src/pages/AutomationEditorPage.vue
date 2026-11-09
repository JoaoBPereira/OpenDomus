<!-- eslint-disable @typescript-eslint/no-explicit-any -->
<script setup lang="ts">
import { computed, onMounted, provide, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { Notify } from 'quasar';
import { VueFlow, useVueFlow, type Connection, type Edge, type Node } from '@vue-flow/core';
import { Background } from '@vue-flow/background';
import { Controls } from '@vue-flow/controls';
import '@vue-flow/core/dist/style.css';
import '@vue-flow/core/dist/theme-default.css';
import '@vue-flow/controls/dist/style.css';
import type { AutomationNode, AutomationNodeType } from '@/components/models';
import {
  createAutomation,
  deleteAutomation,
  getAutomation,
  updateAutomation,
} from '../../services';
import FlowNode from '@/components/automation/FlowNode.vue';
import {
  canConnect,
  formatCron,
  formatOffset,
  formatSeconds,
  fromFlow,
  newNode,
  nodeInfo,
  nodeTypes,
  toFlow,
} from '@/components/automation/graph';
import { useAutomationNames } from '@/components/automation/names';

const route = useRoute();
const router = useRouter();
const { addNodes, addEdges, removeNodes, findNode, getNodes, getEdges, getViewport, dimensions } =
  useVueFlow();
const { floatTags, boolTags, devices, names, load: loadNames } = useAutomationNames();
// condition and waitCheck check a float or a boolean tag
const checkTags = computed(() => [...floatTags.value, ...boolTags.value]);
const isBoolTag = (tagId?: number) => names.tagTypes.get(tagId ?? 0) === 'tagvalueboolean';
provide('automationNames', names);

const id = computed(() => (route.params.id === 'new' ? null : Number(route.params.id)));
const name = ref('');
const enabled = ref(true);
const initialNodes = ref<Node<AutomationNode>[]>([]);
const initialEdges = ref<Edge[]>([]);
const loaded = ref(false);
const saving = ref(false);

const operators = ['>', '<', '=', '>=', '<='];
const methods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'];

// Node settings panel
const selectedId = ref<string | null>(null);
const selected = computed(() =>
  selectedId.value ? findNode<AutomationNode>(selectedId.value)?.data : undefined,
);
const headerRows = ref<{ key: string; value: string }[]>([]);

watch(selectedId, () => {
  headerRows.value = Object.entries(selected.value?.headers ?? {}).map(([key, value]) => ({
    key,
    value,
  }));
});
watch(
  headerRows,
  (rows) => {
    if (selected.value?.type === 'http') {
      selected.value.headers = Object.fromEntries(
        rows.filter((row) => row.key).map((row) => [row.key, row.value]),
      );
    }
  },
  { deep: true },
);

function flowNode(node: AutomationNode, position: { x: number; y: number }): Node<AutomationNode> {
  return { id: node.id, type: nodeInfo(node.type).category, position, data: node };
}

function addNode(type: AutomationNodeType) {
  const viewport = getViewport();
  const node = newNode(type);
  addNodes([
    flowNode(node, {
      x: (dimensions.value.width / 2 - viewport.x) / viewport.zoom - 70,
      y: (dimensions.value.height / 2 - viewport.y) / viewport.zoom - 25,
    }),
  ]);
  selectedId.value = node.id;
}

function removeSelected() {
  if (!selectedId.value) return;
  removeNodes([selectedId.value]);
  selectedId.value = null;
}

function isValidConnection(connection: Connection) {
  const source = findNode<AutomationNode>(connection.source)?.data;
  const target = findNode<AutomationNode>(connection.target)?.data;
  return !!source && !!target && source.id !== target.id && canConnect(source.type, target.type);
}

function onConnect(connection: Connection) {
  addEdges([{ ...connection, id: `${connection.source}->${connection.target}` }]);
}

function errorMessage(err: any) {
  return err?.response?.data?.message || err?.message || String(err);
}

async function load() {
  try {
    await loadNames();
    if (id.value === null) {
      const trigger = route.query.trigger === 'tagValueFloat' ? 'tagValueFloat' : 'timer';
      initialNodes.value = [flowNode(newNode(trigger), { x: 50, y: 100 })];
    } else {
      const automation = await getAutomation(id.value);
      name.value = automation.name;
      enabled.value = automation.enabled;
      const flow = toFlow(automation);
      initialNodes.value = flow.nodes;
      initialEdges.value = flow.edges;
    }
    loaded.value = true;
  } catch (err: any) {
    Notify.create({ type: 'negative', message: 'Failed to load automation: ' + errorMessage(err) });
  }
}

async function save() {
  saving.value = true;
  const automation = {
    name: name.value,
    enabled: enabled.value,
    graph: fromFlow(getNodes.value, getEdges.value),
  };
  try {
    if (id.value === null) {
      const created = await createAutomation(automation);
      await router.replace(`/automations/${created.id}`);
    } else {
      await updateAutomation(id.value, { id: id.value, ...automation });
    }
    Notify.create({ type: 'positive', message: 'Automation saved' });
  } catch (err: any) {
    Notify.create({ type: 'negative', message: 'Not saved: ' + errorMessage(err) });
  } finally {
    saving.value = false;
  }
}

async function remove() {
  if (id.value === null || !confirm(`Delete automation "${name.value}"?`)) return;
  try {
    await deleteAutomation(id.value);
    router.back();
  } catch (err: any) {
    Notify.create({ type: 'negative', message: 'Not deleted: ' + errorMessage(err) });
  }
}

onMounted(() => {
  void load();
});
</script>

<template>
  <q-page class="column q-pa-md" style="height: calc(100vh - 50px)">
    <div class="row items-center q-gutter-sm q-mb-sm">
      <q-btn flat round icon="arrow_back" @click="router.back()" />
      <q-input v-model="name" label="Name" outlined dense style="min-width: 280px" />
      <q-toggle v-model="enabled" label="Enabled" />
      <q-space />
      <q-btn
        v-if="id !== null"
        flat
        color="negative"
        icon="delete"
        label="Delete"
        @click="remove"
      />
      <q-btn color="primary" icon="save" label="Save" :loading="saving" @click="save" />
    </div>

    <div class="row q-gutter-xs q-mb-sm">
      <q-btn
        v-for="info in nodeTypes"
        :key="info.type"
        outline
        dense
        no-caps
        :icon="info.icon"
        :label="info.label"
        :color="
          info.category === 'trigger' ? 'primary' : info.category === 'gate' ? 'purple' : 'positive'
        "
        @click="addNode(info.type)"
      />
    </div>

    <div class="col row no-wrap">
      <div class="col" style="border: 1px solid #ddd; border-radius: 4px">
        <VueFlow
          v-if="loaded"
          :nodes="initialNodes"
          :edges="initialEdges"
          :is-valid-connection="isValidConnection"
          :delete-key-code="['Backspace', 'Delete']"
          fit-view-on-init
          :max-zoom="1.5"
          @connect="onConnect"
          @node-click="(event) => (selectedId = event.node.id)"
          @pane-click="selectedId = null"
        >
          <template #node-trigger="nodeProps"><FlowNode v-bind="nodeProps" /></template>
          <template #node-gate="nodeProps"><FlowNode v-bind="nodeProps" /></template>
          <template #node-action="nodeProps"><FlowNode v-bind="nodeProps" /></template>
          <Background />
          <Controls />
        </VueFlow>
      </div>

      <q-card v-if="selected" flat bordered class="q-ml-sm" style="width: 320px; overflow: auto">
        <q-card-section class="row items-center">
          <q-icon :name="nodeInfo(selected.type).icon" size="20px" class="q-mr-sm" />
          <div class="text-h6">{{ nodeInfo(selected.type).label }}</div>
          <q-space />
          <q-btn flat round dense icon="delete" color="negative" @click="removeSelected" />
        </q-card-section>

        <q-card-section class="q-pt-none q-gutter-md">
          <template v-if="selected.type === 'timer'">
            <q-input
              v-model="selected.cron"
              label="Cron"
              outlined
              dense
              :hint="formatCron(selected.cron)"
            />
            <div class="text-caption text-grey-7">
              minute hour day month weekday, e.g. 30 19 * * *
            </div>
          </template>

          <template v-else-if="selected.type === 'sunrise' || selected.type === 'sunset'">
            <q-input
              v-model.number="selected.offsetMinutes"
              type="number"
              label="Offset (minutes)"
              outlined
              dense
              :hint="formatOffset(selected.type, selected.offsetMinutes)"
            />
            <div class="text-caption text-grey-7">
              Negative is before, positive is after. The time follows the real
              {{ selected.type }} every day.
            </div>
          </template>

          <template v-else-if="selected.type === 'tagValueBoolean'">
            <q-select
              v-model="selected.tagId"
              :options="boolTags"
              option-value="id"
              option-label="name"
              emit-value
              map-options
              label="Tag"
              outlined
              dense
            />
            <q-toggle
              v-model="selected.boolValue"
              :label="selected.boolValue ? 'Is on' : 'Is off'"
            />
          </template>

          <template
            v-else-if="
              selected.type === 'tagValueFloat' ||
              selected.type === 'condition' ||
              selected.type === 'waitCheck'
            "
          >
            <div v-if="selected.type === 'condition'" class="text-body2">
              Passes when its input is true and the last value of the tag matches. A new value does
              not start the automation by itself.
            </div>
            <div v-if="selected.type === 'waitCheck'" class="text-body2">
              When its input becomes true, waits and then checks the last value of the tag again.
              Passes once if it matches.
            </div>
            <q-input
              v-if="selected.type === 'waitCheck'"
              v-model.number="selected.delaySeconds"
              type="number"
              label="Wait (seconds)"
              outlined
              dense
              :hint="formatSeconds(selected.delaySeconds)"
            />
            <q-select
              v-model="selected.tagId"
              :options="selected.type === 'tagValueFloat' ? floatTags : checkTags"
              option-value="id"
              option-label="name"
              emit-value
              map-options
              label="Tag"
              outlined
              dense
            />
            <q-toggle
              v-if="selected.type !== 'tagValueFloat' && isBoolTag(selected.tagId)"
              v-model="selected.boolValue"
              :label="selected.boolValue ? 'Is on' : 'Is off'"
            />
            <template v-else>
              <q-select
                v-model="selected.operator"
                :options="operators"
                label="Operator"
                outlined
                dense
              />
              <q-input v-model.number="selected.value" type="number" label="Value" outlined dense />
            </template>
          </template>

          <template v-else-if="selected.type === 'and' || selected.type === 'or'">
            <div class="text-body2">
              {{
                selected.type === 'and'
                  ? 'True when every input is true.'
                  : 'True when one input is true.'
              }}
            </div>
          </template>

          <template v-else-if="selected.type === 'whatsapp'">
            <q-input
              v-model="selected.text"
              label="Message"
              type="textarea"
              outlined
              dense
              autogrow
            />
          </template>

          <template v-else-if="selected.type === 'device'">
            <q-select
              v-model="selected.deviceId"
              :options="devices"
              option-value="id"
              option-label="name"
              emit-value
              map-options
              label="Device"
              outlined
              dense
            />
            <q-toggle v-model="selected.on" :label="selected.on ? 'Turn on' : 'Turn off'" />
          </template>

          <template v-else-if="selected.type === 'http'">
            <q-select v-model="selected.method" :options="methods" label="Method" outlined dense />
            <q-input v-model="selected.url" label="URL" outlined dense />
            <div>
              <div class="text-subtitle2">Headers</div>
              <div
                v-for="(row, index) in headerRows"
                :key="index"
                class="row no-wrap items-center q-gutter-xs q-mb-xs"
              >
                <q-input v-model="row.key" label="Name" outlined dense class="col" />
                <q-input v-model="row.value" label="Value" outlined dense class="col" />
                <q-btn flat round dense icon="close" @click="headerRows.splice(index, 1)" />
              </div>
              <q-btn
                flat
                dense
                no-caps
                icon="add"
                label="Add header"
                @click="headerRows.push({ key: '', value: '' })"
              />
            </div>
            <q-input v-model="selected.body" label="Body" type="textarea" outlined dense autogrow />
          </template>
        </q-card-section>
      </q-card>
    </div>
  </q-page>
</template>
