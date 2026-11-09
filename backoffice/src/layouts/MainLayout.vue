<!-- eslint-disable @typescript-eslint/no-explicit-any -->
<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import EssentialLink, { type EssentialLinkProps } from '@/components/EssentialLink.vue';
import { Notify } from 'quasar';
import { validateJWT as svcValidateJWT } from '../../services';
import packageJson from '../../package.json';

const leftDrawerOpen = ref(false);
const router = useRouter();

const linksList: EssentialLinkProps[] = [
  {
    title: 'Main',
    caption: 'Main page',
    icon: 'school',
    link: '/main',
  },
  {
    title: 'Devices',
    caption: 'Devices',
    icon: 'school',
    link: '/devices',
  },
  {
    title: 'Tags',
    caption: 'Tags',
    icon: 'label',
    link: '/tags',
  },
  {
    title: 'Tag Values Float',
    caption: 'Tag Float Values',
    icon: 'show_chart',
    link: '/tag-value-float',
  },
  {
    title: 'Automations Time',
    caption: 'Automations started by a timer',
    icon: 'schedule',
    link: '/automations-time',
  },
  {
    title: 'Automations Values',
    caption: 'Automations started by a tag value',
    icon: 'sensors',
    link: '/automations-values',
  },

  {
    title: 'Logs',
    caption: 'Logs',
    icon: 'school',
    link: '/logs',
  },
];

function toggleLeftDrawer() {
  leftDrawerOpen.value = !leftDrawerOpen.value;
}

const checkJWT = async () => {
  try {
    await svcValidateJWT();
  } catch (err: any) {
    console.error('No valid JWT', err);
    Notify.create({
      type: 'negative',
      message: 'Failed to load devices: ' + (err?.message || String(err)),
    });
  }
};

onMounted(() => {
  void checkJWT();
});

function logout() {
  localStorage.removeItem('jwt');
  void router.push('/');
}
</script>

<template>
  <q-layout view="lHh Lpr lFf">
    <q-header elevated>
      <q-toolbar>
        <q-btn flat dense round icon="menu" aria-label="Menu" @click="toggleLeftDrawer" />

        <q-toolbar-title> OpenDomus - BackOffice </q-toolbar-title>

        <div>v{{ packageJson.version }}</div>
        <q-btn flat dense icon="logout" label="Logout" class="q-ml-md" @click="logout" />
      </q-toolbar>
    </q-header>

    <q-drawer v-model="leftDrawerOpen" show-if-above bordered>
      <q-list>
        <q-item-label header> Essential Links </q-item-label>

        <EssentialLink v-for="link in linksList" :key="link.title" v-bind="link" />
      </q-list>
    </q-drawer>

    <q-page-container>
      <router-view />
    </q-page-container>
  </q-layout>
</template>
