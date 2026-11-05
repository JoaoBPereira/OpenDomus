<script setup lang="ts">
  import { computed, onMounted, onUnmounted, ref } from 'vue'
  import { useRouter } from 'vue-router'

  import { login as apiLogin } from '../services/api'
  import { clearJwt, getJwt, isJwtExpired } from '../utils'

  const router = useRouter()

  const username = ref('')
  const password = ref('')
  const showPassword = ref(false)
  const keepSignedIn = ref(false)
  const loading = ref(false)
  const checking = ref(true)
  const error = ref<string | null>(null)

  const now = ref(new Date())
  let clockTimer: ReturnType<typeof setInterval> | undefined

  const clockTime = computed(() => now.value.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' }))
  const clockDate = computed(() => {
    const weekday = now.value.toLocaleDateString('en-GB', { weekday: 'long' })
    const dayMonth = now.value.toLocaleDateString('en-GB', { day: 'numeric', month: 'short' })
    return `${weekday} · ${dayMonth}`
  })

  const handleSubmit = async () => {
    loading.value = true
    error.value = null

    try {
      const response = await apiLogin(username.value, password.value)
      if (!response.ok) {
        throw new Error('Login failed. Please check your credentials.')
      }

      const data = await response.json()
      if (!data.token) {
        throw new Error('No authentication token received')
      }

      clearJwt()
      const storage = keepSignedIn.value ? localStorage : sessionStorage
      storage.setItem('jwt', data.token)
      router.push('/main')
    } catch (err) {
      console.error('Login error:', err)
      error.value = err instanceof Error ? err.message : 'An unknown error occurred'
    } finally {
      loading.value = false
    }
  }

  onMounted(() => {
    clockTimer = setInterval(() => (now.value = new Date()), 1000)

    const oldJWT = getJwt()
    if (oldJWT && !isJwtExpired(oldJWT)) {
      router.push('/main')
      return
    }

    if (oldJWT && isJwtExpired(oldJWT)) {
      clearJwt()
    }

    setTimeout(() => {
      checking.value = false
    }, 500)
  })

  onUnmounted(() => clearInterval(clockTimer))
</script>

<template>
  <div class="login-page">
    <div v-if="checking">
      <div class="spinner" aria-label="Checking session"></div>
    </div>
    <div v-else class="login-content">
      <div class="clock">
        <div class="clock-time">{{ clockTime }}</div>
        <div class="clock-date">{{ clockDate }}</div>
      </div>

      <div class="card">
        <div class="card-header">
          <svg class="home-icon" viewBox="-30 -26 60 52" aria-hidden="true">
            <path d="M -26 4 L 0 -22 L 26 4 L 26 22 L -26 22 Z" fill="none" stroke="currentColor" stroke-width="3" stroke-linejoin="round" />
            <path class="bolt" d="M -2 -4 L -7 6 L -1 6 L -5 14 L 7 2 L 1 2 L 5 -4 Z" />
          </svg>
          <div>
            <h1>Home</h1>
            <p>Sign in to your dashboard</p>
          </div>
        </div>

        <div v-if="error" class="error-message">{{ error }}</div>

        <form @submit.prevent="handleSubmit">
          <label class="field">
            <span class="field-label">Username</span>
            <input v-model="username" type="text" placeholder="username" autocomplete="username" required />
          </label>

          <label class="field">
            <span class="field-label">Password</span>
            <span class="password-wrapper">
              <input v-model="password" :type="showPassword ? 'text' : 'password'" placeholder="••••••••" autocomplete="current-password" required />
              <button type="button" class="eye-button" :aria-label="showPassword ? 'Hide password' : 'Show password'" @click="showPassword = !showPassword">
                <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" aria-hidden="true">
                  <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7S2 12 2 12z" />
                  <circle cx="12" cy="12" r="3" />
                  <path v-if="showPassword" d="M4 4l16 16" />
                </svg>
              </button>
            </span>
          </label>

          <label class="keep-signed-in">
            <input v-model="keepSignedIn" type="checkbox" />
            Keep me signed in
          </label>

          <button type="submit" class="submit-button" :disabled="loading">
            {{ loading ? 'Signing in…' : 'Sign in' }}
          </button>
        </form>
      </div>

      <p class="footer-hint">Forgot your password? Ask the home admin.</p>
    </div>
  </div>
</template>

<style lang="scss" scoped>
  $teal: #7ad79c;
  $background: #050709;
  $card: #101114;
  $input: #16181c;
  $border: #25272d;
  $muted: #8c93a6;
  $mono: 'JetBrains Mono', 'Fira Code', Consolas, 'Courier New', monospace;

  .login-page {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 32px 16px;
    box-sizing: border-box;
    font-family: $mono;
    color: #e6e8ee;
    background:
      radial-gradient(circle at 15% 10%, rgba($teal, 0.07), transparent 40%),
      radial-gradient(circle at 90% 85%, rgba(226, 140, 60, 0.06), transparent 35%),
      $background;
  }

  .login-content {
    width: 100%;
    max-width: 500px;
  }

  .clock {
    text-align: center;
    margin-bottom: 36px;
  }

  .clock-time {
    font-family: 'Seven Segment', $mono;
    font-size: 84px;
    line-height: 1;
    letter-spacing: 4px;
    color: $teal;
    text-shadow: 0 0 18px rgba($teal, 0.45);
  }

  .clock-date {
    margin-top: 14px;
    font-size: 13px;
    letter-spacing: 4px;
    text-transform: uppercase;
    color: $muted;
  }

  .card {
    background: $card;
    border: 1px solid $border;
    border-radius: 14px;
    padding: 36px;
    box-shadow: 0 20px 60px rgba(0, 0, 0, 0.5);

    @media (max-width: 600px) {
      padding: 24px;
    }
  }

  .card-header {
    display: flex;
    align-items: center;
    gap: 20px;
    margin-bottom: 28px;

    h1 {
      margin: 0 0 4px;
      font-size: 17px;
      letter-spacing: 4px;
      text-transform: uppercase;
      color: #ffffff;
    }

    p {
      margin: 0;
      font-size: 14px;
      color: $muted;
    }
  }

  .home-icon {
    width: 40px;
    height: 36px;
    color: #ffffff;
    flex-shrink: 0;

    .bolt {
      fill: $teal;
    }
  }

  .error-message {
    background: rgba(248, 113, 113, 0.1);
    border: 1px solid rgba(248, 113, 113, 0.5);
    color: #fca5a5;
    padding: 12px 16px;
    border-radius: 8px;
    margin-bottom: 20px;
    font-size: 13px;
  }

  .field {
    display: block;
    margin-bottom: 26px;
  }

  .field-label {
    display: block;
    margin-bottom: 10px;
    font-size: 12px;
    letter-spacing: 2px;
    text-transform: uppercase;
    color: $muted;
  }

  .field input {
    width: 100%;
    box-sizing: border-box;
    padding: 16px 18px;
    background: $input;
    border: 1px solid $border;
    border-radius: 8px;
    color: #e6e8ee;
    font: inherit;
    font-size: 15px;
    outline: none;
    transition:
      border-color 0.15s,
      box-shadow 0.15s;

    &::placeholder {
      color: #555a66;
    }

    &:focus {
      border-color: $teal;
      box-shadow: 0 0 0 3px rgba($teal, 0.15);
    }
  }

  .password-wrapper {
    position: relative;
    display: block;

    input {
      padding-right: 52px;
    }
  }

  .eye-button {
    position: absolute;
    top: 50%;
    right: 12px;
    transform: translateY(-50%);
    width: 32px;
    height: 32px;
    padding: 6px;
    background: none;
    border: none;
    color: $muted;
    cursor: pointer;

    &:hover,
    &:focus-visible {
      color: $teal;
    }

    svg {
      width: 100%;
      height: 100%;
    }
  }

  .keep-signed-in {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 28px;
    font-size: 14px;
    cursor: pointer;

    input {
      width: 18px;
      height: 18px;
      margin: 0;
      accent-color: $teal;
      cursor: pointer;
    }
  }

  .submit-button {
    width: 100%;
    padding: 16px;
    background: $teal;
    border: none;
    border-radius: 8px;
    color: #06120b;
    font: inherit;
    font-size: 15px;
    font-weight: 700;
    letter-spacing: 6px;
    text-transform: uppercase;
    cursor: pointer;
    box-shadow: 0 0 24px rgba($teal, 0.3);
    transition:
      box-shadow 0.15s,
      opacity 0.15s;

    &:hover {
      box-shadow: 0 0 32px rgba($teal, 0.5);
    }

    &:disabled {
      opacity: 0.6;
      cursor: not-allowed;
    }
  }

  .footer-hint {
    margin-top: 40px;
    text-align: center;
    font-size: 13px;
    color: #5c616d;
  }

  .spinner {
    width: 72px;
    height: 72px;
    border: 6px solid rgba(255, 255, 255, 0.2);
    border-top-color: #ffffff;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
    margin: 40px auto;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>
