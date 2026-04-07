<script setup lang="ts">
import {OnyxIconButton} from "sit-onyx";
import {iconUserEdit, iconTrash, iconTrashDisabled} from "@sit-onyx/icons";
import {user} from "../auth/user.ts";

const props = defineProps<{
  id: string;
}>();

const emit = defineEmits<{
  edit: [id: string],
  delete: [id: string],
}>();


</script>

<template>
  <div>
    <OnyxIconButton
        :icon="iconUserEdit"
        label="Edit"
        density="compact"
        @click="emit('edit', props.id)"
        color="neutral"/>
    <OnyxIconButton
        :icon="props.id !== user.id ? iconTrash : iconTrashDisabled"
        :disabled="props.id === user.id"
        label="Delete"
        density="compact"
        @click="emit('delete', props.id)"
        color="neutral"/>
  </div>
</template>

<style scoped>
div {
  display: flex;
  gap: .5rem;
}
</style>