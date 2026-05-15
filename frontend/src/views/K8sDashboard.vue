<template>
  <div style="height: calc(100vh - 130px)">
    <iframe
      :src="dashboardUrl"
      style="width: 100%; height: 100%; border: none; border-radius: 4px"
      allow="clipboard-read; clipboard-write"
    />
  </div>
</template>

<script setup>
import { computed } from 'vue'

const dashboardUrl = computed(() => '/dashboard/')
</script>
