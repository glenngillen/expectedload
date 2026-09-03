# expected-load:
#   monthly_calls: 250_000
#   avg_output_tokens: 800
#   last_updated: 2026-08-01
def serve(request):
    return request


def embed(batch):
    """Generates embeddings for a document batch.

    Expected load:
        monthly_calls: 40_000
        avg_input_tokens: 6_000
    """
    return batch
