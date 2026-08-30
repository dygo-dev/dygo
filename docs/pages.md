# Page Metadata

Pages are app-owned metadata bundles that give Studio a stable entry point for a framework renderer. A Page does not contain UI code.

## Bundle shape

Each Page uses a kebab-case directory and the matching `<page>.page.yml` file:

```txt
apps/<app>/pages/<page>/<page>.page.yml
```

The file shape for the first renderer is:

```yaml
label: Home
description: Start page for the entities available to the current Studio user.
icon: house
route:
  path: /
renderer: entity-index
options: {}
```

`path` is `/` or one kebab-case root path such as `/pipeline`. `renderer` is currently `entity-index`. `options` is renderer-specific metadata and must be a YAML map.

Page identity is app-scoped. The bundle directory is the Page key. The runtime name is `<app>.<key>`.

Page metadata is loaded with Entity metadata. Route ownership is checked with framework-reserved and Entity routes, so two Pages or a Page and Entity cannot claim the same public path.

The public Go contract is in `pkg/dygo`. Page rendering and runtime registration remain framework-owned; app code does not register arbitrary renderers.
