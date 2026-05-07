# mediasink — public HEP media export API

This package defines `MediaExporter` / `MediaConfig` and the default implementation
in `open.go`: **raw binary RTCP SR** and **Homer-Lake JSON** on HEP type 5.

## Optional `MediaExporter` extension

The built-in exporter only handles **HEP type 5** in the modes above. You can link a
**separate Go module** at compile time that registers a factory for other reporting
shapes (whatever your extension encodes):

1. Implement `mediasink.MediaExporter` for your encoder.
2. In `init()`, call:

```go
mediasink.RegisterMediaExporterExtension(myFactory)
```

3. Blank-import that module from your `main` (enough if `init` registers the factory).

`myFactory` should return `(nil, nil)` when `cfg` selects a built-in mode (Homer-Lake or raw periodic)
so that `NewMediaExporter` falls through to the OSS implementation.

Pass `nil` to `RegisterMediaExporterExtension` to clear the factory (e.g. in tests).

CLI validation: `-send_media_report=true` with both `-hep_raw_rtcp=false` and `-hep_homer_lake_rtcp=false`
is rejected unless `MediaExporterExtensionRegistered()` is true at parse time (i.e. an extension was linked).
