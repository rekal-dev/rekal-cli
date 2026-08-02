#ifndef NOMIC_EMBED_H
#define NOMIC_EMBED_H

#ifdef __cplusplus
extern "C" {
#endif

// Let llama.cpp's own diagnostics through instead of discarding them.
//
// Loading and inference are normally wrapped in a stderr redirect to /dev/null,
// because llama.cpp is chatty and a CLI must stay quiet. But that also swallows
// the message when it *fails* — a native crash or a load error then leaves no
// trace anywhere, and the caller can only report a timeout. The daemon writes
// its stderr to a log file rather than a terminal, so it turns this on.
void nomic_set_verbose(int on);

// Opaque handle to a loaded embedding model.
typedef struct nomic_embedder nomic_embedder;

// Load a GGUF model for embedding. Returns NULL on failure.
// n_threads: number of CPU threads to use.
nomic_embedder* nomic_embedder_load(const char* model_path, int n_threads);

// Free the embedder and all associated resources.
void nomic_embedder_free(nomic_embedder* e);

// Get the embedding dimension.
int nomic_embedder_n_embd(nomic_embedder* e);

// Embed a text string. Writes n_embd floats to out_embedding.
// Returns 0 on success, non-zero on failure.
int nomic_embedder_embed(nomic_embedder* e, const char* text, float* out_embedding);

#ifdef __cplusplus
}
#endif

#endif // NOMIC_EMBED_H
