import { ref } from 'vue';
import type { Device, Tag } from '@/components/models';
import { getTags, readDevices } from '../../../services';

// Tag and device names, for the pickers and the node summaries
export function useAutomationNames() {
  const floatTags = ref<Tag[]>([]);
  const boolTags = ref<Tag[]>([]);
  const devices = ref<Device[]>([]);
  const names = {
    tags: new Map<number, string>(),
    tagTypes: new Map<number, string>(),
    devices: new Map<number, string>(),
  };

  async function load() {
    const [tags, deviceList] = await Promise.all([getTags(), readDevices()]);
    floatTags.value = tags.filter((tag) => tag.valueType === 'tagvaluefloat');
    boolTags.value = tags.filter((tag) => tag.valueType === 'tagvalueboolean');
    devices.value = deviceList;
    tags.forEach((tag) => {
      names.tags.set(tag.id, tag.name);
      names.tagTypes.set(tag.id, tag.valueType);
    });
    deviceList.forEach((device) => names.devices.set(device.id, device.name));
  }

  return { floatTags, boolTags, devices, names, load };
}
