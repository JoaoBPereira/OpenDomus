<script setup lang="ts">
  import { onMounted, ref } from 'vue';

  import { getLatestTagValueLocation, getTagValueLocation } from '../../services/api';
  import { formatDateTime } from '../../utils';

  const props = defineProps<{
    tagId: number;
    // The period of the route (ISO timestamps).
    from: string;
    to: string;
    // Pin the latest known position (the last 24 hours) instead of the last point of the period (a past day).
    showLatest: boolean;
    // Battery readings of the same period, for the marker popups.
    battery: { value: number; date: Date }[];
  }>();

  // The popup text of a point: its time and the battery reading closest to it.
  const describe = (time: string) => {
    const at = new Date(time);
    const nearest = props.battery.reduce<{ value: number; date: Date } | undefined>(
      (best, reading) => (!best || Math.abs(reading.date.getTime() - at.getTime()) < Math.abs(best.date.getTime() - at.getTime()) ? reading : best),
      undefined,
    );
    const battery = nearest ? `${Math.round(nearest.value)}%` : 'unknown';
    return `${formatDateTime(at)}\nBattery: ${battery}`;
  };

  const mapElement = ref<HTMLElement>();
  const embedUrl = ref<string>();
  const message = ref('Loading...');

  // Adds the Maps JavaScript API script once. With loading=async the API is ready when it calls
  // the callback, not when the script has loaded.
  const loadMaps = (key: string) =>
    (window as any).google?.maps?.importLibrary
      ? Promise.resolve()
      : new Promise<void>((resolve, reject) => {
          (window as any).googleMapsReady = () => resolve();
          const script = document.createElement('script');
          script.src = `https://maps.googleapis.com/maps/api/js?key=${encodeURIComponent(key)}&loading=async&callback=googleMapsReady`;
          script.onerror = () => reject(new Error('Google Maps failed to load'));
          document.head.append(script);
        });

  onMounted(async () => {
    try {
      const key = Config.googleMapsApiKey;

      const [latest, history] = await Promise.all([getLatestTagValueLocation(props.tagId), key ? getTagValueLocation(props.tagId, props.from, props.to) : []]);

      // The server answers "0" instead of an empty list.
      const points = Array.isArray(history) ? history : [];
      // The server answers latitude 0, longitude 0 when the tag has no location yet.
      const known = latest.latitude || latest.longitude ? latest : undefined;
      const pin = props.showLatest || !key ? known : points[points.length - 1];
      if (!pin) {
        message.value = 'No location available.';
        return;
      }

      // Without an API key: the keyless embed, which shows only the latest position.
      if (!key) {
        embedUrl.value = `https://www.google.com/maps?q=${pin.latitude},${pin.longitude}&z=15&t=h&output=embed`;
        return;
      }

      await loadMaps(key);

      const maps = (window as any).google.maps;
      const { Map, Polyline, InfoWindow } = await maps.importLibrary('maps');
      const { Marker } = await maps.importLibrary('marker');
      const { LatLngBounds, SymbolPath } = await maps.importLibrary('core');

      const position = { lat: pin.latitude, lng: pin.longitude };
      const path = points.map((point) => ({ lat: point.latitude, lng: point.longitude }));

      // maxZoom keeps fitBounds from zooming all the way in when the car has not moved.
      const map = new Map(mapElement.value, { center: position, zoom: 15, maxZoom: 18, mapTypeId: 'hybrid' });

      // One popup shared by all markers: clicking a marker shows its time and battery.
      const popup = new InfoWindow();
      const addMarker = (options: object, time: string) => {
        const marker = new Marker({ map, ...options });
        marker.addListener('click', () => {
          const content = document.createElement('div');
          content.style.cssText = 'color: #000000; white-space: pre-line';
          content.textContent = describe(time);
          popup.setContent(content);
          popup.open({ map, anchor: marker });
        });
      };

      const dot = { path: SymbolPath.CIRCLE, scale: 5, fillColor: '#2563eb', fillOpacity: 1, strokeColor: '#ffffff', strokeWeight: 1 };
      points.forEach((point, index) => addMarker({ position: path[index], icon: dot }, point.time));
      // The pin, above the dots.
      addMarker({ position, zIndex: points.length + 1 }, pin.time);

      if (path.length > 1) {
        new Polyline({ map, path, strokeColor: '#2563eb', strokeWeight: 4 });
        const bounds = new LatLngBounds();
        [...path, position].forEach((point) => bounds.extend(point));
        map.fitBounds(bounds);
      }
      message.value = '';
    } catch (error) {
      console.error('Error loading the location map', error);
      message.value = 'Error loading the map.';
    }
  });
</script>

<template>
  <div class="locationMap">
    <iframe v-if="embedUrl" :src="embedUrl" title="Location on Google Maps" />
    <template v-else>
      <div ref="mapElement" class="map" />
      <p v-if="message">{{ message }}</p>
    </template>
  </div>
</template>

<style lang="scss" scoped>
  .locationMap {
    position: relative;

    iframe,
    .map {
      position: absolute;
      inset: 0;
      width: 100%;
      height: 100%;
      border: 0;
    }

    p {
      position: relative;
      margin: 0;
    }
  }
</style>
