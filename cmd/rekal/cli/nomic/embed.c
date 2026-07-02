//go:build (darwin && arm64) || (linux && amd64) || (linux && arm64)

#include "embed.h"
#include "llama.h"
#include "ggml.h"

#include <stdlib.h>
#include <string.h>
#include <math.h>
#include <stdio.h>
#include <unistd.h>
#include <fcntl.h>

#define MAX_TOKENS 2048

struct nomic_embedder {
    struct llama_model   *model;
    struct llama_context *ctx;
    const struct llama_vocab *vocab;
    int n_embd;
    int n_threads;
    llama_token *token_buf;
};

// Redirect stderr to /dev/null, return saved fd. Returns -1 on failure.
static int suppress_stderr(void) {
    fflush(stderr);
    int saved = dup(STDERR_FILENO);
    if (saved < 0) return -1;
    int devnull = open("/dev/null", O_WRONLY);
    if (devnull < 0) { close(saved); return -1; }
    dup2(devnull, STDERR_FILENO);
    close(devnull);
    return saved;
}

// Restore stderr from saved fd.
static void restore_stderr(int saved) {
    if (saved < 0) return;
    fflush(stderr);
    dup2(saved, STDERR_FILENO);
    close(saved);
}

nomic_embedder* nomic_embedder_load(const char* model_path, int n_threads) {
    // Suppress all llama.cpp / ggml log output.
    int saved_stderr = suppress_stderr();
    llama_backend_init();
    llama_log_set(NULL, NULL);
    ggml_log_set(NULL, NULL);

    struct llama_model_params mparams = llama_model_default_params();
    mparams.n_gpu_layers = 99;

    struct llama_model *model = llama_model_load_from_file(model_path, mparams);
    if (!model) {
        restore_stderr(saved_stderr);
        return NULL;
    }

    struct llama_context_params cparams = llama_context_default_params();
    cparams.n_ctx    = MAX_TOKENS;
    cparams.n_batch  = MAX_TOKENS;
    cparams.n_ubatch = MAX_TOKENS;
    cparams.embeddings = true;
    cparams.n_threads = n_threads;
    cparams.n_threads_batch = n_threads;
    cparams.pooling_type = LLAMA_POOLING_TYPE_MEAN;

    struct llama_context *ctx = llama_init_from_model(model, cparams);
    restore_stderr(saved_stderr);

    if (!ctx) {
        llama_model_free(model);
        return NULL;
    }

    llama_token *token_buf = (llama_token*)malloc(sizeof(llama_token) * MAX_TOKENS);
    if (!token_buf) {
        llama_free(ctx);
        llama_model_free(model);
        return NULL;
    }

    nomic_embedder *e = (nomic_embedder*)malloc(sizeof(nomic_embedder));
    e->model     = model;
    e->ctx       = ctx;
    e->vocab     = llama_model_get_vocab(model);
    e->n_embd    = llama_model_n_embd(model);
    e->n_threads = n_threads;
    e->token_buf = token_buf;
    return e;
}

void nomic_embedder_free(nomic_embedder* e) {
    if (!e) return;
    int saved_stderr = suppress_stderr();
    if (e->token_buf) free(e->token_buf);
    if (e->ctx)   llama_free(e->ctx);
    if (e->model) llama_model_free(e->model);
    restore_stderr(saved_stderr);
    free(e);
}

int nomic_embedder_n_embd(nomic_embedder* e) {
    if (!e) return 0;
    return e->n_embd;
}

static void normalize(float *vec, int n) {
    float sum = 0.0f;
    for (int i = 0; i < n; i++) {
        sum += vec[i] * vec[i];
    }
    if (sum > 0.0f) {
        float norm = sqrtf(sum);
        for (int i = 0; i < n; i++) {
            vec[i] /= norm;
        }
    }
}

int nomic_embedder_embed(nomic_embedder* e, const char* text, float* out_embedding) {
    if (!e || !text || !out_embedding) return -1;

    int saved_stderr = suppress_stderr();
    int text_len = (int)strlen(text);

    // Tokenize. If text exceeds MAX_TOKENS, allocate a temp buffer and truncate.
    int n_tokens = llama_tokenize(e->vocab, text, text_len,
                                  e->token_buf, MAX_TOKENS,
                                  true, false);
    if (n_tokens < 0) {
        int required = -n_tokens;
        llama_token *tmp = (llama_token*)malloc(sizeof(llama_token) * required);
        if (!tmp) { restore_stderr(saved_stderr); return -1; }

        int actual = llama_tokenize(e->vocab, text, text_len,
                                    tmp, required, true, false);
        if (actual < 0) {
            free(tmp);
            restore_stderr(saved_stderr);
            return -1;
        }

        n_tokens = actual < MAX_TOKENS ? actual : MAX_TOKENS;
        memcpy(e->token_buf, tmp, sizeof(llama_token) * n_tokens);
        free(tmp);
    }

    if (n_tokens == 0) { restore_stderr(saved_stderr); return -1; }

    // Create batch.
    struct llama_batch batch = llama_batch_init(n_tokens, 0, 1);
    for (int i = 0; i < n_tokens; i++) {
        batch.token[i]    = e->token_buf[i];
        batch.pos[i]      = i;
        batch.n_seq_id[i] = 1;
        batch.seq_id[i][0] = 0;
        batch.logits[i]   = true;
    }
    batch.n_tokens = n_tokens;

    // Encode.
    int rc = llama_encode(e->ctx, batch);
    if (rc != 0) {
        llama_batch_free(batch);
        restore_stderr(saved_stderr);
        return -1;
    }

    // Get pooled embeddings (sequence 0).
    float *emb = llama_get_embeddings_seq(e->ctx, 0);
    if (!emb) {
        emb = llama_get_embeddings_ith(e->ctx, 0);
    }
    if (!emb) {
        llama_batch_free(batch);
        restore_stderr(saved_stderr);
        return -1;
    }

    memcpy(out_embedding, emb, sizeof(float) * e->n_embd);
    normalize(out_embedding, e->n_embd);

    llama_batch_free(batch);
    restore_stderr(saved_stderr);
    return 0;
}
