<!-- eslint-disable @typescript-eslint/no-explicit-any -->
<script setup lang="ts">
import { ref, onMounted, watch, nextTick } from 'vue';
import type { Device } from '@/components/models';
import { Notify } from 'quasar';
import cronstrue from 'cronstrue';
import { useRoute, useRouter } from 'vue-router';
import { readDevices, createDevice, updateDevice } from '../../services';

// Devices list displayed in table
const devices = ref<Device[]>([]);
const loading = ref(false);
const route = useRoute();
const router = useRouter();
const selectedDeviceId = ref<number | null>(null);

// Dialog state
const showDialog = ref(false);
const dialogMode = ref<'create' | 'edit'>('create');
const editingDevice = ref<Device | null>(null);
const dialogForm = ref<Device>({
  id: 0,
  name: '',
  type: '',
  url: '',
  config: '',
  periodicity: null,
  enabled: false,
});

function openCreateDialog() {
  dialogMode.value = 'create';
  editingDevice.value = null;
  dialogForm.value = {
    id: 0,
    name: '',
    type: '',
    url: '',
    config: '',
    periodicity: null,
    enabled: false,
  };
  showDialog.value = true;
}

function closeDialog() {
  showDialog.value = false;
  editingDevice.value = null;
}

async function confirmDialog() {
  if (dialogMode.value === 'create') {
    try {
      const created = await createDevice(dialogForm.value);
      devices.value.push(created);
      Notify.create({ type: 'positive', message: 'Device created successfully' });
      showDialog.value = false;
    } catch (err: any) {
      Notify.create({
        type: 'negative',
        message: `Failed to create device: ${err?.message || String(err)}`,
      });
    }
  } else {
    if (!editingDevice.value) return;
    try {
      await updateDevice(dialogForm.value.id, dialogForm.value);
      Object.assign(editingDevice.value, dialogForm.value);
      Notify.create({ type: 'positive', message: 'Device updated successfully' });
      showDialog.value = false;
      editingDevice.value = null;
    } catch (err: any) {
      Notify.create({
        type: 'negative',
        message: `Failed to update device: ${err?.message || String(err)}`,
      });
    }
  }
}

const columns: any[] = [
  { name: 'id', required: true, label: 'ID', align: 'right', field: 'id' },
  { name: 'name', label: 'Name', align: 'left', field: 'name' },
  { name: 'type', label: 'Type', align: 'left', field: 'type' },
  { name: 'url', label: 'URL', align: 'left', field: 'url' },
  { name: 'config', label: 'Config', align: 'left', field: 'config' },
  { name: 'periodicity', label: 'Periodicity', align: 'left', field: 'periodicity' },
  { name: 'enabled', label: 'Enabled', align: 'left', field: 'enabled' },
];

async function loadDevices() {
  loading.value = true;
  try {
    const data = await readDevices();
    if (Array.isArray(data)) devices.value = data;
  } catch (err: any) {
    console.error('Failed to load devices', err);
    Notify.create({
      type: 'negative',
      message: 'Failed to load devices: ' + (err?.message || String(err)),
    });
  } finally {
    loading.value = false;
  }
}

async function clearEditQueryParam() {
  const query = { ...route.query };
  delete query.editDeviceId;
  await router.replace({ query });
}

async function openEditFromRouteQuery() {
  const queryValue = route.query.editDeviceId;
  const rawId = Array.isArray(queryValue) ? queryValue[0] : queryValue;
  if (!rawId) return;

  const id = Number(rawId);
  if (!Number.isFinite(id)) return;

  const targetDevice = devices.value.find((d) => d.id === id);
  if (!targetDevice) {
    Notify.create({
      type: 'warning',
      message: `Device ${id} was not found`,
    });
    await clearEditQueryParam();
    return;
  }

  selectedDeviceId.value = id;
  await focusDeviceRow(id);
  await clearEditQueryParam();
}

async function focusDeviceRow(deviceId: number) {
  await nextTick();
  const rowEl = document.getElementById(`device-row-${deviceId}`);
  if (!rowEl) return;

  rowEl.scrollIntoView({ behavior: 'smooth', block: 'center' });
  rowEl.focus();
}

function openEditDialog(device: Device) {
  void nextTick(() => {
    dialogMode.value = 'edit';
    editingDevice.value = device;
    dialogForm.value = { ...device };
    showDialog.value = true;
  });
}

onMounted(async () => {
  await loadDevices();
  await openEditFromRouteQuery();
});

watch(
  () => route.query.editDeviceId,
  async (newValue) => {
    if (!newValue || loading.value || devices.value.length === 0) return;
    await openEditFromRouteQuery();
  },
);

function formatCron(cron?: string | null) {
  if (!cron) return 'N/A';
  try {
    return cronstrue.toString(cron, { throwExceptionOnParseError: true, use24HourTimeFormat: true });
  } catch {
    return 'Invalid cron';
  }
}
</script>

<template>
  <q-page class="q-pa-md">
    <q-table
      title="Devices"
      :rows="devices"
      :columns="columns"
      row-key="id"
      :loading="loading"
      flat
      bordered
      separator="vertical"
      :rows-per-page-options="[50, 100, 0]"
    >
      <template v-slot:top-right>
        <q-btn color="primary" icon="add" label="New Device" @click="openCreateDialog" />
      </template>
      <template v-slot:body="props">
        <q-tr
          :id="`device-row-${props.row.id}`"
          :props="props"
          :class="{ 'selected-device-row': selectedDeviceId === props.row.id }"
          tabindex="-1"
          @click="
            selectedDeviceId = props.row.id;
            openEditDialog(props.row);
          "
          style="cursor: pointer"
        >
          <q-td auto-width>
            {{ props.row.id }}
          </q-td>
          <q-td>
            {{ props.row.name }}
          </q-td>
          <q-td>
            {{ props.row.type }}
          </q-td>
          <q-td>
            {{ props.row.url }}
          </q-td>
          <q-td>
            {{ props.row.config }}
          </q-td>
          <q-td>
            {{ props.row.periodicity ?? 'N/A' }}
            <span v-if="props.row.periodicity" class="text-caption text-grey-7 q-ml-xs">
              ({{ formatCron(props.row.periodicity) }})
            </span>
          </q-td>
          <q-td>
            <q-icon
              :name="props.row.enabled ? 'check_circle' : 'cancel'"
              :color="props.row.enabled ? 'positive' : 'negative'"
            />
          </q-td>
        </q-tr>
      </template>
    </q-table>

    <!-- Create / Edit Device Dialog -->
    <q-dialog v-model="showDialog">
      <q-card style="min-width: 500px">
        <q-card-section>
          <div class="text-h6">{{ dialogMode === 'create' ? 'New Device' : 'Edit Device' }}</div>
        </q-card-section>

        <q-card-section class="q-pt-none">
          <q-input v-model="dialogForm.name" label="Name" outlined dense class="q-mb-md" />

          <q-input v-model="dialogForm.type" label="Type" outlined dense class="q-mb-md" />

          <q-input v-model="dialogForm.url" label="URL" outlined dense class="q-mb-md" />

          <q-input v-model="dialogForm.config" label="Config" outlined dense class="q-mb-md" />

          <q-input
            v-model="dialogForm.periodicity"
            label="Periodicity"
            outlined
            dense
            class="q-mb-md"
          />

          <q-toggle v-model="dialogForm.enabled" label="Enabled" />
        </q-card-section>

        <q-card-actions align="right">
          <q-btn flat label="Cancel" color="primary" @click="closeDialog" />
          <q-btn
            flat
            :label="dialogMode === 'create' ? 'Create' : 'Confirm'"
            color="primary"
            @click="confirmDialog"
          />
        </q-card-actions>
      </q-card>
    </q-dialog>
  </q-page>
</template>

<style scoped>
.selected-device-row {
  background: rgba(25, 118, 210, 0.12);
  outline: 2px solid rgba(25, 118, 210, 0.55);
  outline-offset: -2px;
}
</style>
