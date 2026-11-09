<template>
  <div v-if="open" class="modal-overlay" @click.self="emit('close')" @click.stop>
    <div class="modal-content" :style="contentStyle">
      <div class="modal-header">
        <h3 class="modal-title">{{ title }}</h3>
        <button class="modal-close" type="button" @click="emit('close')">x</button>
      </div>
      <div class="modal-body">
        <slot />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'

  const props = defineProps<{
    open: boolean
    title?: string
    width?: string
    height?: string
  }>()

  const emit = defineEmits<{
    (e: 'close'): void
  }>()

  const contentStyle = computed(() => ({
    width: props.width ?? '80%',
    height: props.height ?? '80%',
    maxWidth: 'none',
  }))
</script>

<style>
  .modal-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal-content {
    background: #111111;
    color: #ffffff;
    border-radius: 8px;
    padding: 16px;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.4);
  }

  .modal-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 12px;
  }

  .modal-title {
    margin: 0;
    font-size: 1.1rem;
  }

  .modal-close {
    background: transparent;
    border: 1px solid #444444;
    color: #ffffff;
    width: 28px;
    height: 28px;
    border-radius: 4px;
    cursor: pointer;
  }

  .modal-body {
    width: 100%;
    height: 90%;
  }
</style>
