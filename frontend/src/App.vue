<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useNetworkStore } from './stores/network'

const { isOffline } = storeToRefs(useNetworkStore())
</script>

<template>
  <div v-if="isOffline" class="offline-banner">
    Офлайн — доступен только просмотр
  </div>
  <router-view />
</template>

<style scoped>
.offline-banner {
  position: sticky;
  top: 0;
  z-index: 100;
  padding: 0.5rem 1rem;
  text-align: center;
  font-size: 0.875rem;
  background: #fef3c7;
  color: #92400e;
}
</style>
