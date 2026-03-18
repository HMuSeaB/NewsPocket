# NewsPocket v2.0.3

## Summary

This patch release focuses on configuration safety, JSON API correctness, and email compatibility.

## Included fixes

- Preserve `summary_field` when editing JSON API sources in the GUI.
- Add a dedicated GUI field for editing `summary_field`.
- Respect `hours_lookback` for JSON API items that provide a timestamp.
- Keep timestamp-less items included as before.
- Preserve explicit `POST` requests for JSON API sources even when the body is empty.
- Encode non-ASCII email subjects to reduce Chinese subject garbling in mail clients.
- Fix Wails asset embedding so the GUI package can pass build/test again.

## Verification

- `go test ./...`
- `npm run build` in `cmd/newspocket-gui/frontend`

## Suggested tag

- `v2.0.3`
