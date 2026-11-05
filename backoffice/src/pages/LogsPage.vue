<!-- eslint-disable @typescript-eslint/no-explicit-any -->
<script setup lang="ts">
import { ref, onMounted, watch } from 'vue';
import { Notify } from 'quasar';
import { getLogs as svcGetLogs } from '../../services';

// Logs list displayed in table
const logs = ref<any[]>([]);
const loading = ref(false);
const selectedLogLevel = ref<{ label: string; value: string }>({ label: 'All', value: '' });
const messageFilter = ref('');

const pagination = ref({ page: 1, rowsPerPage: 1000, rowsNumber: 0 });

const logLevels = [
  { label: 'All', value: '' },
  { label: 'LOG', value: 'LOG' },
  { label: 'INFO', value: 'INFO' },
  { label: 'DEBUG', value: 'DEBUG' },
  { label: 'ERROR', value: 'ERROR' },
];

const columns: any[] = [
  { name: 'id', required: true, label: 'ID', align: 'right', field: 'id' },
  { name: 'date', label: 'Date', align: 'left', field: 'date' },
  { name: 'level', label: 'Level', align: 'left', field: 'level' },
  { name: 'message', label: 'Message', align: 'left', field: 'message' },
];

async function loadLogs() {
  loading.value = true;
  try {
    const page = pagination.value.page || 1;
    const perPage = pagination.value.rowsPerPage || 0;
    const offset = perPage === 0 ? 0 : (page - 1) * perPage;
    const count = perPage === 0 ? undefined : perPage;

    const res = await svcGetLogs(
      messageFilter.value.trim(),
      offset,
      count,
      selectedLogLevel.value?.value || undefined,
    );

    logs.value = res.rows;
    pagination.value.rowsNumber = res.meta.total;
  } catch (err: any) {
    console.error('Failed to load logs', err);
    Notify.create({
      type: 'negative',
      message: 'Failed to load logs: ' + (err?.message || String(err)),
    });
  } finally {
    loading.value = false;
  }
}

/**
 * Handle QTable request event: merge incoming pagination state and reload logs.
 * The QTable emits an object like { pagination, filter, sort, ... }.
 */
const handleRequest = (props: any) => {
  if (props && props.pagination) {
    // merge known pagination fields to avoid strict typing issues
    pagination.value.page = props.pagination.page ?? pagination.value.page;
    pagination.value.rowsPerPage = props.pagination.rowsPerPage ?? pagination.value.rowsPerPage;
    if (typeof props.pagination.rowsNumber === 'number') {
      pagination.value.rowsNumber = props.pagination.rowsNumber;
    }
  }
  void loadLogs();
};

watch(messageFilter, () => {
  if (messageFilter.value.trim().length > 3) {
    void loadLogs();
  }
});

onMounted(() => {
  void loadLogs();
});
</script>

<template>
  <q-page class="q-pa-md">
    <div class="q-mb-md row items-center q-col-gutter-md">
      <q-select
        v-model="selectedLogLevel"
        :options="logLevels"
        label="Log Level"
        outlined
        dense
        style="max-width: 200px"
        @update:model-value="loadLogs"
      />
      <q-input
        v-model="messageFilter"
        label="Message contains"
        outlined
        dense
        clearable
        class="col"
      />
    </div>
    <q-table
      row-key="id"
      title="Logs"
      v-model:pagination="pagination"
      :rows="logs"
      :columns="columns"
      :loading="loading"
      :rows-per-page-options="[1000, 2000, 5000, 100000, 0]"
      @request="handleRequest"
    >
      <template v-slot:body="props">
        <q-tr :props="props">
          <q-td auto-width style="cursor: pointer">
            {{ props.row.id }}
          </q-td>
          <q-td>
            {{ props.row.date }}
          </q-td>
          <q-td>
            {{ props.row.level }}
          </q-td>
          <q-td>
            {{ props.row.message }}
          </q-td>
        </q-tr>
      </template>
    </q-table>
  </q-page>
</template>
