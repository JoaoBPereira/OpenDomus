<script setup lang="ts">
  import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
  import router from '../../router';
  import { getLatestTagValueFloat, getServerVersion } from '../../services/api';
  import { clearJwt, getJwt } from '../../utils';
  import { firstRooms, groundRooms, temperatureColor } from '../temperature/temperatureData';
  import SunArc from './SunArc.vue';

  // Same scale as the floor plans, so the average reads like a room tile.
  const MIN_TEMP = 16;
  const MAX_TEMP = 26;
  const EMPTY = '—';

  // Kitchen shares the Living Room sensor and still counts as its own room.
  const sensorRooms = [...groundRooms, ...firstRooms].filter((room) => room.tagId);

  const now = ref(new Date());
  const roomValues = ref<number[]>([]);
  const menuOpen = ref(false);
  const serverVersion = ref(EMPTY);
  const menuRoot = ref<HTMLElement>();

  const time = computed(() => now.value.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' }));
  const date = computed(() => {
    const weekday = now.value.toLocaleDateString('en-GB', { weekday: 'short' });
    const dayMonth = now.value.toLocaleDateString('en-GB', { day: 'numeric', month: 'short' });
    return `${weekday} ${dayMonth}`;
  });

  const average = computed(() => {
    const values = roomValues.value;
    return values.length ? values.reduce((sum, value) => sum + value, 0) / values.length : undefined;
  });
  const averageText = computed(() => (average.value === undefined ? EMPTY : average.value.toFixed(1)));
  const averageStyle = computed(() => (average.value === undefined ? {} : { color: temperatureColor(average.value, MIN_TEMP, MAX_TEMP) }));

  const username = computed(() => {
    try {
      return JSON.parse(atob(getJwt()!.split('.')[1])).username ?? EMPTY;
    } catch {
      return EMPTY;
    }
  });

  let clockTimer: number | undefined;
  let fetchTimer: number | undefined;

  // Rooms that fail to load are left out of the average instead of blanking it.
  const fetchAverage = async () => {
    const results = await Promise.allSettled(sensorRooms.map((room) => getLatestTagValueFloat(room.tagId!)));
    roomValues.value = results.flatMap((result) => (result.status === 'fulfilled' && Number.isFinite(result.value.value) ? [result.value.value] : []));
  };

  async function toggleMenu() {
    menuOpen.value = !menuOpen.value;
    if (menuOpen.value) {
      serverVersion.value = await getServerVersion().catch(() => EMPTY);
    }
  }

  const initials = computed(() => username.value.slice(0, 3).toUpperCase());

  function logOff() {
    clearJwt();
    router.push('/login');
  }

  function onDocumentClick(event: MouseEvent) {
    if (!menuRoot.value?.contains(event.target as Node)) menuOpen.value = false;
  }

  function onKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') menuOpen.value = false;
  }

  onMounted(() => {
    fetchAverage();
    fetchTimer = window.setInterval(fetchAverage, 60000);
    clockTimer = window.setInterval(() => (now.value = new Date()), 1000);
    document.addEventListener('click', onDocumentClick);
    document.addEventListener('keydown', onKeydown);
  });

  onBeforeUnmount(() => {
    clearInterval(fetchTimer);
    clearInterval(clockTimer);
    document.removeEventListener('click', onDocumentClick);
    document.removeEventListener('keydown', onKeydown);
  });
</script>

<template>
  <header class="topBar">
    <div class="left">
      <div class="clock">
        <span class="time">{{ time }}</span>
        <span class="label">{{ date }}</span>
      </div>
      <span class="divider" aria-hidden="true"></span>
      <SunArc :now="now" />
    </div>

    <div class="houseAverage" :aria-label="`House average ${averageText} degrees Celsius`">
      <svg class="houseIcon" viewBox="0 0 24 24" aria-hidden="true">
        <path d="M3 11 12 3l9 8M5 9.5V21h14V9.5" />
        <path d="M12 10v6.5M10.6 17.6a1.6 1.6 0 1 0 2.8 0" />
      </svg>
      <span class="label">House avg</span>
      <span class="average" :style="averageStyle">{{ averageText }}<sup v-if="average !== undefined">&deg;</sup></span>
    </div>

    <div ref="menuRoot" class="menu">
      <button type="button" class="gear" aria-label="Settings" :aria-expanded="menuOpen" @click="toggleMenu">
        <svg viewBox="0 0 24 24" aria-hidden="true">
          <circle cx="12" cy="12" r="3" />
          <path
            d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"
          />
        </svg>
      </button>

      <div v-if="menuOpen" class="menuOverlay" @click.self="menuOpen = false">
        <div class="menuPanel" role="dialog" aria-label="Menu">
          <div class="menuHeader">
            <span class="menuTitle">Menu</span>
            <button type="button" class="menuClose" aria-label="Close menu" @click="menuOpen = false">
              <svg viewBox="0 0 24 24" aria-hidden="true"><path d="M6 6l12 12M18 6 6 18" /></svg>
            </button>
          </div>

          <div class="menuCard">
            <span class="avatar" aria-hidden="true">{{ initials }}</span>
            <span class="userText">
              <span class="cardLabel">Signed in as</span>
              <span class="userName">{{ username }}</span>
            </span>
          </div>

          <div class="menuCard">
            <span class="cardLabel">Server</span>
            <span class="version">v{{ serverVersion }}</span>
          </div>

          <button type="button" class="logOff" @click="logOff">Log off</button>
        </div>
      </div>
    </div>
  </header>
</template>

<style lang="scss" scoped>
  $bar-bg: #101013;
  $border: #25252b;
  $text: #f4f4f8;
  $muted: #8c93a6;
  $mono: 'JetBrains Mono', 'Fira Code', Consolas, 'Courier New', monospace;

  .topBar {
    display: grid;
    grid-template-columns: 1fr auto 1fr;
    align-items: center;
    margin: 0 0.5vw;
    padding: 0vh 0.4vh 0vh 1.2vw;
    border: 0.08vw solid $border;
    border-radius: 0.8vw;
    background: $bar-bg;
    color: $text;
    user-select: none;
    container-type: inline-size;

    @include mobile {
      border-width: 0.1vh;
      border-radius: 1vh;
    }
  }

  .label {
    color: $muted;
    font-family: $mono;
    font-size: 0.6cqw;
    letter-spacing: 0.25em;
    text-transform: uppercase;
  }

  .clock,
  .houseAverage {
    display: flex;
    align-items: baseline;
    gap: 0.6vw;

    @include mobile {
      gap: 1.5vw;
    }
  }

  .time,
  .average {
    font-family: 'Seven Segment', sans-serif;
    font-size: 2.7cqw;
    line-height: 1;

    @include mobile {
      font-size: 6cqw;
    }
    letter-spacing: 0.05em;
    white-space: nowrap;
  }

  .average sup {
    vertical-align: top;
    font-size: 0.45em;
  }

  .houseAverage {
    align-items: center;
  }

  .left {
    display: flex;
    align-items: center;
    gap: 1.2cqw;
  }

  .divider {
    align-self: stretch;
    width: 0.08vw;
    margin: 0.8vh 0;
    background: $border;
  }

  // The phone bar only has room for the clock.
  @include mobile {
    .divider,
    .left :deep(.sun) {
      display: none;
    }
  }

  .houseIcon {
    width: 1.7vh;
    height: 1.7vh;
    fill: none;
    stroke: $text;
    stroke-width: 1.8;
    stroke-linecap: round;
    stroke-linejoin: round;
  }

  // Too tight for the date and label beside the readings on phones.
  @include mobile {
    .clock .label,
    .houseAverage .label {
      display: none;
    }
  }

  .menu {
    position: relative;
    justify-self: end;
  }

  .gear {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 3.2vh;
    aspect-ratio: 1;
    padding: 0;
    border: 0.08vw solid $border;
    border-radius: 0.5vw;
    background: #16171b;
    cursor: pointer;

    svg {
      width: 2.2vh;
      height: 2.2vh;
      fill: none;
      stroke: $text;
      stroke-width: 1.6;
    }

    &:hover {
      box-shadow: inset 0 0 2vw #ffffff12;
    }

    &:focus-visible {
      outline: 0.15vw solid #ffffff;
      outline-offset: 0.15vw;
    }

    @include mobile {
      border-width: 0.1vh;
      border-radius: 1vh;
    }
  }

  $card-bg: #1a1b1f;
  $teal: #7ad79c;
  $orange: #e8935a;

  .menuOverlay {
    position: fixed;
    inset: 0;
    z-index: 10;
    display: flex;
    align-items: center;
    justify-content: center;
    background: #000000a6;
  }

  .menuPanel {
    display: flex;
    flex-direction: column;
    gap: 1.6vh;
    width: 20vw;
    padding: 2.2vh 1.2vw;
    border: 0.08vw solid $border;
    border-radius: 0.8vw;
    background: #131317;
    box-shadow: 0 1vh 4vh #000000b3;
    font-family: $mono;

    @include mobile {
      width: 85vw;
      padding: 2vh 4vw;
      border-width: 0.1vh;
      border-radius: 1.5vh;
    }
  }

  .menuHeader {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .menuTitle {
    font-size: 0.7cqw;
    font-weight: 700;
    letter-spacing: 0.35em;
    text-transform: uppercase;

    @include mobile {
      font-size: 3.5cqw;
    }
  }

  .menuClose {
    display: flex;
    padding: 0.4vh;
    border: 0;
    background: none;
    cursor: pointer;

    svg {
      width: 2.2vh;
      height: 2.2vh;
      fill: none;
      stroke: $muted;
      stroke-width: 1.8;
      stroke-linecap: round;
    }

    &:hover svg {
      stroke: $text;
    }
  }

  .menuCard {
    display: flex;
    align-items: center;
    gap: 1vw;
    padding: 1.6vh 0.9vw;
    border-radius: 0.5vw;
    background: $card-bg;

    @include mobile {
      gap: 3vw;
      padding: 1.6vh 3vw;
      border-radius: 1vh;
    }
  }

  .avatar {
    display: flex;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    height: 4.6vh;
    aspect-ratio: 1;
    border: 0.12vw solid $teal;
    border-radius: 50%;
    background: #7ad79c1a;
    color: $teal;
    font-family: 'Seven Segment', $mono;
    font-size: 0.95cqw;

    @include mobile {
      border-width: 0.2vh;
      font-size: 4cqw;
    }
  }

  .userText {
    display: flex;
    flex-direction: column;
    gap: 0.5vh;
    min-width: 0;
  }

  .cardLabel {
    color: $muted;
    font-size: 0.48cqw;
    letter-spacing: 0.2em;
    text-transform: uppercase;

    @include mobile {
      font-size: 2.4cqw;
    }
  }

  .userName {
    overflow: hidden;
    font-size: 0.95cqw;
    font-weight: 700;
    text-overflow: ellipsis;
    white-space: nowrap;

    @include mobile {
      font-size: 4.5cqw;
    }
  }

  .version {
    font-family: 'Seven Segment', $mono;
    font-size: 1cqw;
    letter-spacing: 0.05em;

    @include mobile {
      font-size: 4.5cqw;
    }
  }

  .logOff {
    min-height: 4.6vh;
    margin-top: 0.6vh;
    border: 0.08vw solid $orange;
    border-radius: 0.5vw;
    background: #2a1a14;
    color: $orange;
    font: inherit;
    font-size: 0.65cqw;
    font-weight: 700;
    letter-spacing: 0.35em;
    text-transform: uppercase;
    cursor: pointer;

    &:hover {
      background: #3a2219;
    }

    &:focus-visible {
      outline: 0.15vw solid #ffffff;
      outline-offset: 0.15vw;
    }

    @include mobile {
      border-width: 0.1vh;
      border-radius: 1vh;
      font-size: 3.2cqw;
    }
  }
</style>
