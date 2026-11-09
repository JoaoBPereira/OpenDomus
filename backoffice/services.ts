import { api } from '@/boot/axios';
import type { Device, TagValueFloat } from '@/components/models';
import type { Tag } from '@/components/models';
import type { Automation } from '@/components/models';

export type PaginatedResult<T> = { items: T[]; total: number };

export type LogEntry = {
  id: number | string;
  date: string;
  level: string;
  message: string;
};

export async function login(username: string, password: string) {
  const res = await api.post('/login', { username, password });
  return res.data;
}

export async function validateJWT() {
  const res = await api.get('/validJWT');
  return res.data;
}

export async function readDevices() {
  const res = await api.get('/devices');
  return res.data as Device[];
}

export async function getDevices() {
  const res = await api.get('/getDevices');
  return res.data as Device[];
}

export async function reloadDevices() {
  const res = await api.post('/devices/reload', {});
  return res.data;
}

export async function createDevice(device: Omit<Device, 'id'>) {
  const res = await api.post('/devices', { device: device });
  return res.data as Device;
}

export async function updateDevice(id: number | string, device: Device) {
  const res = await api.put(`/devices/${id}`, { device: device });
  return res.data;
}

export async function getAutomations() {
  const res = await api.get('/automations');
  return res.data as Automation[];
}

export async function getAutomation(id: number | string) {
  const res = await api.get(`/automations/${id}`);
  return res.data as Automation;
}

export async function createAutomation(automation: Omit<Automation, 'id'>) {
  const res = await api.post('/automations', automation);
  return res.data as Automation;
}

export async function updateAutomation(id: number | string, automation: Automation) {
  const res = await api.put(`/automations/${id}`, automation);
  return res.data as Automation;
}

export async function deleteAutomation(id: number | string) {
  const res = await api.delete(`/automations/${id}`);
  return res.data;
}

export async function getTags() {
  const res = await api.get('/tags');
  return res.data as Tag[];
}

export async function reloadTags() {
  const res = await api.post('/tags/reload', {});
  return res.data;
}

export async function createTag(tag: Omit<Tag, 'id' | 'deviceName' | 'devicePeriodicity'>) {
  const res = await api.post('/tags', { tag: tag });
  return res.data as Tag;
}

export async function updateTag(id: number | string, tag: Tag) {
  const res = await api.put(`/tags/${id}`, { tag: tag });
  return res.data;
}

export async function getLogs(keyword: string, offset?: number, count?: number, level?: string) {
  const params: Record<string, unknown> = {};
  if (keyword) params.keyword = keyword;
  if (typeof offset === 'number') params.offset = offset;
  if (typeof count === 'number') params.count = count;
  if (level) params.level = level;

  const res = await api.get('/logs', { params });
  return res.data;
}

export async function getTagValueFloat(params: { offset?: number; count?: number; tag_id?: number } = {}) {
  const res = await api.get('/tagValueFloat', { params });
  return res.data as
    | TagValueFloat[]
    | { rows?: TagValueFloat[]; items?: TagValueFloat[]; meta?: { total?: number; totalCount?: number } };
}

export async function restartServer() {
  const res = await api.post('/restart', {});
  return res.data;
}

export default {
  login,
  validateJWT,
  getDevices,
  reloadDevices,
  updateDevice,
  getTags,
  reloadTags,
  updateTag,
  getLogs,
  restartServer,
};
