<!-- eslint-disable @typescript-eslint/no-explicit-any -->
<script setup lang="ts">
import { ref, onMounted, watch } from 'vue';
import type { TagValueFloat } from '@/components/models';
import { Notify } from 'quasar';
import { useRoute } from 'vue-router';
import { getTagValueFloat } from '../../services';

const rows = ref<TagValueFloat[]>([]);
const loading = ref(false);
const route = useRoute();

// Filters
const filterTagId = ref<number | null>(null);
const pagination = ref({ page: 1, rowsPerPage: 1000, rowsNumber: 0 });

const columns: any[] = [
  {
    name: 'tagId',
    label: 'Tag ID',
    align: 'left',
    field: 'tag_Id',
    sortable: true,
  },
  {
    name: 'time',
    label: 'Time',
    align: 'left',
    field: 'time',
    sortable: true,
    format: (val: string) => (val ? new Date(val).toLocaleString(undefined, { hour12: false }) : 'N/A'),
  },
  {
    name: 'value',
    label: 'Value',
    align: 'right',
    field: 'value',
    sortable: true,
  },
];

async function loadData() {
  loading.value = true;
  try {
    const page = pagination.value.page || 1;
    const perPage = pagination.value.rowsPerPage || 0;
    const offset = perPage === 0 ? 0 : (page - 1) * perPage;
    const count = perPage === 0 ? undefined : perPage;

    const params: { offset?: number; count?: number; tag_id?: number } = { offset };
    if (typeof count === 'number') {
      params.count = count;
    }
    if (filterTagId.value !== null && Number.isFinite(filterTagId.value)) {
      params.tag_id = filterTagId.value;
    }

    const res = await getTagValueFloat(params);
    if (Array.isArray(res)) {
      rows.value = res;
      pagination.value.rowsNumber = res.length;
    } else {
      rows.value = res.rows ?? res.items ?? [];
      pagination.value.rowsNumber = res.meta?.total ?? res.meta?.totalCount ?? rows.value.length;
    }
  } catch (err: any) {
    console.error('Failed to load tag float values', err);
    Notify.create({
      type: 'negative',
      message: 'Failed to load tag float values: ' + (err?.message || String(err)),
    });
  } finally {
    loading.value = false;
  }
}

const handleRequest = (props: any) => {
  if (props && props.pagination) {
    pagination.value.page = props.pagination.page ?? pagination.value.page;
    pagination.value.rowsPerPage = props.pagination.rowsPerPage ?? pagination.value.rowsPerPage;
    if (typeof props.pagination.rowsNumber === 'number') {
      pagination.value.rowsNumber = props.pagination.rowsNumber;
    }
  }
  void loadData();
};

function applyFilter() {
  pagination.value.page = 1;
  void loadData();
}

function syncFilterFromRoute() {
  const queryValue = route.query.tag_id;
  const raw = Array.isArray(queryValue) ? queryValue[0] : queryValue;
  if (!raw) {
    filterTagId.value = null;
    return;
  }

  const id = Number(raw);
  filterTagId.value = Number.isFinite(id) ? id : null;
}

onMounted(() => {
  syncFilterFromRoute();
  void loadData();
});

watch(
  () => route.query.tag_id,
  () => {
    syncFilterFromRoute();
    pagination.value.page = 1;
    void loadData();
  },
);
</script>

<template>
  <q-page class="q-pa-md">
    <!-- Filters -->
    <div class="row q-gutter-sm q-mb-md items-end">
      <q-input
        v-model.number="filterTagId"
        label="Tag ID (optional)"
        type="number"
        outlined
        dense
        clearable
        style="min-width: 160px"
      />
      <q-btn color="primary" icon="search" label="Load" :loading="loading" @click="applyFilter" />
    </div>

    <q-table
      title="Tag Values (Float)"
      v-model:pagination="pagination"
      :rows="rows"
      :columns="columns"
      row-key="time"
      :loading="loading"
      :rows-per-page-options="[1000, 2000, 5000, 100000, 0]"
      @request="handleRequest"
    >
      <template v-slot:body="props">
        <q-tr :props="props">
          <q-td key="tagId">
            {{ props.row.tagId ?? props.row.tagid ?? props.row.tag_Id ?? 'N/A' }}
          </q-td>
          <q-td key="time">
            {{ props.row.time ? new Date(props.row.time).toLocaleString(undefined, { hour12: false }) : 'N/A' }}
          </q-td>
          <q-td key="value" class="text-right">
            {{ props.row.value }}
          </q-td>
        </q-tr>
      </template>
    </q-table>
  </q-page>
</template>
