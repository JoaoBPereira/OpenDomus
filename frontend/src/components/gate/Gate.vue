<script setup lang="ts">
  import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue';

  import { getLatestTagValueFloat, getTagValueFloat } from '../../services/api';
  import Modal from '../ui/Modal.vue';
  import { TAG_GATE } from '../../constants/tagIds';

  const gateOpen = ref<{ value: number; timestamp: string }>();
  // Transitions are enabled only after the first reading, so the page doesn't load with the gate swinging.
  const animated = ref(false);
  const openTempDialog = ref(false);
  const dialogTitle = ref<string | undefined>();
  const lineData = ref<Array<{ value: number; date: Date }>>([]);
  const timeRange = ref(24);
  const currentTagId = ref('');

  let intervalId: number | undefined;

  const TZ = 'Europe/Lisbon';
  const dayKey = (d: Date) => d.toLocaleDateString('en-CA', { timeZone: TZ });

  const timelineGroups = computed(() => {
    const today = new Date();
    const yesterday = new Date();
    yesterday.setDate(yesterday.getDate() - 1);
    const todayKey = dayKey(today);
    const yesterdayKey = dayKey(yesterday);

    const groups: Array<{ key: string; label: string; items: Array<{ value: number; date: Date }> }> = [];
    [...lineData.value]
      .sort((a, b) => b.date.getTime() - a.date.getTime())
      .forEach((item) => {
        const key = dayKey(item.date);
        let group = groups[groups.length - 1];
        if (!group || group.key !== key) {
          const dateLabel = item.date.toLocaleDateString('en-US', { timeZone: TZ, month: 'long', day: 'numeric', year: 'numeric' });
          const prefix = key === todayKey ? 'Today' : key === yesterdayKey ? 'Yesterday' : item.date.toLocaleDateString('en-US', { timeZone: TZ, weekday: 'long' });
          group = { key, label: `${prefix} · ${dateLabel}`, items: [] };
          groups.push(group);
        }
        group.items.push(item);
      });
    return groups;
  });

  const formatTime = (d: Date) => d.toLocaleTimeString('en-US', { timeZone: TZ, hour: 'numeric', minute: '2-digit', second: '2-digit' });

  const isOpen = computed(() => gateOpen.value !== undefined && gateOpen.value.value !== 0);
  const statusText = computed(() => {
    if (!gateOpen.value) return '\u2014';
    return isOpen.value ? 'Open' : 'Closed';
  });
  const lastChange = computed(() => {
    if (!gateOpen.value) return 'Gate';
    return `Gate ${statusText.value.toLowerCase()} since ${new Date(gateOpen.value.timestamp).toLocaleString('pt-PT', { timeZone: TZ })}`;
  });

  const fetchData = () => {
    getLatestTagValueFloat(TAG_GATE).then(async (json) => {
      gateOpen.value = { value: json.value, timestamp: json.timestamp ?? json.time };
      if (!animated.value) {
        await nextTick();
        requestAnimationFrame(() => {
          animated.value = true;
        });
      }
    });
  };

  const openDialog = (tagId: string, title: string) => {
    currentTagId.value = tagId;
    fetchChartData(tagId, title, timeRange.value);
    dialogTitle.value = title;
    openTempDialog.value = true;
  };

  const fetchChartData = async (tagId: string, title: string, hours: number) => {
    const tempValues: Array<{ value: number; date: Date }> = [];
    const now = new Date();
    const hoursPrevious = new Date();
    hoursPrevious.setHours(hoursPrevious.getHours() - hours);

    const json = await getTagValueFloat(tagId, hoursPrevious.toISOString(), now.toISOString());
    if (json.length > 0) {
      json.forEach((element) => {
        tempValues.push({ value: element.value, date: new Date(element.time) });
      });
    }
    lineData.value = tempValues;

    if (tempValues.length > 0) {
      dialogTitle.value = `${title} - Last Update: ${tempValues[tempValues.length - 1].date.toLocaleString('pt-PT', { timeZone: 'Europe/Lisbon' })}`;
    }
  };

  const handleTimeRangeChange = () => {
    if (currentTagId.value) {
      fetchChartData(currentTagId.value, dialogTitle.value?.split(' - ')[0] || 'Gate', timeRange.value);
    }
  };

  const closeDialog = () => {
    openTempDialog.value = false;
  };

  onMounted(() => {
    fetchData();
    intervalId = window.setInterval(fetchData, 10000);
  });

  onBeforeUnmount(() => {
    if (intervalId) {
      clearInterval(intervalId);
    }
  });
</script>

<template>
  <div class="gateMain">
    <button type="button" class="gateCard" :class="{ open: isOpen, animated }" :title="lastChange" :aria-label="`${lastChange}. Show gate history`" @click="openDialog(TAG_GATE.toString(), 'Gate')">
      <h2>Gate</h2>

      <span class="display">
        <span class="scene">
          <span class="floor">
            <span class="road" />
            <span class="glow closedGlow" />
            <span class="glow openGlow" />
          </span>

          <span class="pillar left">
            <span class="lamp" />
            <span class="cap" />
          </span>
          <span class="pillar right">
            <span class="lamp" />
            <span class="cap" />
          </span>

          <span class="gate">
            <span class="leaf left" />
            <span class="leaf right" />
            <span class="lock">
              <svg viewBox="0 0 24 24" aria-hidden="true">
                <rect x="5" y="11" width="14" height="10" rx="2" />
                <path d="M8 11V8a4 4 0 0 1 8 0v3" />
              </svg>
            </span>
          </span>

          <span class="status">
            <span class="statusDot" />
            {{ statusText }}
          </span>
        </span>
      </span>
    </button>

    <Modal :open="openTempDialog" :title="dialogTitle" @close="closeDialog">
      <div class="gate-modal-body">
        <div style="padding-left: 1vw">
          <label class="gate-select-label">
            Time Range
            <select v-model.number="timeRange" @change="handleTimeRangeChange" class="gate-select">
              <option :value="24">24 Hours</option>
              <option :value="48">48 Hours</option>
              <option :value="168">7 Days</option>
              <option :value="720">1 Month</option>
            </select>
          </label>
        </div>
        <div class="timeline">
          <div v-for="group in timelineGroups" :key="group.key" class="timeline-group">
            <div class="timeline-day">{{ group.label }}</div>
            <div v-for="item in group.items" :key="item.date.toISOString()" class="timeline-item">
              <div class="timeline-time">{{ formatTime(item.date) }}</div>
              <div class="timeline-marker">
                <div class="timeline-icon" :class="item.value === 0 ? 'closed' : 'open'"><span /></div>
              </div>
              <div class="timeline-content">
                <div class="timeline-title">Door &rarr; {{ item.value === 0 ? 'Closed' : 'Open' }}</div>
                <div class="timeline-subtitle">Gate Sensor</div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </Modal>
  </div>
</template>

<style lang="scss" scoped>
  .gate-select-label {
    display: inline-flex;
    flex-direction: column;
    gap: 6px;
    color: #ffffff;
    font-size: 0.85rem;
  }

  .gate-select {
    background: #111111;
    color: #ffffff;
    border: 1px solid #444444;
    border-radius: 4px;
    padding: 6px 8px;
  }

  .gate-modal-body {
    display: flex;
    flex-direction: column;
    height: 100%;
  }

  .timeline {
    flex: 1;
    min-height: 0;
    padding: 12px 1vw;
    color: #ffffff;
    overflow-y: auto;
  }

  .timeline-group {
    margin-bottom: 16px;
  }

  .timeline-day {
    font-size: 0.95rem;
    color: #dddddd;
    margin-bottom: 12px;
  }

  .timeline-item {
    display: flex;
    align-items: flex-start;
    gap: 12px;
    position: relative;
    min-height: 56px;
  }

  .timeline-time {
    width: 90px;
    text-align: right;
    font-size: 0.8rem;
    color: #aaaaaa;
    padding-top: 10px;
  }

  .timeline-marker {
    position: relative;
    display: flex;
    justify-content: center;
    width: 36px;
    align-self: stretch;
  }

  .timeline-item:not(:last-child) .timeline-marker::after {
    content: '';
    position: absolute;
    top: 36px;
    bottom: 0;
    left: 50%;
    width: 1px;
    background: #444444;
  }

  .timeline-icon {
    width: 36px;
    height: 36px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;

    span {
      width: 14px;
      height: 14px;
      box-sizing: border-box;
    }

    &.closed {
      background: #3a3a3a;

      span {
        background: #d0d0d0;
      }
    }

    &.open {
      background: #4a3d10;

      span {
        border: 2px solid #f5c518;
      }
    }
  }

  .timeline-content {
    padding-top: 4px;
  }

  .timeline-title {
    font-size: 0.95rem;
  }

  .timeline-subtitle {
    font-size: 0.75rem;
    color: #aaaaaa;
    margin-top: 2px;
  }

  // ── Gate card ─────────────────────────────────────────────────────────
  $card-bg: #101013;
  $text: #f4f4f8;
  $closed: #6b9bff;
  $closed-lamp: #b3c0ff;
  $open: #f0a04b;
  $open-lamp: #f7c867;
  $stone: #2a2b31;
  $stone-light: #3c3d45;
  $swing: 1.6s cubic-bezier(0.45, 0, 0.25, 1);

  .gateMain {
    position: relative;
    width: 26vw;
    min-height: 0;

    @include mobile {
      width: 46vw;
    }
  }

  // Same padding and title style as the Power and Devices cards.
  .gateCard {
    display: flex;
    flex-direction: column;
    gap: 0.6vh;
    width: 100%;
    height: 100%;
    min-height: 0;
    margin: 0;
    padding: 1.2vh 1.2vw;
    border: 0.08vw solid #25252b;
    border-radius: 0.8vw;
    background: $card-bg;
    color: $text;
    font-family: 'Seven Segment', sans-serif;
    container-type: inline-size;
    box-sizing: border-box;
    cursor: pointer;

    @include mobile {
      padding: 1vh 3vw;
      border-width: 0.1vh;
      border-radius: 1vh;
    }

    &:focus-visible {
      outline: 0.15vw solid #ffffff;
      outline-offset: 0.2vw;
    }

    h2 {
      align-self: flex-start;
      flex-shrink: 0;
      margin: 0;
      font-size: 1.4vh;
      letter-spacing: 0.25em;
      text-transform: uppercase;
    }
  }

  // ── Scene ─────────────────────────────────────────────────────────────
  .display {
    display: flex;
    flex: 1;
    align-items: center;
    justify-content: center;
    width: 100%;
    min-height: 0;
    aspect-ratio: 480 / 286;
    container-type: size;
  }

  // Keep the complete scene inside the available space in both dimensions.
  .scene {
    position: relative;
    width: min(88cqw, calc(100cqh * 480 / 325));
    aspect-ratio: 480 / 325;
    container-type: inline-size;

    @include mobile {
      width: min(96cqw, calc(100cqh * 480 / 325));
    }
  }

  .floor {
    position: absolute;
    top: 80%;
    bottom: 0;
    left: 0.5%;
    right: 0.5%;
    overflow: hidden;
    background: #16171c;
    border-top: 0.1cqw solid #2c2d34;
  }

  .road {
    position: absolute;
    inset: 0;
    background: #1d1f27;
    clip-path: polygon(19% 0, 81% 0, 86% 100%, 14% 100%);
  }

  // Two glows cross-fade, since gradients can't transition between colours.
  .glow {
    position: absolute;
    inset: 0;
    transition: opacity $swing;
  }

  .closedGlow {
    background: radial-gradient(ellipse 45% 100% at 50% 0, rgba(107, 155, 255, 0.35), transparent);
  }

  .openGlow {
    background: radial-gradient(ellipse 45% 100% at 50% 0, rgba(240, 160, 75, 0.4), transparent);
    opacity: 0;
  }

  .open {
    .closedGlow {
      opacity: 0;
    }
    .openGlow {
      opacity: 1;
    }
  }

  // ── Pillars ───────────────────────────────────────────────────────────
  .pillar {
    position: absolute;
    top: 11%;
    bottom: 20%;
    width: 7.5%;
    border: 0.1cqw solid #55565f;
    background: linear-gradient(90deg, $stone-light, $stone 60%);
    box-sizing: border-box;

    &.left {
      left: 5%;
    }
    &.right {
      right: 5%;
    }

    // The joint line halfway down.
    &::after {
      content: '';
      position: absolute;
      top: 61%;
      left: 0;
      right: 0;
      border-top: 0.1cqw solid #55565f;
    }
  }

  .cap {
    position: absolute;
    bottom: 100%;
    left: -22%;
    right: -22%;
    height: 4.6%;
    min-height: 1.4cqw;
    border: 0.1cqw solid #6a6b75;
    border-radius: 0.3cqw;
    background: linear-gradient($stone-light, $stone);
  }

  .lamp {
    position: absolute;
    bottom: calc(100% + 1.6cqw);
    left: 22%;
    right: 22%;
    aspect-ratio: 1;
    border-radius: 50% 50% 15% 15%;
    background: $closed-lamp;
    box-shadow: 0 0 2.5cqw rgba(107, 155, 255, 0.7);
  }

  .open .lamp {
    background: $open-lamp;
    box-shadow: 0 0 2.5cqw rgba(240, 160, 75, 0.8);
  }

  // ── Gate leaves ───────────────────────────────────────────────────────
  .gate {
    position: absolute;
    top: 24%;
    bottom: 28%;
    left: 12.5%;
    right: 12.5%;
    // Weak perspective: an open leaf's free edge looks only slightly taller than its hinge.
    perspective: 300cqw;
  }

  .leaf {
    position: absolute;
    top: 0;
    bottom: 0;
    width: 50%;
    border: 0.45cqw solid $closed;
    background-color: rgba(28, 32, 58, 0.75);
    background-image: repeating-linear-gradient(90deg, transparent 0 9.5%, rgba(220, 226, 255, 0.6) 9.5% 11.5%);
    box-shadow:
      0 0 2cqw rgba(107, 155, 255, 0.45),
      inset 0 0 1.5cqw rgba(107, 155, 255, 0.25);
    box-sizing: border-box;

    // Hinged at the pillar; open leaves swing out towards the viewer.
    &.left {
      left: 0;
      border-radius: 18% 0 0 0 / 14% 0 0 0;
      transform-origin: left center;
    }
    &.right {
      right: 0;
      border-radius: 0 18% 0 0 / 0 14% 0 0;
      transform-origin: right center;
    }
  }

  .open .leaf {
    border-color: $open;
    background-color: rgba(58, 40, 18, 0.75);
    background-image: repeating-linear-gradient(90deg, transparent 0 9.5%, rgba(255, 236, 210, 0.6) 9.5% 11.5%);
    box-shadow:
      0 0 2cqw rgba(240, 160, 75, 0.5),
      inset 0 0 1.5cqw rgba(240, 160, 75, 0.25);

    &.left {
      transform: rotateY(-72deg);
    }
    &.right {
      transform: rotateY(72deg);
    }
  }

  .lock {
    position: absolute;
    top: 50%;
    left: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 10%;
    aspect-ratio: 1;
    border: 0.2cqw solid $closed;
    border-radius: 50%;
    background: #101013;
    transform: translate(-50%, -50%);

    svg {
      width: 50%;
      fill: none;
      stroke: #d6dcff;
      stroke-width: 2;
    }
  }

  .open .lock {
    opacity: 0;
    transform: translate(-50%, -50%) scale(0.6);
  }

  // ── Status pill ───────────────────────────────────────────────────────
  .status {
    position: absolute;
    top: 91%;
    left: 50%;
    display: flex;
    align-items: center;
    gap: 1cqw;
    padding: 0.6cqw 2cqw;
    border: 0.15cqw solid rgba(107, 155, 255, 0.7);
    border-radius: 3cqw;
    background: rgba(107, 155, 255, 0.12);
    color: $closed-lamp;
    font-size: 3.2cqw;
    letter-spacing: 0.2em;
    text-transform: uppercase;
    white-space: nowrap;
    transform: translate(-50%, -50%);

    @include mobile {
      font-size: 4.5cqw;
      letter-spacing: 0.1em;
    }
  }

  .statusDot {
    width: 1.3cqw;
    aspect-ratio: 1;
    border-radius: 50%;
    background: currentColor;
  }

  .open .status {
    border-color: rgba(240, 160, 75, 0.7);
    background: rgba(240, 160, 75, 0.12);
    color: $open;
  }

  // ── Transitions ───────────────────────────────────────────────────────
  // Leaves swing while colours fade; the lock fades out first when opening.
  .animated {
    .leaf {
      transition:
        transform $swing,
        border-color $swing,
        background-color $swing,
        box-shadow $swing;
    }

    .lamp,
    .status {
      transition:
        background $swing,
        box-shadow $swing,
        border-color $swing,
        color $swing;
    }

    .lock {
      transition:
        opacity 0.4s ease,
        transform 0.4s ease;
    }
  }

  // Closing: the lock comes back only once the leaves have met.
  .animated:not(.open) .lock {
    transition-delay: 1.3s;
  }
</style>
