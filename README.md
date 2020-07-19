# srtfan

`srtfan` takes a single "src" SRT stream input and fans it out to multiple "sink" outputs. The source and sinks may be disconnected and connected dynamically.

```
Usage of srtfan:
  -sink-address string
        Address to listen for sink connections (default ":5001")
  -sink-latency int
        SRT sink latency parameter (default 500)
  -sink-passphrase string
        SRT sink passphrase parameter
  -src-address string
        Address to listen for source connection (default ":5000")
  -src-latency int
        SRT source latency parameter (default 500)
  -src-passphrase string
        SRT source passphrase parameter
```
