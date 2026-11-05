<!-- eslint-disable @typescript-eslint/no-explicit-any -->
<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { Notify } from 'quasar';
import { login as svcLogin, validateJWT as svcValidateJWT } from '../../services';

const username = ref('');
const password = ref('');
const loading = ref(false);
const router = useRouter();

const login = async () => {
  loading.value = true;
  try {
    const data = await svcLogin(username.value, password.value);

    if (!data?.token) {
      throw new Error('No authentication token received');
    }

    localStorage.setItem('jwt', data.token);

    // Navigate to main layout route on success
    await router.push('/main');
  } catch (err: any) {
    console.error('Login error:', err);
    Notify.create({
      type: 'negative',
      message: 'Login failed. Please check your credentials. ' + (err?.message || String(err)),
    });
  } finally {
    loading.value = false;
  }
};

const checkJWT = async () => {
  loading.value = true;
  try {
    const valid = await svcValidateJWT();
    if (valid) await router.push('/main');
  } catch (err: any) {
    console.error('No valid JWT', err);
    Notify.create({
      type: 'negative',
      message: 'Failed to load devices: ' + (err?.message || String(err)),
    });
  } finally {
    loading.value = false;
  }
};

onMounted(() => {
  void checkJWT();
});
</script>

<template>
  <q-layout view="lHh Lpr lFf"
    ><q-page-container>
      <q-page class="flex flex-center bg-grey-2">
        <q-card class="q-pa-md shadow-2 my_card" bordered>
          <q-card-section class="text-center">
            <div class="text-grey-9 text-h5 text-weight-bold">Sign in</div>
            <div class="text-grey-8">Sign in below to access your account</div>
          </q-card-section>
          <q-card-section>
            <q-input dense outlined v-model="username" label="Username"></q-input>
            <q-input
              dense
              outlined
              class="q-mt-md"
              v-model="password"
              type="password"
              label="Password"
            ></q-input>
          </q-card-section>
          <q-card-section>
            <q-btn
              style="border-radius: 8px"
              color="dark"
              rounded
              size="md"
              label="Sign in"
              no-caps
              class="full-width"
              :loading="loading"
              :disable="loading"
              @click="login"
            ></q-btn>
          </q-card-section>
          <q-card-section class="text-center q-pt-none">
            <div class="text-grey-8">
              Don't have an account yet?
              <a href="#" class="text-dark text-weight-bold" style="text-decoration: none">Login</a>
            </div>
          </q-card-section>
        </q-card>
      </q-page>
    </q-page-container></q-layout
  >
</template>
