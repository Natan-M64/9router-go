### fix(catalog): remove the NVIDIA models NVIDIA has retired and the non-chat ones

- Six models answer `410 Gone` from NVIDIA (end of life): `minimaxai/minimax-m2.7`, `minimaxai/minimax-m3`, `z-ai/glm-5.2`, `deepseek-ai/deepseek-v4-pro`, `deepseek-ai/deepseek-v4-flash`, `moonshotai/kimi-k2.6`. They are removed from the catalog, and so is `nvidia/nemotron-3-super-120b-a12b` from the free-model suggestions.
- Eleven entries are not chat models (embeddings, ASR, TTS and `nvclip`) and answer `404` on the chat route. They are removed from the chat catalog; the type table still classifies them.
- `nvidia/nemotron-3-ultra-550b-a55b` answers `401` without a key, so the model exists and stays.
