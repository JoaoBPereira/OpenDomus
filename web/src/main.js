import { createApp, computed, onMounted, ref } from 'vue'
import './style.css'

const App = {
  setup() {
    const tags = ref([])
    const readings = ref([])
    const events = ref([])
    const selectedTag = ref('')
    const source = ref('manual')
    const tagName = ref('')
    const tagValue = ref('')
    const automationTag = ref('')
    const automationOperator = ref('gt')
    const automationValue = ref('')
    const automationMessage = ref('')
    const error = ref('')
    const busy = ref(false)

    const numericReadings = computed(() => readings.value
      .map((reading) => ({
        time: new Date(reading.timestamp).getTime(),
        value: Number(reading.tags[selectedTag.value]),
      }))
      .filter((point) => Number.isFinite(point.time) && Number.isFinite(point.value)))
    const chartPoints = computed(() => {
      const points = numericReadings.value
      if (points.length < 2) return ''
      const values = points.map((point) => point.value)
      const min = Math.min(...values)
      const max = Math.max(...values)
      const span = max - min || 1
      const first = points[0].time
      const duration = points.at(-1).time - first || 1
      return points.map((point, index) => {
        const x = 20 + ((point.time - first) / duration) * 560
        const y = 160 - ((point.value - min) / span) * 130
        return `${index === 0 ? 'M' : 'L'} ${x} ${y}`
      }).join(' ')
    })

    async function api(path, options) {
      const response = await fetch(path, options)
      const payload = await response.json()
      if (!response.ok) throw new Error(payload.error || `Request failed (${response.status})`)
      return payload
    }

    async function refresh() {
      try {
        const [latest, history, recentEvents] = await Promise.all([
          api('/api/tags'),
          api(`/api/readings?limit=500${selectedTag.value ? `&tag=${encodeURIComponent(selectedTag.value)}` : ''}`),
          api('/api/events'),
        ])
        tags.value = latest
        readings.value = history
        events.value = recentEvents.slice(-8).reverse()
        if (!selectedTag.value && latest.length) selectedTag.value = latest[0].name
      } catch (problem) {
        error.value = problem.message
      }
    }

    async function addReading() {
      busy.value = true
      error.value = ''
      try {
        const value = Number(tagValue.value)
        if (!tagName.value.trim() || !Number.isFinite(value)) throw new Error('Enter a tag and a numeric reading.')
        await api('/api/readings', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ source: source.value, tags: { [tagName.value.trim()]: value } }),
        })
        selectedTag.value = tagName.value.trim()
        tagValue.value = ''
        await refresh()
      } catch (problem) {
        error.value = problem.message
      } finally {
        busy.value = false
      }
    }

    async function addAutomation() {
      busy.value = true
      error.value = ''
      try {
        const value = Number(automationValue.value)
        if (!automationTag.value || !Number.isFinite(value) || !automationMessage.value.trim()) {
          throw new Error('Choose a tag, numeric threshold and action message.')
        }
        await api('/api/automations', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            id: crypto.randomUUID(),
            tag: automationTag.value,
            operator: automationOperator.value,
            value,
            message: automationMessage.value.trim(),
          }),
        })
        automationValue.value = ''
        automationMessage.value = ''
        await refresh()
      } catch (problem) {
        error.value = problem.message
      } finally {
        busy.value = false
      }
    }

    onMounted(refresh)

    return {
      tags, events, selectedTag, source, tagName, tagValue,
      automationTag, automationOperator, automationValue, automationMessage,
      error, busy, chartPoints, numericReadings, addReading, addAutomation, refresh,
    }
  },
  template: `
    <main class="shell">
      <header class="topbar">
        <a class="brand" href="/" aria-label="OpenDomus home"><span class="brand-mark">O</span> OpenDomus</a>
        <span class="status"><i></i> Local hub</span>
      </header>
      <section class="intro">
        <div>
          <p class="eyebrow">YOUR HOME, WIRED TOGETHER</p>
          <h1>One place for<br /><em>everything at home.</em></h1>
          <p class="lede">Bring device readings together as tags, then automate, chart and query them.</p>
        </div>
        <div class="intro-orbit" aria-hidden="true"><span>HOME</span><i>⌂</i><b>·</b></div>
      </section>
      <p v-if="error" class="alert" role="alert">{{ error }}</p>
      <section class="section">
        <div class="section-heading"><div><p class="eyebrow">LIVE SNAPSHOT</p><h2>Your tags</h2></div><button class="quiet" @click="refresh">Refresh ↻</button></div>
        <div v-if="tags.length" class="tag-grid">
          <button v-for="tag in tags" :key="tag.name" class="tag-card" :class="{ active: selectedTag === tag.name }" @click="selectedTag = tag.name; refresh()">
            <span class="tag-name">{{ tag.name }}</span><strong>{{ tag.value }}</strong>
            <span class="tag-meta">{{ tag.source }} <span>·</span> {{ new Date(tag.timestamp).toLocaleTimeString() }}</span>
          </button>
        </div>
        <div v-else class="empty">No readings yet. Add a tag below or send data to an integration endpoint.</div>
      </section>
      <div class="columns">
        <section class="panel chart-panel">
          <div class="section-heading"><div><p class="eyebrow">HISTORY</p><h2>Tag trends</h2></div>
            <select v-model="selectedTag" @change="refresh"><option v-for="tag in tags" :key="tag.name" :value="tag.name">{{ tag.name }}</option></select>
          </div>
          <div v-if="chartPoints" class="chart">
            <svg viewBox="0 0 600 180" role="img" :aria-label="selectedTag + ' reading history'">
              <path class="chart-grid" d="M20 30H580 M20 95H580 M20 160H580" />
              <path class="chart-line" :d="chartPoints" />
            </svg>
            <div class="chart-caption"><span>{{ numericReadings.length }} readings</span><span>{{ selectedTag }}</span></div>
          </div>
          <div v-else class="empty chart-empty">Add a couple of numeric readings to see a trend.</div>
        </section>
        <section class="panel form-panel">
          <p class="eyebrow">QUICK INGEST</p><h2>Add a reading</h2>
          <form @submit.prevent="addReading">
            <label>Source<input v-model="source" placeholder="Shelly, Sonoff, Renault…" required /></label>
            <div class="form-row"><label>Tag<input v-model="tagName" placeholder="living-room-temp" required /></label><label>Value<input v-model="tagValue" type="number" step="any" placeholder="21.5" required /></label></div>
            <button class="primary" :disabled="busy">Add reading <span>↗</span></button>
          </form>
        </section>
      </div>
      <div class="columns lower">
        <section class="panel form-panel">
          <p class="eyebrow">AUTOMATIONS</p><h2>Make a rule</h2>
          <form @submit.prevent="addAutomation">
            <label>When tag<select v-model="automationTag" required><option disabled value="">Select a tag</option><option v-for="tag in tags" :key="tag.name" :value="tag.name">{{ tag.name }}</option></select></label>
            <div class="form-row"><label>Condition<select v-model="automationOperator"><option value="gt">greater than</option><option value="gte">at least</option><option value="lt">less than</option><option value="lte">at most</option><option value="eq">equals</option></select></label><label>Value<input v-model="automationValue" type="number" step="any" required /></label></div>
            <label>Action message<input v-model="automationMessage" placeholder="It's getting warm" required /></label>
            <button class="primary" :disabled="busy || !tags.length">Save automation <span>↗</span></button>
          </form>
        </section>
        <section class="panel">
          <p class="eyebrow">RECENT ACTIVITY</p><h2>Triggered rules</h2>
          <div v-if="events.length" class="event-list"><div v-for="(event, index) in events" :key="event.automationId + event.timestamp + index" class="event">
            <span class="event-dot"></span><div><strong>{{ event.message }}</strong><small>{{ new Date(event.timestamp).toLocaleString() }}</small></div>
          </div></div>
          <div v-else class="empty">Automation events will show up here.</div>
        </section>
      </div>
      <footer><span>OpenDomus · Open home automation</span><span>Read-only MCP queries at <code>/mcp</code></span></footer>
    </main>`,
}

createApp(App).mount('#app')
