# OpenDomus back office

Admin app for OpenDomus, built with Vue 3, Quasar and Vue Flow. It talks to the
backend REST API with a JWT obtained at login.

## Features

- **Devices** – list, add and edit devices (type, URL, credentials, polling).
- **Tags** – list and edit tags; **Tag values** page to browse the float history of a tag.
- **Automations** – two lists (time-triggered and value-triggered) and a drag-and-drop
  editor (`/automations/:id`) where triggers, gates and actions are wired as a node graph.
- **Logs** – paginated view of the backend log table.
- **Server switch** – the login page lets you pick the development or production API.

## Setup

Server URLs are read at runtime from `public/config.json`:

```sh
cp public/config.example.json public/config.json   # set serverUrl / serverUrlProd
npm install
npm run dev        # dev server, opens the browser
npm run build      # production build in dist/spa/
npm run lint
npm run format
```

The production build uses relative asset paths, so `dist/spa/` can be served from any
prefix (the Docker image serves it at `/bo/`).
