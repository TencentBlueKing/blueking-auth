<template>
  <div class="page-wrapper">
    <template v-if="action">
      <Success
        v-if="action === 'approve'"
        :url
      />
      <Fail v-else />
    </template>
  </div>
</template>

<script setup lang="ts">
import Success from './components/Success.vue';
import Fail from './components/Fail.vue';

const route = useRoute();

const action = ref('');
const url = ref('');

watch(
  () => route.query,
  async () => {
    if (route.query?.action && (['approve', 'deny'].includes(route.query.action as string))) {
      action.value = route.query.action as string;

      if (route.query?.redirect) {
        url.value = route.query.redirect as string;
        window.location.replace(url.value);
      }
    }
  },
  {
    immediate: true,
    deep: true,
  },
);

</script>

<style scoped>
.page-wrapper {
  display: flex;
  height: calc(100vh - 48px);
  justify-content: center;
  align-items: center;
}
</style>
