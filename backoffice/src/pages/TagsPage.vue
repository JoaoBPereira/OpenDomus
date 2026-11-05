<!-- eslint-disable @typescript-eslint/no-explicit-any -->
<script setup lang="ts">
import { ref, onMounted, watch, computed, nextTick } from 'vue';
import type { Tag } from '@/components/models';
import { Notify } from 'quasar';
import cronstrue from 'cronstrue';
import { useRoute, useRouter } from 'vue-router';
import { getTags, createTag, updateTag } from '../../services';

// Tags list displayed in table
const tags = ref<Tag[]>([]);
const loading = ref(false);
const route = useRoute();
const router = useRouter();
const selectedTagId = ref<number | null>(null);

// Dialog state
const showDialog = ref(false);
const dialogMode = ref<'create' | 'edit'>('create');
const editingTag = ref<Tag | null>(null);
const duplicateSourceTagId = ref<number | null>(null);
const displayInText = ref('');
const dialogForm = ref<Tag>({
  id: 0,
  name: '',
  type: '',
  valueType: '',
  valueStorage: '',
  config: '',
  deviceId: 0,
  deviceName: '',
  devicePeriodicity: '',
  periodicityCount: 0,
  expiresIn: 0,
  displayIn: [],
  enabled: false,
});

function normalizeDisplayIn(value: string): string[] {
  return Array.from(
    new Set(
      value
        .split(',')
        .map((entry) => entry.trim())
        .filter((entry) => entry.length > 0),
    ),
  );
}

function openCreateDialog() {
  dialogMode.value = 'create';
  editingTag.value = null;
  duplicateSourceTagId.value = null;
  dialogForm.value = {
    id: 0,
    name: '',
    type: '',
    valueType: '',
    valueStorage: '',
    config: '',
    deviceId: 0,
    deviceName: '',
    devicePeriodicity: '',
    periodicityCount: 0,
    expiresIn: 0,
    displayIn: [],
    enabled: false,
  };
  displayInText.value = '';
  showDialog.value = true;
}

function closeDialog() {
  showDialog.value = false;
  editingTag.value = null;
}

async function confirmDialog() {
  dialogForm.value.displayIn = normalizeDisplayIn(displayInText.value);

  if (dialogMode.value === 'create') {
    try {
      const created = await createTag(dialogForm.value);
      tags.value.push(created);
      Notify.create({ type: 'positive', message: 'Tag created successfully' });
      showDialog.value = false;
    } catch (err: any) {
      Notify.create({
        type: 'negative',
        message: `Failed to create tag: ${err?.message || String(err)}`,
      });
    }
  } else {
    if (!editingTag.value) return;
    try {
      await updateTag(dialogForm.value.id, dialogForm.value);
      Object.assign(editingTag.value, dialogForm.value);
      Notify.create({ type: 'positive', message: 'Tag updated successfully' });
      showDialog.value = false;
      editingTag.value = null;
    } catch (err: any) {
      Notify.create({
        type: 'negative',
        message: `Failed to update tag: ${err?.message || String(err)}`,
      });
    }
  }
}

const columns: any[] = [
  { name: 'id', required: true, label: 'ID', align: 'left', field: 'id' },
  { name: 'name', label: 'Name', align: 'left', field: 'name' },
  { name: 'type', label: 'Type', align: 'left', field: 'type' },
  { name: 'valueType', label: 'Value Type', align: 'left', field: 'valueType' },
  { name: 'valueStorage', label: 'Value Storage', align: 'left', field: 'valueStorage' },
  { name: 'config', label: 'Config', align: 'left', field: 'config' },
  { name: 'deviceId', label: 'Device ID', align: 'left', field: 'deviceId' },
  { name: 'periodicityCount', label: 'Periodicity', align: 'left', field: 'periodicityCount' },
  { name: 'expiresIn', label: 'Expires (Days)', align: 'left', field: 'expiresIn' },
  { name: 'displayIn', label: 'Display In', align: 'left', field: 'displayIn' },
  { name: 'enabled', label: 'Enabled', align: 'left', field: 'enabled' },
  { name: 'actions', label: 'Actions', align: 'center', field: 'actions', sortable: false },
];

async function loadTags() {
  loading.value = true;
  try {
    const data = await getTags();
    tags.value = data;
  } catch (err: any) {
    console.error('Failed to load tags', err);
    Notify.create({
      type: 'negative',
      message: 'Failed to load tags: ' + (err?.message || String(err)),
    });
  } finally {
    loading.value = false;
  }
}

async function clearEditQueryParam() {
  const query = { ...route.query };
  delete query.editTagId;
  await router.replace({ query });
}

async function openEditFromRouteQuery() {
  const queryValue = route.query.editTagId;
  const rawId = Array.isArray(queryValue) ? queryValue[0] : queryValue;
  if (!rawId) return;

  const id = Number(rawId);
  if (!Number.isFinite(id)) return;

  const targetTag = tags.value.find((t) => t.id === id);
  if (!targetTag) {
    Notify.create({
      type: 'warning',
      message: `Tag ${id} was not found`,
    });
    await clearEditQueryParam();
    return;
  }

  selectedTagId.value = id;
  await focusTagRow(id);
  await clearEditQueryParam();
}

async function focusTagRow(tagId: number) {
  await nextTick();
  const rowEl = document.getElementById(`tag-row-${tagId}`);
  if (!rowEl) return;

  rowEl.scrollIntoView({ behavior: 'smooth', block: 'center' });
  rowEl.focus();
}

onMounted(async () => {
  await loadTags();
  await openEditFromRouteQuery();
});

watch(
  () => route.query.editTagId,
  async (newValue) => {
    if (!newValue || loading.value || tags.value.length === 0) return;
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

function getCronIntervalMinutes(cron: string): number | null {
  const parts = cron.trim().split(/\s+/);
  if (parts.length !== 5) return null;

  const [minute, hour, day, month, weekday] = parts;

  if (minute?.startsWith('*/') && hour === '*' && day === '*' && month === '*' && weekday === '*') {
    const value = Number(minute.slice(2));
    return Number.isFinite(value) && value > 0 ? value : null;
  }

  if (minute === '0' && hour?.startsWith('*/') && day === '*' && month === '*' && weekday === '*') {
    const value = Number(hour.slice(2));
    return Number.isFinite(value) && value > 0 ? value * 60 : null;
  }

  if (minute === '0' && hour === '0' && day?.startsWith('*/') && month === '*' && weekday === '*') {
    const value = Number(day.slice(2));
    return Number.isFinite(value) && value > 0 ? value * 1440 : null;
  }

  if (minute === '0' && hour === '0' && day === '*' && month === '*' && weekday === '*') {
    return 1440;
  }

  return null;
}

function formatIntervalMinutes(totalMinutes: number): string {
  if (totalMinutes <= 0 || !Number.isFinite(totalMinutes)) return 'N/A';

  if (totalMinutes % 1440 === 0) {
    const days = totalMinutes / 1440;
    return `${days} day${days === 1 ? '' : 's'}`;
  }

  if (totalMinutes % 60 === 0) {
    const hours = totalMinutes / 60;
    return `${hours} hour${hours === 1 ? '' : 's'}`;
  }

  return `${totalMinutes} minute${totalMinutes === 1 ? '' : 's'}`;
}

function formatTagPeriodicity(tag: Tag): string | null {
  if (!tag.devicePeriodicity) return null;

  const count = Number(tag.periodicityCount);
  if (!Number.isFinite(count) || count <= 0) return null;

  const baseMinutes = getCronIntervalMinutes(tag.devicePeriodicity);
  if (baseMinutes) {
    return formatIntervalMinutes(baseMinutes * count);
  }

  return `${count} x ${formatCron(tag.devicePeriodicity)}`;
}

function formatDisplayIn(values?: string[]): string {
  if (!Array.isArray(values) || values.length === 0) return '—';
  return values.join(', ');
}

function openEditDialog(tag: Tag) {
  void nextTick(() => {
    dialogMode.value = 'edit';
    editingTag.value = tag;
    duplicateSourceTagId.value = null;
    dialogForm.value = {
      ...tag,
      displayIn: Array.isArray(tag.displayIn) ? [...tag.displayIn] : [],
    };
    displayInText.value = dialogForm.value.displayIn.join(', ');
    showDialog.value = true;
  });
}

const duplicateTagOptions = computed(() =>
  tags.value.map((tag) => ({
    label: `${tag.name} (ID: ${tag.id})`,
    value: tag.id,
  })),
);

function applyDuplicateSource(tagId: number | null) {
  if (!tagId) return;
  const sourceTag = tags.value.find((t) => t.id === tagId);
  if (!sourceTag) return;

  dialogForm.value = {
    ...sourceTag,
    id: 0,
    displayIn: Array.isArray(sourceTag.displayIn) ? [...sourceTag.displayIn] : [],
  };
  displayInText.value = dialogForm.value.displayIn.join(', ');
}

function openTagValues(tag: Tag) {
  void router.push({
    path: '/tag-value-float',
    query: { tag_id: String(tag.id) },
  });
}
</script>

<template>
  <q-page class="q-pa-md">
    <q-table
      title="Tags"
      :rows="tags"
      :columns="columns"
      row-key="id"
      :loading="loading"
      flat
      bordered
      separator="vertical"
      :rows-per-page-options="[50, 100, 0]"
    >
      <template v-slot:top-right>
        <q-btn color="primary" icon="add" label="New Tag" @click="openCreateDialog" />
      </template>
      <template v-slot:body="props">
        <q-tr
          :id="`tag-row-${props.row.id}`"
          :props="props"
          :class="{ 'selected-tag-row': selectedTagId === props.row.id }"
          tabindex="-1"
          @click="
            selectedTagId = props.row.id;
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
            {{ props.row.valueType }}
          </q-td>

          <q-td>
            {{ props.row.valueStorage }}
          </q-td>

          <q-td>
            {{ props.row.config }}
          </q-td>

          <q-td>
            {{ props.row.deviceId }}
            <span v-if="props.row.deviceName"> ({{ props.row.deviceName }})</span>
          </q-td>

          <q-td>
            {{ props.row.periodicityCount }}
            -
            <span class="text-caption text-grey-7 q-ml-xs">
              ({{
                props.row.devicePeriodicity ? `x ${formatCron(props.row.devicePeriodicity)}` : ''
              }})
            </span>
            -
            <span v-if="formatTagPeriodicity(props.row)" class="text-caption text-grey-7 q-ml-xs">
              ({{ formatTagPeriodicity(props.row) }})
            </span>
          </q-td>

          <q-td>
            {{ props.row.expiresIn }}
          </q-td>

          <q-td>
            {{ formatDisplayIn(props.row.displayIn) }}
          </q-td>

          <q-td>
            <q-icon
              :name="props.row.enabled ? 'check_circle' : 'cancel'"
              :color="props.row.enabled ? 'positive' : 'negative'"
            />
          </q-td>
          <q-td auto-width @click.stop>
            <q-btn
              dense
              flat
              color="primary"
              icon="show_chart"
              label="View Values"
              @click.stop="openTagValues(props.row)"
            />
          </q-td>
        </q-tr>
      </template>
    </q-table>

    <!-- Create / Edit Tag Dialog -->
    <q-dialog v-model="showDialog">
      <q-card style="min-width: 500px">
        <q-card-section>
          <div class="text-h6">
            {{ dialogMode === 'create' ? 'New Tag' : 'Edit Tag' }}
          </div>
        </q-card-section>

        <q-card-section class="q-pt-none">
          <q-select
            v-if="dialogMode === 'create'"
            v-model="duplicateSourceTagId"
            :options="duplicateTagOptions"
            label="Duplicate From Existing Tag (optional)"
            outlined
            dense
            emit-value
            map-options
            clearable
            class="q-mb-md"
            @update:model-value="applyDuplicateSource"
          />

          <q-input v-model="dialogForm.name" label="Name" outlined dense class="q-mb-md" />

          <q-input v-model="dialogForm.type" label="Type" outlined dense class="q-mb-md" />

          <q-input
            v-model="dialogForm.valueType"
            label="Value Type"
            outlined
            dense
            class="q-mb-md"
          />

          <q-input
            v-model="dialogForm.valueStorage"
            label="Value Storage"
            outlined
            dense
            class="q-mb-md"
          />

          <q-input v-model="dialogForm.config" label="Config" outlined dense class="q-mb-md" />

          <q-input
            v-model.number="dialogForm.deviceId"
            label="Device ID"
            type="number"
            outlined
            dense
            class="q-mb-md"
          />

          <q-input
            v-model.number="dialogForm.periodicityCount"
            label="Periodicity Count"
            type="number"
            outlined
            dense
            class="q-mb-md"
          />

          <q-input
            v-model.number="dialogForm.expiresIn"
            label="Expires In (days)"
            type="number"
            outlined
            dense
            class="q-mb-md"
          />

          <q-input
            v-model="displayInText"
            label="Display In (comma-separated)"
            outlined
            dense
            class="q-mb-md"
            hint="Example: dashboard, mobile, sidebar"
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
.selected-tag-row {
  background: rgba(25, 118, 210, 0.12);
  outline: 2px solid rgba(25, 118, 210, 0.55);
  outline-offset: -2px;
}
</style>
