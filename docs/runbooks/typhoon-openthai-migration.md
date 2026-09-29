# Typhoon OCR 1.5 → OpenThai 2.0: approved 48 GB pilot

Date: 2026-09-29. Status: user approved 48 GB at $1.22/hour. Endpoint `0oemotfk966zij` was saved and its overview confirmed 48 GB, GPU count 1, Active Workers 0, Max Workers 1, idle timeout 5 seconds. FlashBoot Standard was enabled in the editor. No requests were running or queued at verification. The image is still PaddleOCR `98f8fd49c77f9d6617b07a91b1664c6d545f3fea`; the model migration has not been deployed or benchmarked.

The user requested replacement of PaddleOCR with a two-stage pipeline, self-hosted on RunPod, and explicitly requested hardware sizing and cost before changing GPU.

## Proposed pipeline

1. Typhoon OCR 1.5 (`typhoon-ai/typhoon-ocr1.5-2b`) transcribes each page using the publisher's fixed OCR prompt into Markdown/HTML tables.
2. OpenThai 2.0 (`iapp/openthai2.0-qwen3.8-27b`, official Q4_K_M GGUF) receives the transcription and proposes receipt/invoice fields as JSON.
3. PlaiFlow validates JSON, field evidence, money/date formats and tenant/lease provenance; missing or unsupported values remain null with warnings. Human review remains required.

Use one Serverless Flex worker, process one document at a time, and retain both models between requests. The existing scoped input tokens, encrypted source storage, job leases and publication checks remain the transport. There is no call to a hosted Typhoon or iApp inference API.

## Hardware and price proposal

Recommended starting point: **48 GB A6000/A40 tier**, OpenThai Q4_K_M and Typhoon BF16. The OpenThai file is about 17 GB; Typhoon's approximately 2B BF16 parameters imply roughly 4 GB of weights. These are weight sizes, not measured peak VRAM. The remaining space is intended for runtime, vision activations and context caches; verify peak memory under a bounded single-document load before accepting the configuration.

| Serverless tier | Advertised compute rate | Intended use |
| --- | ---: | --- |
| 48 GB A6000/A40 | $1.22/hour | Recommended initial pilot with OpenThai Q4_K_M |
| 48 GB PRO | $1.75/hour | Same quantized model if measured latency warrants faster hardware |
| 80 GB H100 PRO | $4.79/hour | Full BF16 OpenThai plus Typhoon; larger memory margin |

Rates were checked on the RunPod public Serverless pricing page and account endpoint editor. They are rates while billable, not a fixed monthly subscription. Startup, inference and the idle timeout are billable; container storage is additional. Example only: 20 billable seconds at $1.22/hour costs about $0.0068 compute, or $6.78 per 1,000 such requests. This is **not** a measured latency or cost/page estimate.

Approved pilot controls: Active Workers 0, Max Workers 1, FlashBoot Standard enabled, idle timeout 5 seconds, existing $10 aggregate pilot budget with stop at $8 retained. Hardware approval was received and applied. Initial validation is limited to the previously authorized FlowAccount sample, not the cancelled 500-page benchmark.

## Required implementation and verification

- Introduce a versioned OCR artifact that supports transcription without measured boxes or confidence. The current v2 contract requires Paddle-style geometry; do not fabricate polygons or confidence to satisfy it.
- Preserve raw Typhoon output separately from OpenThai suggestions, pin model/runtime revisions, and keep previously stored Paddle artifacts readable.
- Treat OCR text as untrusted data, give the extraction model no tools, enforce a bounded JSON schema, and reject citations that do not exist in the transcription. Render model output as escaped text rather than unsanitized HTML.
- Validate full-page transcription and extraction separately. Use null/warnings for ambiguity, and independently check arithmetic and identifiers.
- Measure cold start, resumed-worker latency, each model's execution, total processing, peak VRAM and billed cost using the same authorized sample. A two-model generative pipeline is not assumed faster than the existing 18.941-second result.
- OpenThai's model card recommends thinking enabled for text-only requests and warns that short generation caps can yield empty answers. Bound context/output and handle incomplete generation explicitly; do not claim image-only no-thinking settings are validated for this text-only stage.

## Sources

- [Typhoon OCR documentation](https://docs.opentyphoon.ai/en/ocr/)
- [Typhoon OCR 1.5 model card](https://huggingface.co/typhoon-ai/typhoon-ocr1.5-2b)
- [OpenThai 2.0 model card](https://huggingface.co/iapp/openthai2.0-qwen3.8-27b)
- [Official OpenThai GGUF weights](https://huggingface.co/iapp/openthai2.0-qwen3.8-27b-GGUF)
- [RunPod Serverless prices](https://www.runpod.io/pricing)
- [RunPod billing scope](https://docs.runpod.io/serverless/pricing)
