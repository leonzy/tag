
## WAV-001: RIFF chunk identifiers differ between taggers

2026-11-01: A WAV contained placeholder question marks in LIST/INFO and valid Unicode title/artist/album in a lowercase `id3 ` chunk. Reading only uppercase `ID3 ` silently discarded that valid metadata. Accept both established forms. A synthetic dual-tag fixture fails for lowercase before the fix and both forms pass afterward; the library suite passes. No audio payload or file tags are rewritten. Seen 1×.

中文：同一 WAV 可同时有占位 RIFF INFO 和正确 ID3；读取器必须兼容 `ID3 ` 与 `id3 `。用双标签样本验证优先读取正确 ID3，不靠改写文件掩盖读取缺陷。Seen 1×（2026-11-01）。
