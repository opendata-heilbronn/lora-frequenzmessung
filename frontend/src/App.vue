<template>
  <OnyxAppLayout class="onyx-grid-max-md onyx-grid-center">
    <template #navBar>
      <OnyxNavBar app-name="LoRa Sensor Management">
        <OnyxNavItem label="Sensors" link="/" />
        <OnyxNavItem label="+ Add Sensor" link="/add" />
      </OnyxNavBar>
    </template>
    <OnyxPageLayout>
      <router-view />
    </OnyxPageLayout>
  </OnyxAppLayout>
</template>

<script setup lang="ts">
import { OnyxAppLayout, OnyxNavBar, OnyxPageLayout } from 'sit-onyx'
import { OnyxNavItem } from 'sit-onyx'
</script>
