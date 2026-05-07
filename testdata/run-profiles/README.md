# HEP script run profiles (`hep-scripts.json`)

JSON mirror of [`scripts/hep-uas-listen.sh`](../../scripts/hep-uas-listen.sh) and [`scripts/hep-uac-send.sh`](../../scripts/hep-uac-send.sh): see [`hep-scripts.json`](hep-scripts.json).

| Alias | Matches |
|-------|---------|
| `hep-uas-listen` | UAS: `uas_pcap.xml`, bind `0.0.0.0:9050`, `-m 0 -r 1 -l 2048`, capture id **2001**. Set **`-hep_addr`** on the CLI (same as `HEP_ADDR` in the script). |
| `hep-uac-send` | UAC defaults: `uac_rtp_raw.xml`, `UAS_ADDR` **127.0.0.1:9050**, `-m 0 -r 2 -l 32`, capture **3001**, `-send_media_report=true -hep_homer_lake_rtcp=true` (`HEP_HOMER_LAKE=1`). |
| `hep-uac-send-raw-rtcp` | Same as UAC script with `HEP_HOMER_LAKE=0` and `HEP_RAW_RTCP=1` (binary RTCP on HEP type 5). |
| `hep-uac-send-short-json` | Same as UAC script with `HEP_HOMER_LAKE=0` and `HEP_RAW_RTCP=0` — requires a linked **mediasink** extension (e.g. gossipper-hepic). |

**Not in JSON** (pass on the command line like the scripts’ env):

- `-hep_addr` / `HEP_ADDR` (required for HEP).
- `-hep_password` / `HEP_PASSWORD`.
- OTLP: `-log_otel_endpoint`, `-log_otel_proto`, `-log_otel_insecure`.

**Examples**

```bash
export HEP_ADDR=collector.example.com:9060

gossipper -config testdata/run-profiles/hep-scripts.json -run-alias=hep-uas-listen -hep_addr "$HEP_ADDR"

gossipper -config testdata/run-profiles/hep-scripts.json -run-alias=hep-uac-send -hep_addr "$HEP_ADDR"
```

Override UAS address or rate like the shell env:

```bash
gossipper -config testdata/run-profiles/hep-scripts.json -run-alias=hep-uac-send -hep_addr "$HEP_ADDR" -rsa 192.168.1.10:9050 -r 5
```
