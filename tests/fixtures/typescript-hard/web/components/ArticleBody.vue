<script setup lang="ts">
import { computed } from "vue";
import DOMPurify from "dompurify";

const props = defineProps<{ html: string }>();
const safeHtml = computed(() => DOMPurify.sanitize(props.html));
</script>

<template>
  <div class="article" v-html="safeHtml"></div>
</template>
