
## MEM-001: Discover binary CUE sheets without loading unrelated FLAC artwork

2026-11-06: Production profiling attributed about 66% of cumulative allocation to pictures read by embedded-CUE discovery. A dedicated STREAMINFO/CUESHEET reader seeks over other payloads, validates block boundaries, and leaves normal metadata/artwork APIs unchanged. A read-prohibiting fixture fails with the full reader; it covers artwork before/after CUE, absent CUE and truncated skipped blocks. An 8 MiB picture benchmark drops from about 8 MiB to 1152 B/op. Full library tests pass. Read-only NAS comparison of 32 real files preserves CUE/sample-rate results and reduces allocations from 8,149,632 to 19,648 bytes. Seen 1×.

中文：扫描内嵌 CUE 应只读取 STREAMINFO/CUESHEET，以 seek 跳过封面和评论块，保留正常标签/封面接口。回归明确禁止读取无关载荷，并检查缺失 CUE、块顺序和截断；真实 NAS 文件对比不修改音频或曲库。Seen 1×（2026-11-06）。

## WAV-001: RIFF chunk identifiers differ between taggers

2026-11-01: A WAV contained placeholder question marks in LIST/INFO and valid Unicode title/artist/album in a lowercase `id3 ` chunk. Reading only uppercase `ID3 ` silently discarded that valid metadata. Accept both established forms. A synthetic dual-tag fixture fails for lowercase before the fix and both forms pass afterward; the library suite passes. No audio payload or file tags are rewritten. Seen 1×.

中文：同一 WAV 可同时有占位 RIFF INFO 和正确 ID3；读取器必须兼容 `ID3 ` 与 `id3 `。用双标签样本验证优先读取正确 ID3，不靠改写文件掩盖读取缺陷。Seen 1×（2026-11-01）。
